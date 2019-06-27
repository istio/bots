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

package testflakes

import (
	"context"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	store "istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/log"
)

type FlakyResult struct {
	OrgID    string
	Repo     string
	TestName string
	IsFlaky  bool
	LastPass *store.TestResult
	LastFail *store.TestResult
}

type FlakeTest struct {
	ctx    context.Context
	client *spanner.Client
	table  string
}

var scope = log.RegisterScope("TestFlaky", "Check if tests are flaky", 0)

func NewFlakeTest(project string, instance string, database string, table string) (*FlakeTest, error) {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/"+project+"/instances/"+instance+"/databases/"+database)

	if err != nil {
		scope.Errorf(err.Error())
		return nil, err
	}

	flakeTest := &FlakeTest{
		ctx:    ctx,
		client: client,
		table:  table,
	}
	return flakeTest, nil
}

func (flakeTest *FlakeTest) ReadAll() ([]*store.TestResult, error) {
	iter := flakeTest.client.Single().Read(flakeTest.ctx, flakeTest.table, spanner.AllKeys(),
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
		if err := row.Columns(&testResult.OrgID, &testResult.RepoID, &testResult.TestName, &testResult.PrNum, &testResult.RunNum,
			&testResult.StartTime, &testResult.FinishTime, &testResult.TestPassed,
			&testResult.CloneFailed, &testResult.Sha, &testResult.Result, &testResult.BaseSha, &testResult.RunPath); err != nil {
			return nil, err
		}
		testResults = append(testResults, testResult)
	}
}

func (flakeTest *FlakeTest) ProcessResults(testResults []*store.TestResult) map[string]map[int64]map[bool][]*store.TestResult {
	resultMap := map[string]map[int64]map[bool][]*store.TestResult{}
	for _, result := range testResults {
		testName := result.TestName
		prNum := result.PrNum
		testPassed := result.TestPassed
		if testMap, ok := resultMap[testName]; ok {
			var prMap map[bool][]*store.TestResult
			if prMap, ok = testMap[prNum]; ok {
				var passMap []*store.TestResult
				if passMap, ok = prMap[testPassed]; !ok {
					passMap = []*store.TestResult{}
				}
				passMap = append(passMap, result)
				prMap[testPassed] = passMap
			} else {
				prMap = map[bool][]*store.TestResult{}
				prMap[testPassed] = []*store.TestResult{result}
				testMap[prNum] = prMap
			}
			testMap[prNum] = prMap
			resultMap[testName] = testMap
		} else {
			prMap := map[bool][]*store.TestResult{}
			results := []*store.TestResult{}
			results = append(results, result)
			prMap[testPassed] = results
			testMap := map[int64]map[bool][]*store.TestResult{}
			testMap[prNum] = prMap
			resultMap[testName] = testMap
		}
	}
	return resultMap
}

func (flakeTest *FlakeTest) CheckResults(resultMap map[string]map[int64]map[bool][]*store.TestResult) []*FlakyResult {
	flakyResultMap := []*FlakyResult{}
	for testName, testMap := range resultMap {
		flakeyResult := &FlakyResult{}
		flakeyResult.TestName = testName
		flakeyResult.IsFlaky = false
		for _, prMap := range testMap {
			if len(prMap) > 1 {
				flakeyResult.IsFlaky = true
			}
			if prMap[false] != nil {
				failedTests := prMap[false]
				for _, fail := range failedTests {
					if flakeyResult.LastFail != nil {
						lastFail := flakeyResult.LastFail
						if lastFail.FinishTime.Before(fail.FinishTime) {
							flakeyResult.LastFail = fail
						}
					} else {
						flakeyResult.LastFail = fail
					}
				}
			}
			if prMap[true] != nil {
				passedTests := prMap[true]
				for _, pass := range passedTests {
					if flakeyResult.LastPass != nil {
						lastPass := flakeyResult.LastPass
						if lastPass.FinishTime.Before(pass.FinishTime) {
							flakeyResult.LastPass = pass
						}
					} else {
						flakeyResult.LastPass = pass
					}
				}
			}
		}
		flakyResultMap = append(flakyResultMap, flakeyResult)
	}
	return flakyResultMap
}
