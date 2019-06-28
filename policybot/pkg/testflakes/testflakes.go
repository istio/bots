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

/*
 * Test Flakes read all rows from TestResults table in Spanner gh database.
 * Convert them back to TestResults struct in types and bind tests based on PR
 * number. If a test both pass and fail for the same PR, it is very likely to be flaky.
 */

package testflakes

import (
	"context"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	store "istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/log"
)

/*
 * FlakyResult struct include the test name, whether or not the test seems to be flaky
 * and the most recent pass and fail instance for the test.
 */
type FlakyResult struct {
	TestName   string
	PrNum      int64
	IsFlaky    bool
	LastPass   string
	passResult *store.TestResult
	LastFail   string
	failResult *store.TestResult
}

type FlakeTest struct {
	ctx    context.Context
	client *spanner.Client
	table  string
}

var scope = log.RegisterScope("TestFlaky", "Check if tests are flaky", 0)

func NewFlakeTest(ctx context.Context, project string, instance string, database string, table string) (*FlakeTest, error) {
	client, err := spanner.NewClient(ctx, "projects/"+project+"/instances/"+instance+"/databases/"+database)

	if err != nil {
		scope.Errorf(err.Error())
		return nil, err
	}

	f := &FlakeTest{
		ctx:    ctx,
		client: client,
		table:  table,
	}
	return f, nil
}

/*
 * Real all rows from table in Spanner and store the results into a slice of TestResult objects.
 */
func (f *FlakeTest) readAll() ([]*store.TestResult, error) {
	iter := f.client.Single().Read(f.ctx, f.table, spanner.AllKeys(),
		[]string{"OrgID", "RepoID", "TestName", "PrNum", "RunNum", "StartTime",
			"FinishTime", "TestPassed", "CloneFailed", "Sha", "Result", "BaseSha", "RunPath"})
	defer iter.Stop()
	testResults := []*store.TestResult{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			scope.Infof("finished reading")
			return testResults, nil
		}
		if err != nil {
			return nil, err
		}
		testResult := &store.TestResult{}
		err = row.ToStruct(testResult)
		if err != nil {
			return nil, err
		}
		testResults = append(testResults, testResult)
	}
}

/*
 * Rearrange information from TestResult to extract test names and whether or not they
 * passed for each pull request and run.
 */
func (f *FlakeTest) processResults(testResults []*store.TestResult) map[string]map[string]map[bool][]*store.TestResult {
	resultMap := map[string]map[string]map[bool][]*store.TestResult{}
	for _, result := range testResults {
		testName := result.TestName
		sha := result.Sha
		testPassed := result.TestPassed
		var testMap map[string]map[bool][]*store.TestResult
		var ok bool
		if testMap, ok = resultMap[testName]; ok {
			var shaMap map[bool][]*store.TestResult
			if shaMap, ok = testMap[sha]; ok {
				var passMap []*store.TestResult
				if passMap, ok = shaMap[testPassed]; !ok {
					passMap = []*store.TestResult{}
				}
				passMap = append(passMap, result)
				shaMap[testPassed] = passMap
			} else {
				shaMap = map[bool][]*store.TestResult{}
				passMap := []*store.TestResult{}
				passMap = append(passMap, result)
				shaMap[testPassed] = passMap
			}
			testMap[sha] = shaMap
		} else {
			shaMap := map[bool][]*store.TestResult{}
			results := []*store.TestResult{}
			results = append(results, result)
			shaMap[testPassed] = results
			testMap = map[string]map[bool][]*store.TestResult{}
			testMap[sha] = shaMap
		}
		resultMap[testName] = testMap
	}
	return resultMap
}

/*
 * Process the map returned from ProcessResults to check if one test has multiple TestPass values
 * coexisting at the same time. If for one Pull Request the test passed and failed for different runs
 * we mark the test to be flaky.
 */
func (f *FlakeTest) checkResults(resultMap map[string]map[string]map[bool][]*store.TestResult) []*FlakyResult {
	flakyResults := []*FlakyResult{}
	for testName, testMap := range resultMap {
		for _, shaMap := range testMap {
			flakyResult := &FlakyResult{}
			flakyResult.TestName = testName
			flakyResult.IsFlaky = false
			if len(shaMap) > 1 {
				flakyResult.IsFlaky = true
			}
			if shaMap[false] != nil {
				failedTests := shaMap[false]
				for _, fail := range failedTests {
					if flakyResult.PrNum == 0 {
						flakyResult.PrNum = fail.PrNum
					}
					if strings.Compare(flakyResult.LastFail, "") != 0 {
						if flakyResult.failResult.FinishTime.Before(fail.FinishTime) {
							flakyResult.failResult = fail
							flakyResult.LastFail = fail.RunPath
						}
					} else {
						flakyResult.failResult = fail
						flakyResult.LastFail = fail.RunPath
					}
				}
			}
			if shaMap[true] != nil {
				passedTests := shaMap[true]
				for _, pass := range passedTests {
					if flakyResult.PrNum == 0 {
						flakyResult.PrNum = pass.PrNum
					}
					if strings.Compare(flakyResult.LastPass, "") != 0 {
						if flakyResult.passResult.FinishTime.Before(pass.FinishTime) {
							flakyResult.LastPass = pass.RunPath
							flakyResult.passResult = pass
						}
					} else {
						flakyResult.passResult = pass
						flakyResult.LastPass = pass.RunPath
					}
				}
			}
			flakyResults = append(flakyResults, flakyResult)
		}
	}
	return flakyResults
}

/*
 * Read table to process stored data and output flaky results
 */
func (f *FlakeTest) ReadTableAndCheckForFlake() ([]*FlakyResult, error) {
	testResults, err := f.readAll()
	if err != nil {
		return nil, err
	}
	resultMap := f.processResults(testResults)
	flakyResults := f.checkResults(resultMap)
	return flakyResults, nil
}
