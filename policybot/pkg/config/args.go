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
	"bytes"
	"fmt"
	"time"
)

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

	// InactiveDays represents the days that a flakey test issue hasn't been updated.
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

type AutoLabel struct {
	// Name of the auto label
	Name string

	// MatchTitle represents content that must be in the PR or issue's title
	MatchTitle []string // regexes

	// MatchBody represents content that must be in the PR or issue's body
	MatchBody []string // regexes

	// AbsentLabels represents labels that must not be on the PR or issue
	AbsentLabels []string // regexes

	// The labels to apply when any of the Match* expressions match and none of the Absent* expressions do.
	Labels []string
}

// Configuration for an individual repo.
type Repo struct {
	// Name of the repo
	Name string `json:"name"`
}

// Configuration for an individual GitHub organization.
type Org struct {
	// Name of the org
	Name string `json:"name"`

	// Per-repo configuration
	Repos []Repo `json:"repos"`

	// Nags to apply within this organization
	Nags       []Nag       `json:"nags"`
	AutoLabels []AutoLabel `json:"autolabels"`
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

	// Global flaky test bots to nag issuer owner.
	FlakeChaser FlakeChaser `json:"flakechaser"`

	// Global auto-labeling
	AutoLabels []AutoLabel `json:"autolabels"`

	// Name to use as sender when sending emails
	EmailFrom string `json:"email_from"`

	//BucketName to use to directo to gcs bucket
	BucketName string `json:"bucket_name"`

	// Email address to use as originating address when sending emails
	EmailOriginAddress string `json:"email_origin_address"`

	// The amount of time cache state is kept around before being discarded
	CacheTTL time.Duration `json:"cache_ttl"`
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
	buf := &bytes.Buffer{}

	// don't output secrets in the logs...
	// _, _ = fmt.Fprintf(buf, "GitHubWebhookSecret: %s\n", a.StartupOptions.GitHubWebhookSecret)
	// _, _ = fmt.Fprintf(buf, "GitHubToken: %s\n", a.StartupOptions.GitHubToken)
	// _, _ = fmt.Fprintf(buf, "GCPCredentials: %s\n", a.StartupOptions.GCPCredentials)
	// _, _ = fmt.Fprintf(buf, "SendGridAPIKey: %s\n", a.StartupOptions.SendGridAPIKey)
	// _, _ = fmt.Fprintf(buf, "ZenHubToken: %s\n", a.StartupOptions.ZenHubToken)
	// _, _ = fmt.Fprintf(buf, "GitHubOAuthClientSecret: %s\n", a.StartupOptions.GitHubOAuthClientSecret)
	// _, _ = fmt.Fprintf(buf, "GitHubOAuthClientID: %s\n", a.StartupOptions.GitHubOAuthClientID)

	_, _ = fmt.Fprintf(buf, "StartupOptions.ConfigFile: %s\n", a.StartupOptions.ConfigFile)
	_, _ = fmt.Fprintf(buf, "StartupOptions.ConfigRepo: %s\n", a.StartupOptions.ConfigRepo)
	_, _ = fmt.Fprintf(buf, "StartupOptions.Port: %d\n", a.StartupOptions.Port)
	_, _ = fmt.Fprintf(buf, "SpannerDatabase: %s\n", a.SpannerDatabase)
	_, _ = fmt.Fprintf(buf, "Orgs: %+v\n", a.Orgs)
	_, _ = fmt.Fprintf(buf, "Nags: %+v\n", a.Nags)
	_, _ = fmt.Fprintf(buf, "AutoLabels: %+v\n", a.AutoLabels)
	_, _ = fmt.Fprintf(buf, "EmailFrom: %s\n", a.EmailFrom)
	_, _ = fmt.Fprintf(buf, "EmailOriginAddress: %s\n", a.EmailOriginAddress)
	_, _ = fmt.Fprintf(buf, "CacheTTL: %s\n", a.CacheTTL)

	return buf.String()
}
