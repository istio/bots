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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/ghodss/yaml"

	"istio.io/bots/policybot/pkg/blobstorage"
	pipelinetwo "istio.io/bots/policybot/pkg/pipeline"
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
	FinalSha string `json:"final_sha"`
}

//TestOutcome struct to store values from yaml
//https://github.com/istio/istio/blob/77d9c1040b1a56064f7e59593f53331cca6c7578/pkg/test/framework/suitecontext.go#L232
type TestOutcome struct {
	Name          string
	Type          string
	Outcome       string
	FeatureLabels map[string][]string `yaml:"featureLabels,omitempty"`
}

//SuiteOutcome struct to store values from yaml
//https://github.com/istio/istio/blob/6c8b0942298420b94a2af47c7a7f6cd08567851e/pkg/test/framework/suite.go#L311
type SuiteOutcome struct {
	Name         string
	Environment  string
	Multicluster bool
	TestOutcomes []TestOutcome
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

func (trg *TestResultGatherer) GetTestsForPR(ctx context.Context, orgLogin string, repoName string, prNum string) (map[string][]string, error) {
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
	testNames := bucket.ListPrefixesProducer(ctx, pathPrefix).Go()
	testMap := map[string][]string{}
	for item := range testNames {
		if item.Err() != nil {
			return nil, item.Err()
		}
		testPref := item.Output()
		testPrefSplit := strings.Split(testPref.(string), "/")
		testname := testPrefSplit[len(testPrefSplit)-2]
		runs, err := bucket.ListPrefixes(ctx, testPref.(string))
		if err != nil {
			return nil, err
		}
		runPaths := testMap[testname]
		testMap[testname] = append(runPaths, runs...)
	}
	return testMap, nil
}

func (trg *TestResultGatherer) getInformationFromFinishedFile(ctx context.Context, pref string) (*finished, error) {
	bucket := trg.getBucket()
	nrdr, err := bucket.Reader(ctx, pref+"finished.json")
	var finish finished

	if err != nil {
		return nil, fmt.Errorf("error retrieving finished.json from %s: %v", pref, err)
	}

	defer nrdr.Close()
	finishFile, err := ioutil.ReadAll(nrdr)
	if err != nil {
		return nil, fmt.Errorf("error reading finished.json from %s: %v", pref, err)
	}

	if err = json.Unmarshal(finishFile, &finish); err != nil {
		return nil, fmt.Errorf("error parsing finished.json from %s: %v", pref, err)
	}
	return &finish, nil
}

func (trg *TestResultGatherer) getInformationFromStartedFile(ctx context.Context, pref string) (*started, error) {
	bucket := trg.getBucket()
	nrdr, err := bucket.Reader(ctx, pref+"started.json")
	if err != nil {
		return nil, fmt.Errorf("error retrieving started.json from %s: %v", pref, err)
	}

	defer nrdr.Close()
	startFile, nerr := ioutil.ReadAll(nrdr)
	if nerr != nil {
		return nil, fmt.Errorf("error reading started.json from %s: %v", pref, nerr)
	}

	var started started

	if err := json.Unmarshal(startFile, &started); err != nil {
		return nil, fmt.Errorf("error parsing started.json from %s: %v", pref, err)
	}
	return &started, nil
}

func (trg *TestResultGatherer) getInformationFromProwFile(ctx context.Context, pref string) (*ProwJob, error) {
	bucket := trg.getBucket()
	rdr, err := bucket.Reader(ctx, pref+"prowjob.json")
	if err != nil {
		return nil, fmt.Errorf("error retrieving prowjob.json from %s: %v", pref, err)
	}

	defer rdr.Close()
	prowFile, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, fmt.Errorf("error reading prowjob.json from %s: %v", pref, err)
	}

	var result *ProwJob

	if err = json.Unmarshal(prowFile, &result); err != nil {
		return nil, fmt.Errorf("error parsing prowjob.json from %s: %v", pref, err)
	}

	return result, nil
}

func (trg *TestResultGatherer) getInformationFromCloneFile(ctx context.Context, pref string) ([]*cloneRecord, error) {
	bucket := trg.getBucket()
	rdr, err := bucket.Reader(ctx, pref+"clone-records.json")
	if err != nil {
		return nil, fmt.Errorf("error retrieving clone-records.json from %s: %v", pref, err)
	}

	defer rdr.Close()
	cloneFile, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, fmt.Errorf("error reading clone-records.json from %s: %v", pref, err)
	}

	var records []*cloneRecord

	if err = json.Unmarshal(cloneFile, &records); err != nil {
		return nil, fmt.Errorf("error parsing clone-records.json from %s: %v", pref, err)
	}

	return records, nil
}

