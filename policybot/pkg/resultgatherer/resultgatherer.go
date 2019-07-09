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
	"context"
	"encoding/json"

	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

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
	client     *storage.Client
	bucketName string
}

// Function to initionalize new TestResultGatherer object.
// Use `testResultGatherer, err := NewTestResultGatherer()` and `testFlakes := testResultGatherer.checkTestFlakesForPr(<pr_number>)`
// to get test flakes information for a given pr.
func NewTestResultGatherer(client *storage.Client, bucketName string) (*TestResultGatherer, error) {
	return &TestResultGatherer{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (trg *TestResultGatherer) query(ctx context.Context, prefix string) ([]string, error) {
	client := trg.client
	bucket := client.Bucket(trg.bucketName)
	query := &storage.Query{Prefix: prefix, Delimiter: "/"}
	it := bucket.Objects(ctx, query)
	paths := []string{}
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		paths = append(paths, attrs.Prefix)

	}
	return paths, nil
}

// GetTest function get all directories under the given pr in blob storage for each test suite name.
// Client: client used to get buckets and objects.
// PrNum: the PR number inputted.
// Return []Tests return a slice of Tests objects.
func (trg *TestResultGatherer) getTests(ctx context.Context, orgLogin string, repoName string, prNumInt int64) (map[string][]string, error) {
	prNum := strconv.FormatInt(prNumInt, 10)
	prefixForPr := "pr-logs/pull/" + orgLogin + "_" + repoName + "/" + prNum + "/"
	testNames, err := trg.query(ctx, prefixForPr)
	if err != nil {
		return nil, err
	}
	testMap := map[string][]string{}
	var runs []string
	for _, testPref := range testNames {
		testPrefSplit := strings.Split(testPref, "/")
		testname := testPrefSplit[len(testPrefSplit)-2]
		runs, err = trg.query(ctx, testPref)
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

func (trg *TestResultGatherer) getInformationFromFinishedFile(ctx context.Context, pref string, eachRun *store.TestResult) (*store.TestResult, error) {
	// It is possible that the folder might not contain finished.json.
	client := trg.client
	bucket := client.Bucket(trg.bucketName)
	newObj := bucket.Object(pref + "finished.json")
	nrdr, err := newObj.NewReader(ctx)
	var finish finished

	if err != nil {
		return eachRun, err
	}

	defer nrdr.Close()
	finishFile, err := ioutil.ReadAll(nrdr)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(finishFile, &finish); err != nil {
		return nil, err
	}

	passed := finish.Passed
	result := finish.Result
	t := finish.Timestamp
	tm := time.Unix(t, 0)

	eachRun.TestPassed = passed
	eachRun.Result = result
	eachRun.FinishTime = tm

	return eachRun, nil
}

func (trg *TestResultGatherer) getInformationFromStartedFile(ctx context.Context, pref string, eachRun *store.TestResult) (*store.TestResult, error) {
	client := trg.client
	bucket := client.Bucket(trg.bucketName)
	newObj := bucket.Object(pref + "started.json")
	nrdr, err := newObj.NewReader(ctx)
	if err != nil {
		return eachRun, err
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
	t := started.Timestamp
	tm := time.Unix(t, 0)
	eachRun.StartTime = tm

	return eachRun, nil
}

func (trg *TestResultGatherer) getInformationFromCloneFile(ctx context.Context, pref string, eachRun *store.TestResult) (*store.TestResult, error) {
	client := trg.client
	bucket := client.Bucket(trg.bucketName)
	obj := bucket.Object(pref + "clone-records.json")
	rdr, err := obj.NewReader(ctx)
	if err != nil {
		return eachRun, err
	}

	defer rdr.Close()
	cloneFile, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}

	var records []cloneRecord

	if err = json.Unmarshal(cloneFile, &records); err != nil {
		return nil, err
	}
	record := records[0]
	refs := record.Refs
	orgLogin := refs.Org
	repoName := refs.Repo
	pulls := refs.Pulls
	pull := pulls[0]
	sha := pull.Sha
	baseSha := refs.BaseSha
	failed := record.Failed

	eachRun.Sha = sha
	eachRun.BaseSha = baseSha
	eachRun.CloneFailed = failed
	eachRun.OrgLogin = orgLogin
	eachRun.RepoName = repoName

	return eachRun, nil
}

// GetShaAndPassStatus function return the status of test passing, clone failure, sha number, base sha for each test run under each test suite for the given pr.
// Client: client used to get buckets and objects from google cloud storage.
// TestSlice: a slice of Tests objects containing all tests and the path to folder for each test run for the test under such pr.
// Return a map of test suite name -- pr number -- run number -- ForEachRun objects.
func (trg *TestResultGatherer) getShaAndPassStatus(ctx context.Context, testSlice map[string][]string) ([]*store.TestResult, error) {
	var allTestRuns = []*store.TestResult{}

	for testName, runPaths := range testSlice {
		for _, runPath := range runPaths {

			var eachRun = &store.TestResult{}
			eachRun.TestName = testName
			eachRun.RunPath = runPath
			eachRun.Done = false

			var err error
			eachRun, err = trg.getInformationFromCloneFile(ctx, runPath, eachRun)
			if err != nil {
				return nil, err
			}

			eachRun, err = trg.getInformationFromStartedFile(ctx, runPath, eachRun)
			if err != nil {
				return nil, err
			}

			eachRun, err = trg.getInformationFromFinishedFile(ctx, runPath, eachRun)
			if err != nil {
				return nil, err
			}

			prefSplit := strings.Split(runPath, "/")

			runNo, err := strconv.ParseInt(prefSplit[len(prefSplit)-2], 10, 64)
			if err != nil {
				return nil, err
			}
			eachRun.RunNumber = runNo
			prNo, newError := strconv.ParseInt(prefSplit[len(prefSplit)-4], 10, 64)
			if newError != nil {
				return nil, newError
			}
			eachRun.PullRequestNumber = prNo
			allTestRuns = append(allTestRuns, eachRun)

		}
	}
	return allTestRuns, nil
}

// Read in gcs the folder of the given pr number and write the result of each test runs into a slice of TestFlake struct.
func (trg *TestResultGatherer) CheckTestResultsForPr(ctx context.Context, orgLogin string, repoName string, prNum int64) ([]*store.TestResult, error) {
	testSlice, err := trg.getTests(ctx, orgLogin, repoName, prNum)
	if err != nil {
		return nil, err
	}
	fullResult, err := trg.getShaAndPassStatus(ctx, testSlice)

	if err != nil {
		return nil, err
	}
	return fullResult, nil
}
