/**
 * Take in a pr number from path "istio-prow/pr-logs/pull/istio-istio" and examine the pr
 * for all tests that are run and their results. The results are then written to Spanner.
 */
package main

import (
	"context"
	"strings"
	"encoding/json"
	"io/ioutil"
	"log"
	"fmt"
	"strconv"
	"time"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

/*
 * Pull struct for the structure under refs/pulls in clone-records.json
 */
type Pull struct {
	Number int
	Author string
	Sha string
}

/*
 * Cmd struct for Commands object under clone-records.json
 */
type Cmnd struct {
	Command string
	Output string
}

/*
 * Finished struct to store values for all fields in Finished.json
 */
type Finished struct {
	Timestamp int64
	Passed bool
	Result string
}

/*
 * Clone_Record struct to store values for all fields in clone-records.json
 */
type CloneRecord struct {
	Refs struct {
		Org string
		Repo string
		BaseRef string
		BaseSha string
		Pulls []Pull
		PathAlias string
	}
	Commands []Cmnd
	Failed bool
}

/*
 * Tests strut to keep track of the test suite names and the directory for each test runs for the pr.
 */
type Tests struct {
	Name string
	Prs []string
}

/*
 * Started struct to store values from started.json
 */
type Started struct {
	Timestamp int64
}

/*
 * TestFlakes struct stores all elements to be writtened to Spanner for each test run for a given pr under a given test.
 */
type TestFlake struct {
	TestName string
	PrNum string
	RunNum string
	StartTime time.Time 
	FinishTime time.Time
	Passed string
	CloneFailed string
	Sha string
	Result string
	BaseSha string
	RunPath string
}

/*
 * Contains function check if a string exists in a given slice of strings.
 */
func contains(slic []string, ele string) bool {
	for _, e := range(slic) {
		if strings.Compare(e, ele) == 0 {
			return true
		}
	}
	return false
}

/*
 * GetTest function get all directories under the given pr in istio-prow/pr-logs/pull/istio-istio/PRNUMBER for each test suite name.
 * @param client client used to get buckets and objects.
 * @param prNum the PR number inputted.
 * @return []Tests return a slice of Tests objects.
 */
func getTests(client *storage.Client, prNum string) []Tests {
	ctx := context.Background()
	bucket := client.Bucket("istio-prow")
	query := &storage.Query{Prefix:    "pr-logs/pull/istio_istio/" + prNum}
	it := bucket.Objects(ctx, query)
	var testSlice []Tests
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		name := attrs.Name

		nameSlice := strings.Split(name, "/")
		prNum := nameSlice[3]
		pullNum := nameSlice[5]
		testName := nameSlice[len(nameSlice) - 3]
		fileName := nameSlice[len(nameSlice)-1] // C
		var newString = "pr-logs/pull/istio_istio/" + prNum + "/" + testName + "/" + pullNum
		if strings.Compare(fileName, "started.json") == 0 || strings.Compare(fileName, "clone-records.json") == 0 || strings.Compare(fileName, "finished.json") == 0 {
			var contain = false
			for ind, ele := range(testSlice) {
				if strings.Compare(ele.Name, testName) == 0 {
					prs := ele.Prs

					if contains(prs, newString) == false {
						prs = append(prs, newString)
						ele.Prs = prs
						testSlice[ind] = ele
					}
					
					contain = true
				}

			}
			if contain == false {
				t := Tests{
					Name: testName,
				}
				newSlice := []string{newString}
				t.Prs = newSlice
				testSlice = append(testSlice, t)
			}
		}
	}
	return testSlice
}

/*
 * GetShaAndPassStatus function return the status of test passing, clone failure, sha number, base sha for each test run under each test suite for the given pr.
 * @param client client used to get buckets and objects from google cloud storage.
 * @param testSlice a slice of Tests objects containing all tests and the path to folder for each test run for the test under such pr.
 * @return a map of test suite name -- pr number -- run number -- ForEachRun objects.
 */