func (trg *TestResultGatherer) getInformationFromYamlFile(ctx context.Context, pref string) (*SuiteOutcome, error) {
	bucket := trg.getBucket()
	rdr, err := bucket.Reader(ctx, pref)
	if err != nil {
		return nil, fmt.Errorf("error retrieving yaml from %s: %v", pref, err)
	}

	defer rdr.Close()
	yamlFile, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, fmt.Errorf("error reading yaml from %s: %v", pref, err)
	}

	var suiteOutcome SuiteOutcome

	if err = yaml.Unmarshal(yamlFile, &suiteOutcome); err != nil {
		return nil, fmt.Errorf("error parsing yaml from %s: %v", pref, err)
	}

	return &suiteOutcome, nil
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
			continue
		}
		var signatures []string
		var names []string
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
	return trg.getBucket().ListItems(ctx, testRun+"artifacts/")
}

// getManyResults function return the status of test passing, clone failure, sha number, base sha for each test
// run under each test suite for the given pr.
// Client: client used to get buckets and objects from google cloud storage.
// TestSlice: a slice of Tests objects containing all tests and the path to folder for each test run for the test under such pr.
// Return a map of test suite name -- pr number -- run number -- FortestResult objects.
func (trg *TestResultGatherer) getManyResults(ctx context.Context, testSlice map[string][]string,
	orgLogin string, repoName string) ([]*store.TestResult, error) {

	var allTestRuns []*store.TestResult

	for testName, runPaths := range testSlice {
		for _, runPath := range runPaths {
			if testResult, err := trg.GetTestResult(ctx, testName, runPath); err == nil {
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

func (trg *TestResultGatherer) getManyPostSubmitResults(ctx context.Context, testNames chan pipelinetwo.OutResult,
	orgLogin string, repoName string) (*store.PostSubtmitAllResult, error) {
	allTestResult := &store.PostSubtmitAllResult{}
	var allTestRuns []*store.PostSubmitTestResult
	var allSuiteOutcome []*store.SuiteOutcome
	var allTestOutcome []*store.TestOutcome
	var allFeatureLabel []*store.FeatureLabel

	for item := range testNames {
		if item.Err() != nil {
			return nil, item.Err()
		}
		bucket := trg.getBucket()
		testPref := item.Output()
		testPrefSplit := strings.Split(testPref.(string), "/")
		testName := testPrefSplit[len(testPrefSplit)-2]
		runPaths, err := bucket.ListPrefixes(ctx, testPref.(string))
		if err != nil {
			return nil, err
		}
		for _, runPath := range runPaths {
			if postSubtmitAllResult, err := trg.GetPostSubmitTestResult(ctx, testName, runPath, orgLogin, repoName); err == nil {
				allTestRuns = append(allTestRuns, postSubtmitAllResult.TestResult[0])
				allSuiteOutcome = append(allSuiteOutcome, postSubtmitAllResult.SuiteOutcome...)
				allTestOutcome = append(allTestOutcome, postSubtmitAllResult.TestOutcome...)
				allFeatureLabel = append(allFeatureLabel, postSubtmitAllResult.FeatureLabel...)
			} else {
				return nil, err
			}
		}
	}
	allTestResult.TestResult = allTestRuns
	allTestResult.SuiteOutcome = allSuiteOutcome
	allTestResult.TestOutcome = allTestOutcome
	allTestResult.FeatureLabel = allFeatureLabel
	return allTestResult, nil
}

func (trg *TestResultGatherer) GetTestResult(ctx context.Context, testName string, testRun string) (testResult *store.TestResult, err error) {
	testResult = &store.TestResult{}
	testResult.TestName = testName
	testResult.RunPath = testRun
	testResult.Done = false
	pj, err := trg.getInformationFromProwFile(ctx, testRun)
	if err != nil {
		return nil, err
	}

	if pj.Status.State == TriggeredState || pj.Status.State == PendingState {
		return nil, fmt.Errorf("test is still in progress")
	}
	if pj.Status.State == ErrorState {
		testResult.StartTime = pj.Status.StartTime.Time
		testResult.FinishTime = pj.Status.CompletionTime.Time
		testResult.Result = "ERROR"
	}
	if pj.Status.State == AbortedState {
		testResult.StartTime = pj.Status.StartTime.Time
		testResult.FinishTime = pj.Status.CompletionTime.Time
		testResult.Result = "ABORTED"
	}
	if pj.Status.State == SuccessState || pj.Status.State == FailureState {

		records, err := trg.getInformationFromCloneFile(ctx, testRun)
		if err != nil {
			return nil, err
		}

		if len(records) < 1 {
			return nil, fmt.Errorf("test %s %s has an empty clone file.  Cannot proceed", testName, testRun)
		}
		record := records[0]

		if len(record.Refs.Pulls) < 1 {
			return nil, fmt.Errorf("test %s %s has a malformed clone file.  Cannot proceed", testName, testRun)
		}
		testResult.Sha, err = hex.DecodeString(record.Refs.Pulls[0].Sha)
		if err != nil {
			return nil, err
		}
		testResult.BaseSha = record.Refs.BaseSha
		testResult.CloneFailed = record.Failed

		started, err := trg.getInformationFromStartedFile(ctx, testRun)
		if err != nil {
			return nil, err
		}

		testResult.StartTime = time.Unix(started.Timestamp, 0)

		finished, err := trg.getInformationFromFinishedFile(ctx, testRun)
		if err != storage.ErrObjectNotExist {
			if err != nil {
				return nil, err
			}
			testResult.TestPassed = finished.Passed
			testResult.Result = finished.Result
			testResult.FinishTime = time.Unix(finished.Timestamp, 0)
		}
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

func (trg *TestResultGatherer) AddChildSuiteOutcome(testResult *store.PostSubmitTestResult,
	suiteOutcome *store.SuiteOutcome) *store.SuiteOutcome {
	suiteOutcome.OrgLogin = testResult.OrgLogin
	suiteOutcome.RepoName = testResult.RepoName
	suiteOutcome.RunNumber = testResult.RunNumber
	suiteOutcome.TestName = testResult.TestName
	suiteOutcome.BaseSha = testResult.BaseSha
	suiteOutcome.Done = testResult.Done
	return suiteOutcome
}

func (trg *TestResultGatherer) AddChildTestOutcome(suiteOutcome *store.SuiteOutcome,
	testOutcome *store.TestOutcome) *store.TestOutcome {
	testOutcome.OrgLogin = suiteOutcome.OrgLogin
	testOutcome.RepoName = suiteOutcome.RepoName
	testOutcome.RunNumber = suiteOutcome.RunNumber
	testOutcome.TestName = suiteOutcome.TestName
	testOutcome.BaseSha = suiteOutcome.BaseSha
	testOutcome.Done = suiteOutcome.Done
	testOutcome.SuiteName = suiteOutcome.SuiteName
	return testOutcome
}

func (trg *TestResultGatherer) AddChildFeatureLabel(testOutcome *store.TestOutcome,
	featureLabel *store.FeatureLabel) *store.FeatureLabel {
	featureLabel.OrgLogin = testOutcome.OrgLogin
	featureLabel.RepoName = testOutcome.RepoName
	featureLabel.RunNumber = testOutcome.RunNumber
	featureLabel.TestName = testOutcome.TestName
	featureLabel.BaseSha = testOutcome.BaseSha
	featureLabel.Done = testOutcome.Done
	featureLabel.SuiteName = testOutcome.SuiteName
	featureLabel.TestOutcomeName = testOutcome.TestOutcomeName
	return featureLabel
}

func (trg *TestResultGatherer) GetPostSubmitTestResult(ctx context.Context, testName string,
	testRun string, orgLogin string, repoName string) (allTestResult *store.PostSubtmitAllResult, err error) {
	allTestResult = &store.PostSubtmitAllResult{}
	var testResultList []*store.PostSubmitTestResult
	var suiteOutcomeList []*store.SuiteOutcome
	var testOutcomeList []*store.TestOutcome
	var featureList []*store.FeatureLabel
	testResult := &store.PostSubmitTestResult{}
	testResult.TestName = testName
	testResult.RunPath = testRun
	testResult.Done = false

	pj, err := trg.getInformationFromProwFile(ctx, testRun)
	if err != nil {
		return nil, err
	}

	if pj.Status.State == TriggeredState || pj.Status.State == PendingState {
		return nil, fmt.Errorf("test is still in progress")
	}
	if pj.Status.State == ErrorState {
		testResult.StartTime = pj.Status.StartTime.Time
		testResult.FinishTime = pj.Status.CompletionTime.Time
		testResult.Result = "ERROR"
	}
	if pj.Status.State == AbortedState {
		testResult.StartTime = pj.Status.StartTime.Time
		testResult.FinishTime = pj.Status.CompletionTime.Time
		testResult.Result = "ABORTED"
	}
	if pj.Status.State == SuccessState || pj.Status.State == FailureState {
		records, err := trg.getInformationFromCloneFile(ctx, testRun)
		if err != nil {
			return nil, err
		}

		if len(records) < 1 {
			return nil, fmt.Errorf("test %s %s has an empty clone file.  Cannot proceed", testName, testRun)
		}
		record := records[0]

		testResult.Sha, err = hex.DecodeString(record.FinalSha)
		if err != nil {
			return nil, err
		}

		testResult.BaseSha = record.Refs.BaseSha
		testResult.CloneFailed = record.Failed

		started, err := trg.getInformationFromStartedFile(ctx, testRun)
		if err != nil {
			return nil, err
		}

		testResult.StartTime = time.Unix(started.Timestamp, 0)

		finished, err := trg.getInformationFromFinishedFile(ctx, testRun)
		if err != storage.ErrObjectNotExist {
			if err != nil {
				return nil, err
			}
			testResult.TestPassed = finished.Passed
			testResult.Result = finished.Result
			testResult.FinishTime = time.Unix(finished.Timestamp, 0)
		}
	}

	prefSplit := strings.Split(testRun, "/")

	runNo, err := strconv.ParseInt(prefSplit[len(prefSplit)-2], 10, 64)
	if err != nil {
		return
	}
	testResult.RunNumber = runNo
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

	testResult.OrgLogin = orgLogin
	testResult.RepoName = repoName

	//saves all artifacts
	for _, yamlFilePath := range artifacts {
		if strings.Contains(strings.Split(yamlFilePath, "/")[4], "yaml") {
			readInSuiteOutcome, err := trg.getInformationFromYamlFile(ctx, yamlFilePath)
			if err != nil {
				return nil, err
			}
			suiteOutcome := &store.SuiteOutcome{}
			suiteOutcome.SuiteName = readInSuiteOutcome.Name
			suiteOutcome.Environment = readInSuiteOutcome.Environment
			suiteOutcome.Multicluster = readInSuiteOutcome.Multicluster
			suiteOutcome = trg.AddChildSuiteOutcome(testResult, suiteOutcome)
			suiteOutcomeList = append(suiteOutcomeList, suiteOutcome)
			for _, readInTestOutcomes := range readInSuiteOutcome.TestOutcomes {
				var testOutcome *store.TestOutcome = &store.TestOutcome{}
				testOutcome.TestOutcomeName = readInTestOutcomes.Name
				testOutcome.Type = readInTestOutcomes.Type
				testOutcome.Outcome = readInTestOutcomes.Outcome
				testOutcome = trg.AddChildTestOutcome(suiteOutcome, testOutcome)
				testOutcomeList = append(testOutcomeList, testOutcome)
				for Feature, Scenario := range readInTestOutcomes.FeatureLabels {
					var featureLabel *store.FeatureLabel = &store.FeatureLabel{}
					featureLabel.Label = Feature
					featureLabel.Scenario = Scenario
					featureLabel = trg.AddChildFeatureLabel(testOutcome, featureLabel)
					featureList = append(featureList, featureLabel)
				}
			}
		}
	}
	testResultList = append(testResultList, testResult)
	allTestResult.TestResult = testResultList
	allTestResult.SuiteOutcome = suiteOutcomeList
	allTestResult.TestOutcome = testOutcomeList
	allTestResult.FeatureLabel = featureList
	return
}

// Read in gcs the folder of the given pr number and write the result of each test runs into a slice of TestFlake struct.
func (trg *TestResultGatherer) CheckTestResultsForPr(ctx context.Context, orgLogin string, repoName string, prNum string) ([]*store.TestResult, error) {
	testSlice, err := trg.GetTestsForPR(ctx, orgLogin, repoName, prNum)
	if err != nil {
		return nil, err
	}
	fullResult, err := trg.getManyResults(ctx, testSlice, orgLogin, repoName)

	if err != nil {
		return nil, err
	}
	return fullResult, nil
}

func (trg *TestResultGatherer) CheckPostSubmitTestResults(ctx context.Context, orgLogin string, repoName string) (*store.PostSubtmitAllResult, error) {
	testNames := trg.GetAllPostSubmitTestChan(ctx).Go()
	fullResult, err := trg.getManyPostSubmitResults(ctx, testNames, orgLogin, repoName)

	if err != nil {
		return nil, err
	}
	return fullResult, nil
}

func (trg *TestResultGatherer) GetAllPullRequestsChan(ctx context.Context, orgLogin string, repoName string) pipelinetwo.Pipeline {
	return trg.getBucket().ListPrefixesProducer(ctx, trg.getRepoPrPath(orgLogin, repoName))
}

func (trg *TestResultGatherer) GetAllPostSubmitTestChan(ctx context.Context) pipelinetwo.Pipeline {
	return trg.getBucket().ListPrefixesProducer(ctx, "logs/")
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
