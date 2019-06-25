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
		BaseRef   string
		BaseSha   string
		Pulls     []pull
		PathAlias string
	}
	Commands []cmnd
	Failed   bool
}

/*
 * Tests struct to keep track of the test suite names and the directory for each test runs for the pr.
 */
type tests struct {
	Name string
	Prs  []string
}

/*
 * Started struct to store values from started.json
 */
type started struct {
	Timestamp int64
}

type PrResultTester struct {
	client     *storage.Client
	ctx        context.Context
	bucketName string
}

var scope = log.RegisterScope("TestResult", "Check error while reading from google cloud storage", 0)

/*
 * Function to initionalize new PrFlakeTest object.
 * Use `prFlakeyTest, err := NewPrFlakeTest()` and `testFlakes := prFlakeyTest.checkTestFlakesForPr(<pr_number>)`
 * to get test flakes information for a given pr.
 */
func NewPrResultTester(bucketName string) (*PrResultTester, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &PrResultTester{
		client:     client,
		ctx:        ctx,
		bucketName: bucketName,
	}, nil
}

/*
 * Contains function check if a string exists in a given slice of strings.
 */
func contains(slic []string, ele string) bool {
	for _, e := range slic {
		if strings.Compare(e, ele) == 0 {
			return true
		}
	}
	return false
}

/*
 * GetTest function get all directories under the given pr in istio-prow/pr-logs/pull/istio-istio/PRNUMBER for each test suite name.
 * Client: client used to get buckets and objects.
 * PrNum: the PR number inputted.
 * Return []Tests return a slice of Tests objects.
 */
func (prt PrResultTester) getTests(prNumInt int64) ([]tests, error) {
	prNum := strconv.FormatInt(prNumInt, 10)
	ctx := prt.ctx
	client := prt.client
	bucket := client.Bucket(prt.bucketName)
	query := &storage.Query{Prefix: "pr-logs/pull/istio_istio/" + prNum}
	it := bucket.Objects(ctx, query)
	var testSlice []tests
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		name := attrs.Name

		nameSlice := strings.Split(name, "/")
		prNum = nameSlice[3]
		pullNum := nameSlice[5]
		testName := nameSlice[len(nameSlice)-3]
		fileName := nameSlice[len(nameSlice)-1] // C
		var newString = "pr-logs/pull/istio_istio/" + prNum + "/" + testName + "/" + pullNum
		if strings.Compare(fileName, "started.json") == 0 || strings.Compare(fileName, "clone-records.json") == 0 || strings.Compare(fileName, "finished.json") == 0 {
			var contain = false
			for ind, ele := range testSlice {
				if strings.Compare(ele.Name, testName) == 0 {
					prs := ele.Prs

					if !contains(prs, newString) {
						prs = append(prs, newString)
						ele.Prs = prs
						testSlice[ind] = ele
					}

					contain = true
				}

			}
			if !contain {
				t := tests{
					Name: testName,
				}
				newSlice := []string{newString}
				t.Prs = newSlice
				testSlice = append(testSlice, t)
			}
		}
	}
	return testSlice, nil
}

func (prt PrResultTester) getInformationFromFinishedFile(pref string, onePull *store.TestResult) (*store.TestResult, error) {
	// It is possible that the folder might not contain finished.json.
	client := prt.client
	bucket := client.Bucket(prt.bucketName)
	newObj := bucket.Object(pref + "/finished.json")
	nrdr, nerr := newObj.NewReader(prt.ctx)
	if nerr != nil {
		return nil, nerr
	}

	defer nrdr.Close()
	slur, err := ioutil.ReadAll(nrdr)
	if err != nil {
		return nil, nerr
	}
	ns := string(slur)
	ndec := json.NewDecoder(strings.NewReader(ns))

	for ndec.More() {
		var finished finished
		err = ndec.Decode(&finished)
		if err != nil {
			return nil, err
		}

		passed := finished.Passed
		result := finished.Result
		t := finished.Timestamp
		tm := time.Unix(t, 0)

		onePull.TestPassed = passed
		onePull.Result = result
		onePull.FinishTime = tm
	}
	return onePull, nil
}

