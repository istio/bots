// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Take in a pr number from blob storage and examines the pr
// for all tests that are run and their results. The results are then written to storage.

package resultgatherer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"

	"cloud.google.com/go/storage"

	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"istio.io/bots/policybot/pkg/blobstorage"
	store "istio.io/bots/policybot/pkg/storage"
)

// Pull struct for the structure under refs/pulls in clone-records.json
type pull struct {
	Number int
	Author string
	Sha    string
}

// Cmd struct for Commands object under clone-records.json
type cmnd struct {
	Command string
	Output  string
}

// Finished struct to store values for all fields in Finished.json
type finished struct {
	Timestamp int64
	Passed    bool
	Result    string
}

// Clone_Record struct to store values for all fields in clone-records.json
type cloneRecord struct {
	Refs struct {
		Org       string
		Repo      string
		BaseRef   string `json:"base_ref"`
		BaseSha   string `json:"base_sha"`
		Pulls     []pull
		PathAlias string
	}
	Commands []cmnd
	Failed   bool
}

// Started struct to store values from started.json
type started struct {
	Timestamp int64
}

type TestResultGatherer struct {
	Client           blobstorage.Store
	BucketName       string
	PreSubmitPrefix  string
	PostSubmitPrefix string
}

func (trg *TestResultGatherer) getRepoPrPath(orgLogin string, repoName string) string {
	return trg.PreSubmitPrefix + orgLogin + "_" + repoName + "/"
}

func (trg *TestResultGatherer) getTestsForPR(ctx context.Context, orgLogin string, repoName string, prNumInt int64) (map[string][]string, error) {
	prNum := strconv.FormatInt(prNumInt, 10)
	prefixForPr := trg.getRepoPrPath(orgLogin, repoName) + prNum + "/"
	return trg.getTests(ctx, prefixForPr)
}

func (trg *TestResultGatherer) getBucket() blobstorage.Bucket {
	return trg.Client.Bucket(trg.BucketName)
}

// GetTest given a gcs path that contains test results in the format [testname]/[runnumber]/[resultfiles], return a map of testname to []runnumber
// Client: client used to get buckets and objects.
// PrNum: the PR number inputted.
// Return []Tests return a slice of Tests objects.
func (trg *TestResultGatherer) getTests(ctx context.Context, pathPrefix string) (map[string][]string, error) {
	bucket := trg.getBucket()
	testNames, err := bucket.ListPrefixes(ctx, pathPrefix)
	if err != nil {
		return nil, err
	}
	testMap := map[string][]string{}
	var runs []string
	for _, testPref := range testNames {
		testPrefSplit := strings.Split(testPref, "/")
		testname := testPrefSplit[len(testPrefSplit)-2]
		runs, err = bucket.ListPrefixes(ctx, testPref)
		if err != nil {
			return nil, err
		}
		var runPaths []string
		var ok bool
		if runPaths, ok = testMap[testname]; !ok {
			runPaths = []string{}
		}
		for _, runPath := range runs {
			if runPath != "" {
				runPaths = append(runPaths, runPath)
			}
		}
		testMap[testname] = runPaths
	}
	return testMap, nil
}

func (trg *TestResultGatherer) getInformationFromFinishedFile(ctx context.Context, pref string) (*finished, error) {
	bucket := trg.getBucket()
	nrdr, err := bucket.Reader(ctx, pref+"finished.json")
	var finish finished

	if err != nil {
		return nil, err
	}

	defer nrdr.Close()
	finishFile, err := ioutil.ReadAll(nrdr)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(finishFile, &finish); err != nil {
		return nil, err
	}
	return &finish, nil
}

func (trg *TestResultGatherer) getInformationFromStartedFile(ctx context.Context, pref string) (*started, error) {
	bucket := trg.getBucket()
	nrdr, err := bucket.Reader(ctx, pref+"started.json")
	if err != nil {
		return nil, err
	}

	defer nrdr.Close()
	startFile, nerr := ioutil.ReadAll(nrdr)
	if nerr != nil {
		return nil, nerr
	}

	var started started

	if err := json.Unmarshal(startFile, &started); err != nil {
		return nil, err
	}
	return &started, nil
}

func (trg *TestResultGatherer) getInformationFromCloneFile(ctx context.Context, pref string) ([]*cloneRecord, error) {
	bucket := trg.getBucket()
	rdr, err := bucket.Reader(ctx, pref+"clone-records.json")
	if err != nil {
		return nil, err
	}

	defer rdr.Close()
	cloneFile, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}

	var records []*cloneRecord

	if err = json.Unmarshal(cloneFile, &records); err != nil {
		return nil, err
	}

	return records, nil
}

var knownSignatures map[string]map[string]string

// = {
// 	"build-log.txt": {
// 		"error parsing HTTP 408 response body": "",
// 		"failed to get a Boskos resource": "",
// 		"recipe for target '.*docker.*' failed": "",
// 		"Entrypoint received interrupt: terminated": "",
// 		"release istio failed: Service \"istio-ingressgateway\" is invalid: spec\\.ports\\[\\d\\]\\.nodePort\\: Invalid value\\:": "",
// 		"The connection to the server \\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\ was refused - did you specify the right host or port\\?": "",
// 		"gzip: stdin: unexpected end of file": "",
// 		"Process did not finish before": "",
// 		"No cluster named ": "boskos refers to non-existent cluster or project"
// 		"API Server failed to come up": "",
// 	}
// }

