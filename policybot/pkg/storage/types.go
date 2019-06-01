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
	UserID  string
	Login   string
	Name    string
	Company string
}

type Label struct {
	OrgID       string
	RepoID      string
	LabelID     string
	Name        string
	Description string
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
}

type PullRequest struct {
	OrgID                string
	RepoID               string
	IssueID              string
	UpdatedAt            time.Time
	RequestedReviewerIDs []string
	Files                []string
}

type PullRequestReview struct {
	OrgID               string
	RepoID              string
	IssueID             string
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
	LastSyncStart time.Time
	LastSyncEnd   time.Time
}

type Maintainer struct {
	OrgID  string
	UserID string
	Paths  []string
}