func (prt PrResultTester) getInformationFromStartedFile(pref string, onePull *store.TestResult) (*store.TestResult, error) {
	client := prt.client
	bucket := client.Bucket(prt.bucketName)
	newObj := bucket.Object(pref + "/started.json")
	nrdr, nerr := newObj.NewReader(prt.ctx)
	if nerr != nil {
		return nil, nerr
	}

	defer nrdr.Close()
	slur, nerr := ioutil.ReadAll(nrdr)
	if nerr != nil {
		return nil, nerr
	}
	ns := string(slur)
	ndec := json.NewDecoder(strings.NewReader(ns))

	for ndec.More() {
		var started started
		err := ndec.Decode(&started)
		if err != nil {
			return nil, err
		}

		t := started.Timestamp
		tm := time.Unix(t, 0)
		onePull.StartTime = tm

	}
	return onePull, nil
}

func (prt PrResultTester) getInformationFromCloneFile(pref string, onePull *store.TestResult) (*store.TestResult, error) {
	client := prt.client
	bucket := client.Bucket(prt.bucketName)
	obj := bucket.Object(pref + "/clone-records.json")
	scope.Infof("read clone")
	rdr, err := obj.NewReader(prt.ctx)
	if err != nil {
		return nil, err
	}

	defer rdr.Close()
	slurp, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	s := string(slurp)
	dec := json.NewDecoder(strings.NewReader(s))

	_, err = dec.Token()
	if err != nil {
		return nil, err
	}

	for dec.More() {
		var record cloneRecord
		err := dec.Decode(&record)
		if err != nil {
			return nil, err
		}

		refs := record.Refs
		pulls := refs.Pulls
		pull := pulls[0]
		sha := pull.Sha
		baseSha := refs.BaseSha

		failed := record.Failed

		onePull.Sha = sha
		onePull.BaseSha = baseSha
		onePull.CloneFailed = failed

	}
	return onePull, nil
}

/*
 * GetShaAndPassStatus function return the status of test passing, clone failure, sha number, base sha for each test run under each test suite for the given pr.
 * Client: client used to get buckets and objects from google cloud storage.
 * TestSlice: a slice of Tests objects containing all tests and the path to folder for each test run for the test under such pr.
 * Return a map of test suite name -- pr number -- run number -- ForEachRun objects.
 */
func (prt PrResultTester) getShaAndPassStatus(testSlice []tests, orgID string, repoID string) ([]*store.TestResult, error) {
	var allTestRuns = []*store.TestResult{}

	for _, test := range testSlice {
		testName := test.Name

		prefs := test.Prs

		for _, pref := range prefs {

			var onePull = &store.TestResult{}
			onePull.OrgID = orgID
			onePull.RepoID = repoID
			onePull.TestName = testName
			onePull.RunPath = pref

			var err error
			onePull, err = prt.getInformationFromCloneFile(pref, onePull)
			if err != nil {
				return nil, err
			}

			onePull, err = prt.getInformationFromStartedFile(pref, onePull)
			if err != nil {
				return nil, err
			}

			onePull, err = prt.getInformationFromFinishedFile(pref, onePull)
			if err != nil {
				return nil, err
			}

			prefSplit := strings.Split(pref, "/")

			runNo, errr := strconv.ParseInt(prefSplit[len(prefSplit)-1], 10, 64)
			if errr != nil {
				return nil, errr
			}
			onePull.RunNum = runNo
			prNo, newError := strconv.ParseInt(prefSplit[len(prefSplit)-3], 10, 64)
			if newError != nil {
				return nil, newError
			}
			onePull.PrNum = prNo
			allTestRuns = append(allTestRuns, onePull)

		}
	}
	return allTestRuns, nil
}

/*
 * Read in gcs the folder of the given pr number and write the result of each test runs into a slice of TestFlake struct.
 */
func (prt PrResultTester) CheckTestResultsForPr(prNum int64, orgID string, repoID string) ([]*store.TestResult, error) {
	client := prt.client
	defer client.Close()

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
