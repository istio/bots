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

// Test Flakes read all rows from TestResults table in storage.
// Convert them back to TestResults struct in types and bind tests based on PR
// number. If a test both pass and fail for the same PR, it is very likely to be flaky.
package testflakes

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/gh"
	store "istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// FlakyResult struct include the test name, whether or not the test seems to be flaky
// and the most recent pass and fail instance for the test.
type FlakyResult struct {
	TestName   string
	OrgLogin   string
	RepoName   string
	PrNum      int64
	IsFlaky    bool
	LastPass   string
	passResult *store.TestResult
	LastFail   string
	failResult *store.TestResult
}

type FlakeTester struct {
	ght   *gh.ThrottledClient
	ctx   context.Context
	table string
	cache *cache.Cache
	store store.Store
}

var scope = log.RegisterScope("FlakeTester", "Check if tests are flaky", 0)

func NewFlakeTester(ctx context.Context, cache *cache.Cache, store store.Store, ght *gh.ThrottledClient, table string) (*FlakeTester, error) {
	f := &FlakeTester{
		ght:   ght,
		ctx:   ctx,
		cache: cache,
		table: table,
		store: store,
	}

	return f, nil
}

// Rearrange information from TestResult to extract test names and whether or not they
// passed for each pull request and run.
func (f *FlakeTester) ProcessResults(testResults []*store.TestResult) map[string]map[string]map[bool][]*store.TestResult {
	resultMap := map[string]map[string]map[bool][]*store.TestResult{}
	for _, result := range testResults {
		testName := result.TestName
		sha := result.Sha
		testPassed := result.TestPassed
		var testMap map[string]map[bool][]*store.TestResult
		var ok bool
		if testMap, ok = resultMap[testName]; ok {
			var shaMap map[bool][]*store.TestResult
			if shaMap, ok = testMap[hex.EncodeToString(sha)]; ok {
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
			testMap[hex.EncodeToString(sha)] = shaMap
		} else {
			shaMap := map[bool][]*store.TestResult{}
			results := []*store.TestResult{}
			results = append(results, result)
			shaMap[testPassed] = results
			testMap = map[string]map[bool][]*store.TestResult{}
			testMap[hex.EncodeToString(sha)] = shaMap
		}
		resultMap[testName] = testMap
	}
	return resultMap
}

// Process the map returned from ProcessResults to check if one test has multiple TestPass values
// coexisting at the same time. If for one Pull Request the test passed and failed for different runs
// we mark the test to be flaky.
func (f *FlakeTester) CheckResults(resultMap map[string]map[string]map[bool][]*store.TestResult) []*FlakyResult {
	flakyResults := []*FlakyResult{}
	for testName, testMap := range resultMap {
		for _, shaMap := range testMap {
			flakyResult := &FlakyResult{}
			flakyResult.TestName = testName
			flakyResult.IsFlaky = false
			if len(shaMap) > 1 {
				flakyResult.IsFlaky = true
			}
			failFirst := store.TestResult{
				OrgLogin: "",
				RepoName: "",
			}
			if shaMap[false] != nil {
				failedTests := shaMap[false]
				failFirst = *failedTests[0]
				flakyResult.OrgLogin = failFirst.OrgLogin
				flakyResult.RepoName = failFirst.RepoName
				for _, fail := range failedTests {
					if flakyResult.PrNum == 0 {
						flakyResult.PrNum = fail.PullRequestNumber
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
				if strings.Compare(failFirst.OrgLogin, "") == 0 {
					passFirst := passedTests[0]
					flakyResult.RepoName = passFirst.RepoName
					flakyResult.OrgLogin = passFirst.OrgLogin
				}
				for _, pass := range passedTests {
					if flakyResult.PrNum == 0 {
						flakyResult.PrNum = pass.PullRequestNumber
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

// Chase function add issue comment about the flake results
func (f *FlakeTester) ReportFlake(context context.Context, flakeResults []*FlakyResult, message string) {
	for _, flake := range flakeResults {
		comment := &github.PullRequestComment{
			Body: &message,
		}

		url := fmt.Sprintf("https://github.com/%v/%v/pull/%v", flake.OrgLogin, flake.RepoName, flake.PrNum)
		scope.Infof("About to nag test flaky pr with %v", url)

		_, _, err := f.ght.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.CreateComment(
				context, flake.OrgLogin, flake.RepoName, int(flake.PrNum), comment)
		})

		if err != nil {
			scope.Errorf("Failed to create flakes nagging comments: %v", err)
		}
		scope.Infof("Found potential flakes for pull request number %v test %s, in org %s and repo %s", flake.PrNum, flake.TestName, flake.OrgLogin, flake.RepoName)
	}
}
