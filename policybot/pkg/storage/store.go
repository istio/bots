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
	"io"
)

// Store defines how the bot interacts with the database
type Store interface {
	io.Closer

	WriteOrgs(orgs []*Org) error
	WriteRepos(repos []*Repo) error
	WriteIssues(issues []*Issue) error
	WriteIssueComments(issueComments []*IssueComment) error
	WritePullRequests(prs []*PullRequest) error
	WritePullRequestReviews(prReviews []*PullRequestReview) error
	WriteUsers(users []*User) error
	WriteLabels(labels []*Label) error
	WriteAllMembers(members []*Member) error
	WriteAllMaintainers(maintainers []*Maintainer) error
	WriteBotActivities(activities []*BotActivity) error

	ReadOrgByID(orgID string) (*Org, error)
	ReadOrgByLogin(login string) (*Org, error)
	ReadRepoByID(orgID string, repoID string) (*Repo, error)
	ReadRepoByName(orgID string, name string) (*Repo, error)
	ReadIssueByID(orgID string, repoID string, issueID string) (*Issue, error)
	ReadIssueByNumber(orgID string, repoID string, number int) (*Issue, error)
	ReadIssueCommentByID(orgID string, repoID string, issueID string, issueCommentID string) (*IssueComment, error)
	ReadLabelByID(orgID string, repoID string, labelID string) (*Label, error)
	ReadUserByID(userID string) (*User, error)
	ReadUserByLogin(login string) (*User, error)
	ReadPullRequestByID(orgID string, repoID string, issueID string) (*PullRequest, error)
	ReadPullRequestReviewByID(orgID string, repoID string, issueID string, prReviewID string) (*PullRequestReview, error)
	ReadBotActivityByID(orgID string, repoID string) (*BotActivity, error)

	QueryMembersByOrg(orgID string, cb func(*Member) error) error
	QueryMaintainersByOrg(orgID string, cb func(*Maintainer) error) error

	EnumUsers(cb func(*User) bool) error
}
