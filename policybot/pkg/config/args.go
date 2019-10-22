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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Work around the fact time.Duration doesn't support JSON serialization for some reason
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration value %s")
	}
}

// Secrets are typically only set when the process starts.
type Secrets struct {
	GitHubWebhookSecret     string
	GitHubToken             string
	GCPCredentials          string
	SendGridAPIKey          string
	ZenHubToken             string
	GitHubOAuthClientSecret string
	GitHubOAuthClientID     string
}

// Nag expresses some matching conditions against a PR, along with a message to inject into a PR
// when it matches.
//
// The model is for a given PR:
//
//	if (PR title matches any of NatchTitle) || (PR body matches any of MatchBody) {
// 		if (PR files match any of MatchFiles) {
//			if (PR does not match any of AbsentFiles) {
//				produce a nag message in the PR
//			}
//		}
//	}
type Nag struct {
	// Name of the nag
	Name string

	// MatchTitle represents content that must be in the PR's title
	MatchTitle []string // regexes

	// MatchBody represents content that must be in the PR's body
	MatchBody []string // regexes

	// MatchFiles represents files that must be in the PR
	MatchFiles []string // regexes

	// AbsentFiles represents files that must not be in the PR
	AbsentFiles []string // regexes

	// The message to inject when any of the Match* expressions match and none of the Absent* expressions do.
	Message string
}

type FlakeChaser struct {
	// Name of the nag.
	Name string

	// InactiveDays represents the days that a flaky test issue hasn't been updated.
	InactiveDays int

	// CreatedDays determines the bot search range, only issues created within this days ago
	// are considered.
	CreatedDays int

	// Message is the message the bot comments on flaky test issues.
	Message string

	// DryRun determines whether we post updates to the issues.
	DryRun bool

	// Repos determines the repo this bot is applied on, format, "istio/istio", "istio/proxy"
	Repos []string
}

type Label struct {
	Name        string
	Description string
	Color       string
}

type AutoLabel struct {
	// Name of the auto label
	Name string

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

type Milestone struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	Closed      bool      `json:"closed"`
}

type Boilerplate struct {
	Name        string `json:"name"`
	Regex       string `json:"regex"`
	Replacement string `json:"replacement"`
}

// Configuration for an individual repo.
type Repo struct {
	// Name of the repo
	Name string `json:"name"`

	// Labels to create for this repo
	LabelsToCreate []Label `json:"labels_to_create"`

	// Milestones to create for this repo
	MilestonesToCreate []Milestone `json:"milestones_to_create"`
}

// Configuration for an individual GitHub organization.
type Org struct {
	// Name of the org
	Name string `json:"name"`

	// Per-repo configuration
	Repos []Repo `json:"repos"`

	// Nags to apply within this organization
	Nags []Nag `json:"nags"`

	// Automatic labels to apply within this organization
	AutoLabels []AutoLabel `json:"autolabels"`

	// Labels to create in all repos being controlled in this organization
	LabelsToCreate []Label `json:"labels_to_create"`

	// Milestones to create in all repos being controlled in this organization
	MilestonesToCreate []Milestone `json:"milestones_to_create"`

	// PR or issue boilerplate to remove in all repos being controlled in this organization
	BoilerplatesToClean []Boilerplate `json:"boilerplates_to_clean"`

	// BucketName to locate prow test output
	BucketName string `json:"bucket_name"`

	// PresubmitTestPath to locate presubmit test output within the bucket
	PreSubmitTestPath string `json:"presubmit_path"`

	// PostSubmitTestPath to locate postsubmit test output within the bucket
	PostSubmitTestPath string `json:"postsubmit_path"`
}

type Lifecycle struct {
	FeatureRequestLabel string   `json:"feature_request_label"`
	IgnoreLabels        []string `json:"ignore_labels"`
	RealOldDelay        Duration `json:"real_old_delay"`

	TriageDelay Duration `json:"triage_delay"`
	TriageLabel string   `json:"triage_label"`

	EscalationDelay Duration `json:"escalation_delay"`
	EscalationLabel string   `json:"escalation_label"`

	PullRequestStaleDelay    Duration `json:"pull_request_stale_delay"`
	FeatureRequestStaleDelay Duration `json:"feature_request_stale_delay"`
	IssueStaleDelay          Duration `json:"issue_stale_delay"`
	StaleLabel               string   `json:"stale_label"`
	StaleComment             string   `json:"stale_comment"`
	CantBeStaleLabel         string   `json:"cant_be_stale_label"`

	PullRequestCloseDelay    Duration `json:"pull_request_close_delay"`
	FeatureRequestCloseDelay Duration `json:"feature_request_close_delay"`
	IssueCloseDelay          Duration `json:"issue_close_delay"`
	CloseLabel               string   `json:"close_label"`
	CloseComment             string   `json:"close_comment"`
}

