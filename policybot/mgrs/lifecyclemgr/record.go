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

package lifecyclemgr

import (
	"time"

	"istio.io/bots/policybot/pkg/config"
)

const RecordType = "lifecycle"

type lifecycleRecord struct {
	config.RecordBase

	FeatureRequestLabel string          `json:"feature_request_label"`
	IgnoreLabels        []string        `json:"ignore_labels"`
	RealOldDelay        config.Duration `json:"real_old_delay"`

	TriageDelay config.Duration `json:"triage_delay"`
	TriageLabel string          `json:"triage_label"`

	EscalationDelay config.Duration `json:"escalation_delay"`
	EscalationLabel string          `json:"escalation_label"`

	PullRequestStaleDelay    config.Duration `json:"pull_request_stale_delay"`
	FeatureRequestStaleDelay config.Duration `json:"feature_request_stale_delay"`
	IssueStaleDelay          config.Duration `json:"issue_stale_delay"`
	StaleLabel               string          `json:"stale_label"`
	StaleComment             string          `json:"stale_comment"`
	CantBeStaleLabel         string          `json:"cant_be_stale_label"`

	PullRequestCloseDelay    config.Duration `json:"pull_request_close_delay"`
	FeatureRequestCloseDelay config.Duration `json:"feature_request_close_delay"`
	IssueCloseDelay          config.Duration `json:"issue_close_delay"`
	CloseLabel               string          `json:"close_label"`
	CloseComment             string          `json:"close_comment"`
}

func init() {
	config.RegisterType(RecordType, config.OnePerRepo, func() config.Record {
		return &lifecycleRecord{
			TriageLabel:              "lifecycle/needs triage",
			EscalationDelay:          config.Duration(7 * 24 * time.Hour),
			EscalationLabel:          "lifecycle/needs escalation",
			FeatureRequestLabel:      "enhancement",
			PullRequestStaleDelay:    config.Duration(30 * 24 * time.Hour),
			FeatureRequestStaleDelay: config.Duration(30 * 24 * time.Hour),
			IssueStaleDelay:          config.Duration(30 * 24 * time.Hour),
			StaleLabel:               "lifecycle/stale",
			StaleComment:             "",
			CantBeStaleLabel:         "lifecycle/staleproof",
			PullRequestCloseDelay:    config.Duration(60 * 24 * time.Hour),
			FeatureRequestCloseDelay: config.Duration(60 * 24 * time.Hour),
			IssueCloseDelay:          config.Duration(60 * 24 * time.Hour),
			CloseLabel:               "",
			CloseComment:             "",
		}
	})
}
