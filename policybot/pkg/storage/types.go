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

package storage

import (
	"time"

	"cloud.google.com/go/spanner"
)

// This file defines the shapes we csn read/write to/from the DB. Before
// adding a new column, it must pre-exist in Spanner. The order of steps
// is as follows:
//
//     1. Add the column to Spanner (which must be nullable).
//     2. Add the field with a pointer type to the storage struct, to
//        allow nil value.
//     3. Run syncer to populate the column.
//     4. Convert the column to be not nullable.
//     5. Change the pointer type to non pointer type in the struct.

type Issue struct {
	OrgLogin    string
	RepoName    string
	IssueNumber int64
	Title       string
	Body        string
	Labels      []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    time.Time
	State       string
	Author      string
	Assignees   []string
}

type IssueComment struct {
	OrgLogin       string
	RepoName       string
	IssueNumber    int64
	IssueCommentID int64
	Author         string
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type User struct {
	UserLogin string
	Name      string
	Company   string
	AvatarURL string
}

type Label struct {
	OrgLogin    string
	RepoName    string
	LabelName   string
	Description string
	Color       string
}

type Org struct {
	OrgLogin    string
	Company     string
	AvatarURL   string
	Description string
}

type Repo struct {
	OrgLogin    string
	RepoName    string
	Description string
	RepoNumber  int64
}

type PullRequest struct {
	OrgLogin           string
	RepoName           string
	PullRequestNumber  int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ClosedAt           time.Time
	MergedAt           time.Time
	Title              string
	Body               string
	Labels             []string
	Assignees          []string
	RequestedReviewers []string
	Files              []string
	Author             string
	State              string
	BranchName         string
	HeadCommit         string
	Merged             bool
}

type PullRequestReviewComment struct {
	OrgLogin                   string
	RepoName                   string
	PullRequestNumber          int64
	PullRequestReviewCommentID int64
	Author                     string
	Body                       string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type PullRequestReview struct {
	OrgLogin            string
	RepoName            string
	PullRequestNumber   int64
	PullRequestReviewID int64
	Author              string
	Body                string
	SubmittedAt         time.Time
	State               string
}

type Member struct {
	OrgLogin   string
	UserLogin  string
	CachedInfo string // a JSON encoded ActivityInfo
}

// Release Qualification Test Related Types
// Monitor represents the status of specific Monitor
type Monitor struct {
	// MonitorName is the name of the monitor, e.g. ContainerMemoryUsage
	MonitorName string
	// Status represents the status of the monitor, e.g. HEALTHY, ALERTING
	Status string
	// ProjectID points to the project where the test is running
	ProjectID string
	// ClusterName points to the cluster where the test is running
	ClusterName string
	TestID      string
	// Branch of the test, e.g. release-1.7
	Branch string
	// Description of the monitor
	Description spanner.NullString
	// UpdatedTime represents the time of the monitor status update
	UpdatedTime time.Time
	// LastFiredTime represents the last timestamp when the alert is fired.
	LastFiredTime spanner.NullTime
	// FiredTimes represents the number of times the monitor fired alerts.
	FiredTimes spanner.NullInt64
	IsActive   *bool
}

// ReleaseQualTestMetadata represents the metadata of specific test run.
type ReleaseQualTestMetadata struct {
	// ProjectID points to the project where the test is running
	ProjectID string
	// ClusterName points to the cluster where the test is running
	ClusterName string
	// TestID is the ID of specific Test run
	TestID string
	// Branch of the test, e.g. release-1.7
	Branch string
	// PrometheusLink links to prometheus running in the cluster
	PrometheusLink spanner.NullString
	// GrafanaLink links to prometheus running in the cluster
	GrafanaLink spanner.NullString
}

type BotActivity struct {
	OrgLogin                              string
	RepoName                              string
	LastIssueSyncStart                    time.Time
	LastIssueCommentSyncStart             time.Time
	LastPullRequestReviewCommentSyncStart time.Time
}

type Maintainer struct {
	OrgLogin   string
	UserLogin  string
	Paths      []string // where each path is of the form RepoName/path_in_repo
	Emeritus   bool
	CachedInfo string // a JSON encoded ActivityInfo
}

type TimedEntry struct {
	Time   time.Time
	Number int64 // an object number (issue or pr)
}

type RepoPathActivityInfo struct {
	LastPullRequestSubmitted TimedEntry
	LastPullRequestReviewed  TimedEntry
}

type RepoActivityInfo struct {
	Paths              map[string]RepoPathActivityInfo // info about each maintained path in the repo
	LastIssueCommented TimedEntry                      // last issue commented on by the maintainer
	LastIssueClosed    TimedEntry                      // last issue closed by the maintainer
	LastIssueTriaged   TimedEntry                      // last issue triaged by the maintainer
}

type ActivityInfo struct {
	Repos        map[string]*RepoActivityInfo // about user activity in different repos (index is repo name)
	LastActivity time.Time                    // when is the last time any activity took place
}

type TestResult struct {
	StartTime  time.Time
	FinishTime time.Time
	Signatures []string
	OrgLogin   string
	RepoName   string
	TestName   string
	Sha        []byte
	Result     string
	// TODO: why is Sha bytes and basesha string?
	BaseSha           string
	RunPath           string
	PullRequestNumber int64
	RunNumber         int64
	TestPassed        bool
	CloneFailed       bool
	Done              bool
	HasArtifacts      bool
	Artifacts         []string
}

type PostSubmitTestResult struct {
	StartTime    time.Time
	FinishTime   time.Time
	Signatures   []string
	OrgLogin     string
	RepoName     string
	TestName     string
	Sha          []byte
	Result       string
	BaseSha      string
	RunPath      string
	RunNumber    int64
	TestPassed   bool
	CloneFailed  bool
	Done         bool
	HasArtifacts bool
	Artifacts    []string
}

type SuiteOutcome struct {
	OrgLogin     string
	RepoName     string
	RunNumber    int64
	TestName     string
	BaseSha      string
	Done         bool
	SuiteName    string
	Environment  string
	Multicluster bool
}

type TestOutcome struct {
	OrgLogin        string
	RepoName        string
	RunNumber       int64
	TestName        string
	BaseSha         string
	Done            bool
	SuiteName       string
	TestOutcomeName string
	Type            string
	Outcome         string
}

type FeatureLabel struct {
	OrgLogin        string
	RepoName        string
	RunNumber       int64
	TestName        string
	BaseSha         string
	Done            bool
	SuiteName       string
	TestOutcomeName string
	Label           string
	Scenario        []string
}

type PostSubtmitAllResult struct {
	TestResult   []*PostSubmitTestResult
	SuiteOutcome []*SuiteOutcome
	TestOutcome  []*TestOutcome
	FeatureLabel []*FeatureLabel
}

type TestOutcomeFeatureLabel struct {
	TestOutcome  []*TestOutcome
	FeatureLabel []*FeatureLabel
}

type RepoComment struct {
	OrgLogin  string
	RepoName  string
	CommentID int64
	Body      string
	Author    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RepoCommentEvent struct {
	OrgLogin      string
	RepoName      string
	RepoCommentID int64
	CreatedAt     time.Time
	Actor         string
	Action        string
}

type PullRequestReviewEvent struct {
	OrgLogin            string
	RepoName            string
	PullRequestNumber   int64
	PullRequestReviewID int64
	CreatedAt           time.Time
	Actor               string
	Action              string
}

type PullRequestEvent struct {
	OrgLogin          string
	RepoName          string
	PullRequestNumber int64
	CreatedAt         time.Time
	Actor             string
	Action            string
	Merged            bool
}

type PullRequestReviewCommentEvent struct {
	OrgLogin                   string
	RepoName                   string
	PullRequestNumber          int64
	PullRequestReviewCommentID int64
	CreatedAt                  time.Time
	Actor                      string
	Action                     string
}

type IssueCommentEvent struct {
	OrgLogin       string
	RepoName       string
	IssueNumber    int64
	IssueCommentID int64
	CreatedAt      time.Time
	Actor          string
	Action         string
}

type IssueEvent struct {
	OrgLogin    string
	RepoName    string
	IssueNumber int64
	CreatedAt   time.Time
	Actor       string
	Action      string
}

type CoverageData struct {
	OrgLogin     string
	RepoName     string
	BranchName   string
	PackageName  string
	Sha          string
	TestName     string
	Type         string
	CompletedAt  time.Time
	StmtsCovered int64
	StmtsTotal   int64
}

type UserAffiliation struct {
	UserLogin    string
	Counter      int64
	Organization string
	StartTime    time.Time
	EndTime      time.Time
}

type BaseSha struct {
	BaseSha string
}

type LatestBaseSha struct {
	BaseSha        string
	LastFinishTime time.Time
	NumberofTest   int64
}

type PostSubmitTestEnvLabel struct {
	Environment string
	Label       string
}

type LatestBaseShaSummary struct {
	LatestBaseSha []LatestBaseSha
}

type TestNameByEnvLabel struct {
	TestOutcomeName string
	RunNumber       int64
	TestName        string
}

type ConfirmedFlake struct {
	OrgLogin          string
	RepoName          string
	PullRequestNumber int64
	RunNumber         int64
	TestName          string
	Done              bool
	PassingRunNumber  int64
	IssueNum          int64
}
