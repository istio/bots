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

// StartupOptions are set when the process starts and cannot be updated afterwards.
type StartupOptions struct {
	ConfigFile              string
	ConfigRepo              string
	GitHubWebhookSecret     string
	GitHubToken             string
	GCPCredentials          string
	SendGridAPIKey          string
	ZenHubToken             string
	Port                    int
	GitHubOAuthClientSecret string
	GitHubOAuthClientID     string
	HTTPSOnly               bool
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

// Configuration for an individual repo.
type Repo struct {
	// Name of the repo
	Name string `json:"name"`

	// Labels to create for this repo
	LabelsToCreate []Label `json:"labels_to_create"`
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

	// BucketName to locate prow test output
	BucketName string `json:"bucket_name"`

	// PresubmitTestPath to locate presubmit test output within the bucket
	PreSubmitTestPath string `json:"presubmit_path"`

	// PostSubmitTestPath to locate postsubmit test output within the bucket
	PostSubmitTestPath string `json:"postsubmit_path"`
}

// Args represents the set of options that control the behavior of the bot.
type Args struct {
	// StartupOptions are set when the process starts and cannot be updated afterwards
	StartupOptions StartupOptions

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

	// Name to use as sender when sending emails
	EmailFrom string `json:"email_from"`

	// Email address to use as originating address when sending emails
	EmailOriginAddress string `json:"email_origin_address"`

	// The amount of time cache state is kept around before being discarded
	CacheTTL time.Duration `json:"cache_ttl"`

	// Labels to create in all repos being controlled
	LabelsToCreate []Label `json:"labels_to_create"`

	// Name of GCP project that holds the GCS test buckets
	GCPProject string `json:"gcp_project"`

	// Time window within which a maintainer is considered active on the project
	MaintainerActivityWindow Duration `json:"maintainer_activity_window"`

	// Time window within which a member is considered active on the project
	MemberActivityWindow Duration `json:"member_activity_window"`

	// Default GitHub org to use in the UI when none is specified
	DefaultOrg string `json:"default_org"`
}

func DefaultArgs() *Args {
	return &Args{
		StartupOptions: StartupOptions{
			Port: 8080,
		},
		CacheTTL: 15 * time.Minute,
	}
}

// String produces a stringified version of the arguments for debugging.
func (a *Args) String() string {
	var sb strings.Builder

	// don't output secrets in the logs...
	// _, _ = fmt.Fprintf(&sb, "GitHubWebhookSecret: %s\n", a.StartupOptions.GitHubWebhookSecret)
	// _, _ = fmt.Fprintf(&sb, "GitHubToken: %s\n", a.StartupOptions.GitHubToken)
	// _, _ = fmt.Fprintf(&sb, "GCPCredentials: %s\n", a.StartupOptions.GCPCredentials)
	// _, _ = fmt.Fprintf(&sb, "SendGridAPIKey: %s\n", a.StartupOptions.SendGridAPIKey)
	// _, _ = fmt.Fprintf(&sb, "ZenHubToken: %s\n", a.StartupOptions.ZenHubToken)
	// _, _ = fmt.Fprintf(&sb, "GitHubOAuthClientSecret: %s\n", a.StartupOptions.GitHubOAuthClientSecret)
	// _, _ = fmt.Fprintf(&sb, "GitHubOAuthClientID: %s\n", a.StartupOptions.GitHubOAuthClientID)

	_, _ = fmt.Fprintf(&sb, "StartupOptions.ConfigFile: %s\n", a.StartupOptions.ConfigFile)
	_, _ = fmt.Fprintf(&sb, "StartupOptions.ConfigRepo: %s\n", a.StartupOptions.ConfigRepo)
	_, _ = fmt.Fprintf(&sb, "StartupOptions.Port: %d\n", a.StartupOptions.Port)
	_, _ = fmt.Fprintf(&sb, "SpannerDatabase: %s\n", a.SpannerDatabase)
	_, _ = fmt.Fprintf(&sb, "Orgs: %+v\n", a.Orgs)
	_, _ = fmt.Fprintf(&sb, "Nags: %+v\n", a.Nags)
	_, _ = fmt.Fprintf(&sb, "AutoLabels: %+v\n", a.AutoLabels)
	_, _ = fmt.Fprintf(&sb, "EmailFrom: %s\n", a.EmailFrom)
	_, _ = fmt.Fprintf(&sb, "EmailOriginAddress: %s\n", a.EmailOriginAddress)
	_, _ = fmt.Fprintf(&sb, "CacheTTL: %s\n", a.CacheTTL)
	_, _ = fmt.Fprintf(&sb, "GCPProject: %s\n", a.GCPProject)
	_, _ = fmt.Fprintf(&sb, "MaintainerActivityWindow: %v\n", a.MaintainerActivityWindow)
	_, _ = fmt.Fprintf(&sb, "MemberActivityWindow: %v\n", a.MemberActivityWindow)
	_, _ = fmt.Fprintf(&sb, "DefaultOrg: %v\n", a.DefaultOrg)

	return sb.String()
}
