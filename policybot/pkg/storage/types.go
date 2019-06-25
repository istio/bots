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

// This file defines the shapes we csn read/write to/from the DB.

type Issue struct {
	OrgID       string
	RepoID      string
	IssueID     string
	Number      int64
	Title       string
	Body        string
	LabelIDs    []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    time.Time
	State       string
	AuthorID    string
	AssigneeIDs []string
}

type IssueComment struct {
	OrgID          string
	RepoID         string
	IssueID        string
	IssueCommentID string
	AuthorID       string
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type User struct {
	UserID    string
	Login     string
	Name      string
	Company   string
	AvatarURL string
}

type Label struct {
	OrgID       string
	RepoID      string
	LabelID     string
	Name        string
	Description string
	Color       string
}

type Org struct {
	OrgID string
	Login string
}

type Repo struct {
	OrgID       string
	RepoID      string
	Name        string
	Description string
	RepoNumber  int64
}

type PullRequest struct {
	OrgID                string
	RepoID               string
	PullRequestID        string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ClosedAt             time.Time
	MergedAt             time.Time
	Title                string
	Body                 string
	LabelIDs             []string
	AssigneeIDs          []string
	RequestedReviewerIDs []string
	Files                []string
	AuthorID             string
	State                string
	Number               int64
}

type PullRequestComment struct {
	OrgID                string
	RepoID               string
	PullRequestID        string
	PullRequestCommentID string
	AuthorID             string
	Body                 string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type PullRequestReview struct {
	OrgID               string
	RepoID              string
	PullRequestID       string
	PullRequestReviewID string
	AuthorID            string
	Body                string
	SubmittedAt         time.Time
	State               string
}

type Member struct {
	OrgID  string
	UserID string
}

type BotActivity struct {
	OrgID              string
	RepoID             string
	LastIssueSyncStart time.Time
}

type Maintainer struct {
	OrgID    string
	UserID   string
	Paths    []string // where each path is of the form RepoID/path_in_repo
	Emeritus bool
}

type IssuePipeline struct {
	OrgID       string
	RepoID      string
	IssueNumber int64
	Pipeline    string
}

type TimedEntry struct {
	Time time.Time
	ID   string // an object ID (pr, issue, issue comment)
}

type RepoActivityInfo struct {
	RepoID                         string                // ID of the repo
	LastPullRequestCommittedByPath map[string]TimedEntry // last update a maintainer has done to one of their maintained paths
	LastIssueCommented             TimedEntry            // last issue commented on by the maintainer
	LastIssueClosed                TimedEntry            // last issue closed by the maintainer
	LastIssueTriaged               TimedEntry            // last issue triaged by the maintainer
}

type MaintainerInfo struct {
	Repos map[string]*RepoActivityInfo // about the maintainer's activity in different repos (index is repo id)
}

type RepoComment struct {
	OrgID     string
	RepoID    string
	CommentID string
	Body      string
	AuthorID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
