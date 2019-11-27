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

package labeler

import (
	"istio.io/bots/policybot/pkg/config"
)

const recordType = "autolabel"

type autoLabelRecord struct {
	config.RecordBase

	// MatchTitle represents content that must be in the PR or issue's title
	MatchTitle []string // regexes

	// MatchBody represents content that must be in the PR or issue's body
	MatchBody []string // regexes

	// AbsentLabels represents labels that must not be on the PR or issue
	AbsentLabels []string // regexes

	// PresentLabels represents labels that must be on the PR or issue
	PresentLabels []string // regexes

	// The labels to apply when any of the Match* expressions match and none of the Absent* expressions do.
	LabelsToApply []string

	// The labels to remove when any of the Match* expressions match and none of the Absent* expressions do.
	LabelsToRemove []string
}

func init() {
	config.RegisterType(recordType, config.MultiplePerRepo, func() config.Record {
		return new(autoLabelRecord)
	})
}