func (trg *TestResultGatherer) getEnvironmentalSignatures(ctx context.Context, testRun string) (result []string) {
	bucket := trg.getBucket()
	for filename, sigmap := range knownSignatures {
		r, err := bucket.Reader(ctx, testRun+filename)
		if err != nil {
			log.Fatal("foo")
		}
		signatures := []string{}
		names := []string{}
		for signature, name := range sigmap {
			signatures = append(signatures, signature)
			names = append(names, name)
		}
		foo := getSignature(r, signatures)
		result = append(result, names[foo])
	}
	return
}

func (trg *TestResultGatherer) getTestRunArtifacts(ctx context.Context, testRun string) ([]string, error) {
	artifacts, err := trg.getBucket().ListItems(ctx, testRun+"artifacts/")
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

// getManyResults function return the status of test passing, clone failure, sha number, base sha for each test
// run under each test suite for the given pr.
// Client: client used to get buckets and objects from google cloud storage.
// TestSlice: a slice of Tests objects containing all tests and the path to folder for each test run for the test under such pr.
// Return a map of test suite name -- pr number -- run number -- FortestResult objects.
func (trg *TestResultGatherer) getManyResults(ctx context.Context, testSlice map[string][]string,
	orgLogin string, repoName string) ([]*store.TestResult, error) {

	var allTestRuns = []*store.TestResult{}

	for testName, runPaths := range testSlice {
		for _, runPath := range runPaths {
			if testResult, err := trg.getTestResult(ctx, testName, runPath); err == nil {
				testResult.OrgLogin = orgLogin
				testResult.RepoName = repoName
				allTestRuns = append(allTestRuns, testResult)
			} else {
				return nil, err
			}
		}
	}
	return allTestRuns, nil
}

func (trg *TestResultGatherer) getTestResult(ctx context.Context, testName string, testRun string) (testResult *store.TestResult, err error) {
	testResult = &store.TestResult{}
	testResult.TestName = testName
	testResult.RunPath = testRun
	testResult.Done = false

	records, err := trg.getInformationFromCloneFile(ctx, testRun)
	if err != nil {
		return
	}

	record := records[0]
	testResult.Sha = record.Refs.Pulls[0].Sha
	testResult.BaseSha = record.Refs.BaseSha
	testResult.CloneFailed = record.Failed

	started, err := trg.getInformationFromStartedFile(ctx, testRun)
	if err != nil {
		return
	}

	testResult.StartTime = time.Unix(started.Timestamp, 0)

	finished, err := trg.getInformationFromFinishedFile(ctx, testRun)
	if err != storage.ErrObjectNotExist {
		if err != nil {
			return
		}
		testResult.TestPassed = finished.Passed
		testResult.Result = finished.Result
		testResult.FinishTime = time.Unix(finished.Timestamp, 0)
	}

	prefSplit := strings.Split(testRun, "/")

	runNo, err := strconv.ParseInt(prefSplit[len(prefSplit)-2], 10, 64)
	if err != nil {
		return
	}
	testResult.RunNumber = runNo
	prNo, newError := strconv.ParseInt(prefSplit[len(prefSplit)-4], 10, 64)
	if newError != nil {
		return nil, newError
	}
	testResult.PullRequestNumber = prNo

	artifacts, err := trg.getTestRunArtifacts(ctx, testRun)
	if err != nil {
		return
	}
	testResult.HasArtifacts = len(artifacts) != 0
	testResult.Artifacts = artifacts

	if !testResult.TestPassed && !testResult.HasArtifacts {
		// this is almost certainly an environmental failure, check for known sigs
		testResult.Signatures = trg.getEnvironmentalSignatures(ctx, testRun)
	}
	return
}

// Read in gcs the folder of the given pr number and write the result of each test runs into a slice of TestFlake struct.
func (trg *TestResultGatherer) CheckTestResultsForPr(ctx context.Context, orgLogin string, repoName string, prNum int64) ([]*store.TestResult, error) {
	testSlice, err := trg.getTestsForPR(ctx, orgLogin, repoName, prNum)
	if err != nil {
		return nil, err
	}
	fullResult, err := trg.getManyResults(ctx, testSlice, orgLogin, repoName)

	if err != nil {
		return nil, err
	}
	return fullResult, nil
}

func (trg *TestResultGatherer) GetAllPullRequests(ctx context.Context, orgLogin string, repoName string) (prs []string, err error) {
	return trg.getBucket().ListPrefixes(ctx, trg.getRepoPrPath(orgLogin, repoName))
}

// if any pattern is found in the object, return it's index
// if no pattern is found, return -1
func getSignature(r io.Reader, patterns []string) int {
	kdk := bufio.NewReader(r)
	re := compileRegex(patterns)

	indices := re.FindReaderSubmatchIndex(kdk)

	// the array is effectively start/end tuples
	// with the first two tuple representing the whole regex
	// and outer parens.  indices[4] = pattern[0].start
	for i := 4; i < len(indices); i += 2 {
		if indices[i] > -1 {
			return (i - 4) / 2
		}
	}
	return -1
}

func compileRegex(patterns []string) *regexp.Regexp {
	s := fmt.Sprintf("((%s))", strings.Join(patterns, ")|("))
	return regexp.MustCompile(s)
}