func getShaAndPassStatus(client *storage.Client, testSlice []Tests) []TestFlake {
	ctx := context.Background()
	bucket := client.Bucket("istio-prow")

	var allTestRuns = []TestFlake{}

	for _, test := range(testSlice) {
		testName := test.Name

		prefs := test.Prs

		for _, pref := range(prefs) {

			var onePull = TestFlake{}

			onePull.TestName = testName
			onePull.RunPath = pref

			obj := bucket.Object(pref + "/clone-records.json")
			log.Println("read clone")
			rdr, err := obj.NewReader(ctx)
			if err != nil {
		        log.Println("readFile: unable to open file from bucket %q, file %q: %v", err)
		    }

		    defer rdr.Close()
		    slurp, err := ioutil.ReadAll(rdr)
			if err != nil {
		        log.Println("readFile: unable to read data from bucket %q, file %q: %v", err)
		    }
			s := string(slurp)
		    dec := json.NewDecoder(strings.NewReader(s))

		    t, err := dec.Token()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%T: %v\n", t, t)

			for dec.More() {
				var record CloneRecord
				err := dec.Decode(&record)
				if err != nil {
					log.Fatal(err)
				}

				refs := record.Refs
				pulls := refs.Pulls
				pull := pulls[0]
				sha := pull.Sha
				baseSha := refs.BaseSha

				failed := record.Failed
				failedToString := strconv.FormatBool(failed)

				onePull.Sha = sha
				onePull.BaseSha = baseSha
				onePull.CloneFailed = failedToString
				
				
			}
			newObj := bucket.Object(pref + "/started.json")
			nrdr, nerr := newObj.NewReader(ctx)
			if nerr != nil {
		        log.Println("readFile: unable to open file from bucket %q, file %q: %v", nerr)
		    }

		    defer nrdr.Close()
		    slur, nerr := ioutil.ReadAll(nrdr)
			if err != nil {
		        log.Println("readFile: unable to read data from bucket %q, file %q: %v", nerr)
		    }
			ns := string(slur)
		    ndec := json.NewDecoder(strings.NewReader(ns))

			for ndec.More() {
				var started Started
				err = ndec.Decode(&started)
				if err != nil {
					log.Println("error second to last ")
					log.Fatal(err)
				}

				t := started.Timestamp
				tm := time.Unix(t, 0)
				onePull.StartTime = tm
				
			}

			// It is possible that the folder might not contain finished.json.
			newObj = bucket.Object(pref + "/finished.json")
			nrdr, nerr = newObj.NewReader(ctx)
			if nerr != nil {
		        log.Println("readFile: unable to open file from bucket %q, file %q: %v", nerr)
		    } else {

		    	defer nrdr.Close()
			    slur, nerr = ioutil.ReadAll(nrdr)
				if err != nil {
			        log.Println("readFile: unable to read data from bucket %q, file %q: %v", nerr)
			    }
				ns = string(slur)
			    ndec = json.NewDecoder(strings.NewReader(ns))

				for ndec.More() {
					var finished Finished
					err = ndec.Decode(&finished)
					if err != nil {
						log.Println("error second to last ")
						log.Fatal(err)
					}

					passed := finished.Passed
					result := finished.Result
					t := finished.Timestamp
					tm := time.Unix(t, 0)
					passedToString := strconv.FormatBool(passed)

					onePull.Passed = passedToString
					onePull.Result = result
					onePull.FinishTime = tm
				}

		    }

			prefSplit := strings.Split(pref, "/")

			onePull.RunNum = prefSplit[len(prefSplit) - 1]
			onePull.PrNum = prefSplit[len(prefSplit) - 3]

			allTestRuns = append(allTestRuns, onePull)

		}
	}
	return allTestRuns
}

/* 
 * Read in gcs the folder of the given pr number and write the result of each test runs into a slice of TestFlake struct.
 */
func checkTestFlakesForPr(prNum string) []fullResult {
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	var testSlice = getTests(client, prNum)
	var fullResult = getShaAndPassStatus(client, testSlice)

	log.Println(fullResult)
	return fullResult
}