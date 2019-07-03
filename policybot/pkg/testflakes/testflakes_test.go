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

package testflakes_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/testflakes"
)

func checkEqual(f1, f2 *testflakes.FlakyResult) bool {
	if strings.Compare(f1.TestName, f2.TestName) != 0 {
		return false
	}
	if f1.PrNum != f2.PrNum {
		return false
	}
	if f1.IsFlaky != f2.IsFlaky {
		return false
	}
	if strings.Compare(f1.LastPass, f2.LastPass) != 0 {
		return false
	}
	if strings.Compare(f1.LastFail, f2.LastFail) != 0 {
		return false
	}
	return true
}

func TestFlakes(t *testing.T) {

	const layout = "1/2/2006 15:04:05"
	time1, _ := time.Parse(layout, "11/16/2018 07:03:22")
	t1 := time1.Local()
	time2, _ := time.Parse(layout, "11/16/2019 07:15:44")
	t2 := time2.Local()

	time3, _ := time.Parse(layout, "11/13/2018 07:03:22")
	t3 := time3.Local()
	time4, _ := time.Parse(layout, "11/26/2018 07:15:44")
	t4 := time4.Local()

	time5, _ := time.Parse(layout, "8/13/2018 07:03:22")
	t5 := time5.Local()
	time6, _ := time.Parse(layout, "9/26/2018 07:15:44")
	t6 := time6.Local()

	time7, _ := time.Parse(layout, "8/13/2006 07:03:22")
	t7 := time7.Local()
	time8, _ := time.Parse(layout, "9/26/2019 07:15:44")
	t8 := time8.Local()

	ctx := context.Background()
	testResult1 := &storage.TestResult{
		TestName:   "test1",
		Sha:        "sha1",
		StartTime:  t1,
		FinishTime: t2,
		PrNum:      1,
		RunNum:     1,
		TestPassed: true,
	}

	testResult2 := &storage.TestResult{
		TestName:   "test1",
		Sha:        "sha1",
		StartTime:  t3,
		FinishTime: t4,
		PrNum:      1,
		RunNum:     2,
		TestPassed: false,
	}

	testResult3 := &storage.TestResult{
		TestName:   "test2",
		Sha:        "sha2",
		StartTime:  t5,
		FinishTime: t6,
		PrNum:      1,
		RunNum:     3,
		TestPassed: true,
	}

	testResult4 := &storage.TestResult{
		TestName:   "test2",
		Sha:        "sha2",
		StartTime:  t7,
		FinishTime: t8,
		PrNum:      1,
		RunNum:     3,
		TestPassed: true,
	}
	testResults := []*storage.TestResult{testResult1, testResult2, testResult3, testResult4}

	flakeTester, err := testflakes.NewFlakeTester(ctx, nil, nil, nil, "TestResults")
	if err != nil {
		t.Fail()
	}

	resultMap := flakeTester.ProcessResults(testResults)
	flakyResults := flakeTester.CheckResults(resultMap)

	flakyResult1 := &testflakes.FlakyResult{
		TestName: "test1",
		PrNum:    1,
		IsFlaky:  true,
		LastPass: "",
		LastFail: "",
	}

	flakyResult2 := &testflakes.FlakyResult{
		TestName: "test2",
		PrNum:    1,
		IsFlaky:  false,
		LastPass: "",
		LastFail: "",
	}

	for _, fr := range flakyResults {
		if !checkEqual(fr, flakyResult1) {
			if !checkEqual(fr, flakyResult2) {
				t.Fail()
			}
		}
	}

}
