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

package resultgatherer

import (
	"context"
	"reflect"
	"testing"
	"time"

	"istio.io/bots/policybot/pkg/blobstorage/gcs"
	"istio.io/bots/policybot/pkg/storage"
)

// the 110 pr directory in istio-flakey-test/pr-logs/pull/istio-istio only has a release-test folder
func TestTestResultGatherer(t *testing.T) {
	context := context.Background()
	const layout = "1/2/2006 15:04:05"
	time1, _ := time.Parse(layout, "11/16/2018 07:03:22")
	t1 := time1.Local()
	time2, _ := time.Parse(layout, "11/16/2018 07:15:44")
	t2 := time2.Local()
	var correctInfo = &storage.TestResult{
		OrgLogin:          "istio",
		RepoName:          "istio",
		TestName:          "release-test",
		PullRequestNumber: 110,
		RunNumber:         155,
		StartTime:         t1,
		FinishTime:        t2,
		TestPassed:        true,
		CloneFailed:       false,
		Sha:               "fee4aae74eb4debaf621d653abe8bfcf0ce6a4ea",
		Result:            "SUCCESS",
		BaseSha:           "d995c19aefe6b5ff0748b783e8b69c59963bc8ae",
		RunPath:           "pr-logs/pull/istio_istio/110/release-test/155/",
		Artifacts:         nil,
	}
	var prNum int64 = 110

	client, err := gcs.NewStore(context, nil)
	if err != nil {
		t.Fatalf("unable to create GCS client: %v", err)
	}

	testResultGatherer := TestResultGatherer{client, "istio-flakey-test", "pr-logs/pull/", ""}
	testResults, err := testResultGatherer.CheckTestResultsForPr(context, "istio", "istio", prNum)
	if err != nil {
		t.Errorf("Expecting no error, got %v", err)
		return
	}

	if len(testResults) == 0 {
		t.Errorf("Expected at least one test result from bucket istio-flakey-test")
		return
	}

	test := testResults[0]

	if !reflect.DeepEqual(test, correctInfo) {
		t.Errorf("Wanted %#v, got %#v", correctInfo, test)
	}
}
