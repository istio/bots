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

package refresher

import (
	"istio.io/bots/policybot/pkg/config"
)

const RecordType = "testoutputs"

type TestOutputRecord struct {
	config.RecordBase

	// BucketName to locate prow test output
	BucketName string `json:"bucket_name"`

	// PresubmitTestPath to locate presubmit test output within the bucket
	PreSubmitTestPath string `json:"presubmit_path"`

	// PostSubmitTestPath to locate postsubmit test output within the bucket
	PostSubmitTestPath string `json:"postsubmit_path"`
}

func init() {
	config.RegisterType(RecordType, config.OnePerRepo, func() config.Record {
		return new(TestOutputRecord)
	})
}
