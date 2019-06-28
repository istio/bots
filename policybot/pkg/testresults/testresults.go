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

/**
 * Take in a pr number from path "istio-prow/pr-logs/pull/istio-istio" and examine the pr
 * for all tests that are run and their results. The results are then written to Spanner.
 */
package testresults

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
	"istio.io/pkg/log"
)

/*
 * Pull struct for the structure under refs/pulls in clone-records.json
 */
type pull struct {
	Number int
	Author string
	Sha    string
}

/*
 * Cmd struct for Commands object under clone-records.json
 */
type cmnd struct {
	Command string
	Output  string
}

/*
 * Finished struct to store values for all fields in Finished.json
 */
type finished struct {
	Timestamp int64
	Passed    bool
	Result    string
}

/*
 * Clone_Record struct to store values for all fields in clone-records.json
 */
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

/*
 * Started struct to store values from started.json
 */
type started struct {
	Timestamp int64
}

type PrResultTester struct {
	Client     *storage.Client
	ctx        context.Context
	bucketName string
}

var scope = log.RegisterScope("TestResult", "Check error while reading from google cloud storage", 0)

/*
 * Function to initionalize new PrFlakeTest object.
 * Use `prFlakeyTest, err := NewPrFlakeTest()` and `testFlakes := prFlakeyTest.checkTestFlakesForPr(<pr_number>)`
 * to get test flakes information for a given pr.
 */
func NewPrResultTester(ctx context.Context, bucketName string) (*PrResultTester, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &PrResultTester{
		Client:     client,
		ctx:        ctx,
		bucketName: bucketName,
	}, nil
}

func (prt *PrResultTester) query(prefix string) ([]string, error) {
	ctx := prt.ctx
	client := prt.Client
	bucket := client.Bucket(prt.bucketName)
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

/*
 * GetTest function get all directories under the given pr in istio-prow/pr-logs/pull/istio-istio/PRNUMBER for each test suite name.
 * Client: client used to get buckets and objects.
 * PrNum: the PR number inputted.
 * Return []Tests return a slice of Tests objects.
 */
func (prt *PrResultTester) getTests(prNumInt int64) (map[string][]string, error) {
	prNum := strconv.FormatInt(prNumInt, 10)
	prefixForPr := "pr-logs/pull/istio_istio/" + prNum + "/"
	testNames, err := prt.query(prefixForPr)
	if err != nil {
		return nil, err
	}
	testMap := map[string][]string{}
	var runs []string
	for _, testPref := range testNames {
		testPrefSplit := strings.Split(testPref, "/")
		testname := testPrefSplit[len(testPrefSplit)-2]
		runs, err = prt.query(testPref)
		if err != nil {
			return nil, err
		}
		var runPaths []string
		var ok bool
		runPaths, ok = testMap[testname]
		if !ok {
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

func (prt *PrResultTester) getInformationFromFinishedFile(pref string, eachRun *store.TestResult) (*store.TestResult, error) {
	// It is possible that the folder might not contain finished.json.
	client := prt.Client
	bucket := client.Bucket(prt.bucketName)
	newObj := bucket.Object(pref + "finished.json")
	nrdr, nerr := newObj.NewReader(prt.ctx)
	var finish finished

	if nerr != nil {
		return nil, nerr
	}

	defer nrdr.Close()
	finishFile, err := ioutil.ReadAll(nrdr)
	if err != nil {
		return nil, nerr
	}
	err = json.Unmarshal(finishFile, &finish)
	if err != nil {
		return eachRun, err
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

func (prt *PrResultTester) getInformationFromStartedFile(pref string, eachRun *store.TestResult) (*store.TestResult, error) {
	client := prt.Client
	bucket := client.Bucket(prt.bucketName)
	newObj := bucket.Object(pref + "started.json")
	nrdr, nerr := newObj.NewReader(prt.ctx)
	if nerr != nil {
		return nil, nerr
	}

	defer nrdr.Close()
	startFile, nerr := ioutil.ReadAll(nrdr)
	if nerr != nil {
		return nil, nerr
	}

	var started started
	err := json.Unmarshal(startFile, &started)
	if err != nil {
		return eachRun, err
	}
	t := started.Timestamp
	tm := time.Unix(t, 0)
	eachRun.StartTime = tm

	return eachRun, nil
}

func (prt *PrResultTester) getInformationFromCloneFile(pref string, eachRun *store.TestResult) (*store.TestResult, error) {
	client := prt.Client
	bucket := client.Bucket(prt.bucketName)
	obj := bucket.Object(pref + "clone-records.json")
	scope.Infof("read clone")
	scope.Infof(pref + "clone-records.json")
	rdr, err := obj.NewReader(prt.ctx)
	if err != nil {
		return nil, err
	}

	defer rdr.Close()
	cloneFile, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}

	var records []cloneRecord
	err = json.Unmarshal(cloneFile, &records)
	if err != nil {
		return eachRun, err
	}
	record := records[0]
	refs := record.Refs
	pulls := refs.Pulls
	pull := pulls[0]
	sha := pull.Sha
	baseSha := refs.BaseSha
	failed := record.Failed

	eachRun.Sha = sha
	eachRun.BaseSha = baseSha
	eachRun.CloneFailed = failed

	return eachRun, nil
}

/*
 * GetShaAndPassStatus function return the status of test passing, clone failure, sha number, base sha for each test run under each test suite for the given pr.
 * Client: client used to get buckets and objects from google cloud storage.
 * TestSlice: a slice of Tests objects containing all tests and the path to folder for each test run for the test under such pr.
 * Return a map of test suite name -- pr number -- run number -- ForEachRun objects.
 */
func (prt *PrResultTester) getShaAndPassStatus(testSlice map[string][]string, orgID string, repoID string) ([]*store.TestResult, error) {
	var allTestRuns = []*store.TestResult{}

	for testName, runPaths := range testSlice {
		for _, runPath := range runPaths {

			var eachRun = &store.TestResult{}
			eachRun.OrgID = orgID
			eachRun.RepoID = repoID
			eachRun.TestName = testName
			eachRun.RunPath = runPath

			var err error
			eachRun, err = prt.getInformationFromCloneFile(runPath, eachRun)
			if err != nil {
				return nil, err
			}

			eachRun, err = prt.getInformationFromStartedFile(runPath, eachRun)
			if err != nil {
				return nil, err
			}

			eachRun, err = prt.getInformationFromFinishedFile(runPath, eachRun)
			if err != nil {
				return nil, err
			}

			prefSplit := strings.Split(runPath, "/")

			runNo, errr := strconv.ParseInt(prefSplit[len(prefSplit)-2], 10, 64)
			if errr != nil {
				return nil, errr
			}
			eachRun.RunNum = runNo
			prNo, newError := strconv.ParseInt(prefSplit[len(prefSplit)-4], 10, 64)
			if newError != nil {
				return nil, newError
			}
			eachRun.PrNum = prNo
			allTestRuns = append(allTestRuns, eachRun)

		}
	}
	return allTestRuns, nil
}

/*
 * Read in gcs the folder of the given pr number and write the result of each test runs into a slice of TestFlake struct.
 */
func (prt *PrResultTester) CheckTestResultsForPr(prNum int64, orgID string, repoID string) ([]*store.TestResult, error) {
	testSlice, err := prt.getTests(prNum)
	if err != nil {
		return nil, err
	}
	fullResult, er := prt.getShaAndPassStatus(testSlice, orgID, repoID)

	if er != nil {
		return nil, er
	}
	return fullResult, nil
}
