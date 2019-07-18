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
	OrgLogin  string
	UserLogin string
}

type BotActivity struct {
	OrgLogin                              string
	RepoName                              string
	LastIssueSyncStart                    time.Time
	LastIssueCommentSyncStart             time.Time
	LastPullRequestReviewCommentSyncStart time.Time
}

type Maintainer struct {
	OrgLogin  string
	UserLogin string
	Paths     []string // where each path is of the form RepoName/path_in_repo
	Emeritus  bool
}

type IssuePipeline struct {
	OrgLogin    string
	RepoName    string
	IssueNumber int64
	Pipeline    string
}

type TimedEntry struct {
	Time time.Time
	ID   int64 // an object ID
}

type RepoActivityInfo struct {
	RepoName                       string                // ID of the repo
	LastPullRequestCommittedByPath map[string]TimedEntry // last update a maintainer has done to one of their maintained paths
	LastIssueCommented             TimedEntry            // last issue commented on by the maintainer
	LastIssueClosed                TimedEntry            // last issue closed by the maintainer
	LastIssueTriaged               TimedEntry            // last issue triaged by the maintainer
}

type MaintainerInfo struct {
	Repos map[string]*RepoActivityInfo // about the maintainer's activity in different repos (index is repo name)
}

type TestResult struct {
	OrgLogin          string
	RepoName          string
	TestName          string
	TestPassed        bool
	CloneFailed       bool
	Done              bool
	PullRequestNumber int64
	RunNumber         int64
	StartTime         time.Time
	FinishTime        time.Time
	Sha               string
	Result            string
	BaseSha           string
	RunPath           string
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
