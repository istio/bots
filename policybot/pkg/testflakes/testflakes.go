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
package testflakes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	store "istio.io/bots/policybot/pkg/storage"
)

/*
 * Pull struct for the structure under refs/pulls in clone-records.json
 */
type Pull struct {
	Number int
	Author string
	Sha    string
}

/*
 * Cmd struct for Commands object under clone-records.json
 */
type Cmnd struct {
	Command string
	Output  string
}

/*
 * Finished struct to store values for all fields in Finished.json
 */
type Finished struct {
	Timestamp int64
	Passed    bool
	Result    string
}

/*
 * Clone_Record struct to store values for all fields in clone-records.json
 */
type CloneRecord struct {
	Refs struct {
		Org       string
		Repo      string
		BaseRef   string
		BaseSha   string
		Pulls     []Pull
		PathAlias string
	}
	Commands []Cmnd
	Failed   bool
}

/*
 * Tests strut to keep track of the test suite names and the directory for each test runs for the pr.
 */
type Tests struct {
	Name string
	Prs  []string
}

/*
 * Started struct to store values from started.json
 */
type Started struct {
	Timestamp int64
}

type PrFlakeTest struct {
	client *storage.Client
	ctx    context.Context
}

/*
 * Function to initionalize new PrFlakeTest object.
 * Use `prFlakeyTest, err := NewPrFlakeTest()` and `testFlakes := prFlakeyTest.checkTestFlakesForPr(<pr_number>)`
 * to get test flakes information for a given pr.
 */
func NewPrFlakeTest() (*PrFlakeTest, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &PrFlakeTest{
		client: client,
		ctx:    ctx,
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
func (prFlakeTest PrFlakeTest) getTests(prNumInt int64) ([]Tests, error) {
	prNum := strconv.FormatInt(prNumInt, 10)
	ctx := prFlakeTest.ctx
	client := prFlakeTest.client
	bucket := client.Bucket("istio-prow")
	query := &storage.Query{Prefix: "pr-logs/pull/istio_istio/" + prNum}
	it := bucket.Objects(ctx, query)
	var testSlice []Tests
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
				t := Tests{
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

/*
 * GetShaAndPassStatus function return the status of test passing, clone failure, sha number, base sha for each test run under each test suite for the given pr.
 * Client: client used to get buckets and objects from google cloud storage.
 * TestSlice: a slice of Tests objects containing all tests and the path to folder for each test run for the test under such pr.
 * Return a map of test suite name -- pr number -- run number -- ForEachRun objects.
 */
func (prFlakeTest PrFlakeTest) getShaAndPassStatus(testSlice []Tests) ([]*store.TestFlake, error) {
	ctx := prFlakeTest.ctx
	client := prFlakeTest.client
	bucket := client.Bucket("istio-prow")

	var allTestRuns = []*store.TestFlake{}

	for _, test := range testSlice {
		testName := test.Name

		prefs := test.Prs

		for _, pref := range prefs {

			var onePull = &store.TestFlake{}

			onePull.TestName = testName
			onePull.RunPath = pref

			obj := bucket.Object(pref + "/clone-records.json")
			log.Println("read clone")
			rdr, err := obj.NewReader(ctx)
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

			t, err := dec.Token()
			if err != nil {
				return nil, err
			}
			fmt.Printf("%T: %v\n", t, t)

			for dec.More() {
				var record CloneRecord
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
			newObj := bucket.Object(pref + "/started.json")
			nrdr, nerr := newObj.NewReader(ctx)
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
				var started Started
				err = ndec.Decode(&started)
				if err != nil {
					return nil, err
				}

				t := started.Timestamp
				tm := time.Unix(t, 0)
				onePull.StartTime = tm

			}

			// It is possible that the folder might not contain finished.json.
			newObj = bucket.Object(pref + "/finished.json")
			nrdr, nerr = newObj.NewReader(ctx)
			if nerr != nil {
				return nil, nerr
			}

			defer nrdr.Close()
			slur, nerr = ioutil.ReadAll(nrdr)
			if err != nil {
				return nil, nerr
			}
			ns = string(slur)
			ndec = json.NewDecoder(strings.NewReader(ns))

			for ndec.More() {
				var finished Finished
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

func (prFlakeTest PrFlakeTest) SetOrgID(orgID string, testFlakes []*store.TestFlake) []*store.TestFlake {
	newTestFlakes := []*store.TestFlake{}
	for _, testFlake := range testFlakes {
		testFlake.OrgID = orgID
		newTestFlakes = append(newTestFlakes, testFlake)
	}
	return newTestFlakes
}

/*
 * Read in gcs the folder of the given pr number and write the result of each test runs into a slice of TestFlake struct.
 */
func (prFlakeTest PrFlakeTest) CheckTestFlakesForPr(prNum int64) ([]*store.TestFlake, error) {
	client := prFlakeTest.client
	defer client.Close()

	testSlice, err := prFlakeTest.getTests(prNum)
	if err != nil {
		return nil, err
	}
	fullResult, er := prFlakeTest.getShaAndPassStatus(testSlice)

	if er != nil {
		return nil, er
	}
	return fullResult, nil
}