// Args represents the set of options that control the behavior of the bot.
type Args struct {
	Secrets Secrets

	ConfigFile             string
	ConfigRepo             string
	ServerPort             int
	HTTPSOnly              bool
	EnableTestResultFilter bool
	SyncerFilter           string

	// The path to the Google Cloud Spanner database to use
	SpannerDatabase string `json:"spanner_db"`

	// Configuration for individual GitHub organizations
	Orgs []Org `json:"orgs"`

	// Global nagging state
	Nags []Nag `json:"nags"`

	// Global flaky test bots to nag issue owner.
	FlakeChaser FlakeChaser `json:"flakechaser"`

	// Global auto-labeling
	AutoLabels []AutoLabel `json:"autolabels"`

	// Global PR or issue boilerplate to remove
	BoilerplatesToClean []Boilerplate `json:"boilerplates_to_clean"`

	// Name to use as sender when sending emails
	EmailFrom string `json:"email_from"`

	// Email address to use as originating address when sending emails
	EmailOriginAddress string `json:"email_origin_address"`

	// The amount of time cache state is kept around before being discarded
	CacheTTL Duration `json:"cache_ttl"`

	// Labels to create in all repos being controlled
	LabelsToCreate []Label `json:"labels_to_create"`

	// Milestones to create in all repos being controlled
	MilestonesToCreate []Milestone `json:"milestones_to_create"`

	// Name of GCP project that holds the GCS test buckets
	GCPProject string `json:"gcp_project"`

	// Time window within which a maintainer is considered active on the project
	MaintainerActivityWindow Duration `json:"maintainer_activity_window"`

	// Time window within which a member is considered active on the project
	MemberActivityWindow Duration `json:"member_activity_window"`

	// Users that are in fact robots
	Robots []string `json:"robots"`

	// Default GitHub org to use in the UI when none is specified
	DefaultOrg string `json:"default_org"`

	// Settings for issue and pr lifecycle
	Lifecycle Lifecycle `json:"lifecycle"`
}

func DefaultArgs() *Args {
	return &Args{
		ServerPort:               8080,
		CacheTTL:                 Duration(15 * time.Minute),
		MaintainerActivityWindow: Duration(90 * 24 * time.Hour),
		MemberActivityWindow:     Duration(180 * 24 * time.Hour),
		Lifecycle: Lifecycle{
			TriageLabel:              "lifecycle/needs triage",
			EscalationDelay:          Duration(7 * 24 * time.Hour),
			EscalationLabel:          "lifecycle/needs escalation",
			FeatureRequestLabel:      "enhancement",
			PullRequestStaleDelay:    Duration(30 * 24 * time.Hour),
			FeatureRequestStaleDelay: Duration(30 * 24 * time.Hour),
			IssueStaleDelay:          Duration(30 * 24 * time.Hour),
			StaleLabel:               "lifecycle/stale",
			StaleComment:             "",
			CantBeStaleLabel:         "lifecycle/staleproof",
			PullRequestCloseDelay:    Duration(60 * 24 * time.Hour),
			FeatureRequestCloseDelay: Duration(60 * 24 * time.Hour),
			IssueCloseDelay:          Duration(60 * 24 * time.Hour),
			CloseLabel:               "",
			CloseComment:             "",
		},
	}
}

// String produces a stringified version of the arguments for debugging.
func (a *Args) String() string {
	var sb strings.Builder

	// don't output secrets in the logs...
	_, _ = fmt.Fprintf(&sb, "ConfigFile: %s\n", a.ConfigFile)
	_, _ = fmt.Fprintf(&sb, "ConfigRepo: %s\n", a.ConfigRepo)
	_, _ = fmt.Fprintf(&sb, "ServerPort: %d\n", a.ServerPort)
	_, _ = fmt.Fprintf(&sb, "HTTPSOnly: %v\n", a.HTTPSOnly)
	_, _ = fmt.Fprintf(&sb, "EnableTestResultFilter: %v\n", a.EnableTestResultFilter)
	_, _ = fmt.Fprintf(&sb, "SpannerDatabase: %s\n", a.SpannerDatabase)
	_, _ = fmt.Fprintf(&sb, "Orgs: %+v\n", a.Orgs)
	_, _ = fmt.Fprintf(&sb, "Nags: %+v\n", a.Nags)
	_, _ = fmt.Fprintf(&sb, "AutoLabels: %+v\n", a.AutoLabels)
	_, _ = fmt.Fprintf(&sb, "EmailFrom: %s\n", a.EmailFrom)
	_, _ = fmt.Fprintf(&sb, "EmailOriginAddress: %s\n", a.EmailOriginAddress)
	_, _ = fmt.Fprintf(&sb, "CacheTTL: %s\n", time.Duration(a.CacheTTL))
	_, _ = fmt.Fprintf(&sb, "GCPProject: %s\n", a.GCPProject)
	_, _ = fmt.Fprintf(&sb, "MaintainerActivityWindow: %v\n", time.Duration(a.MaintainerActivityWindow))
	_, _ = fmt.Fprintf(&sb, "MemberActivityWindow: %v\n", time.Duration(a.MemberActivityWindow))
	_, _ = fmt.Fprintf(&sb, "DefaultOrg: %v\n", a.DefaultOrg)
	_, _ = fmt.Fprintf(&sb, "Lifecycle: %v\n", a.Lifecycle)

	return sb.String()
}
