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
	"context"
	"io"
)

// Store defines how the bot interacts with the database
type Store interface {
	io.Closer

	WriteOrgs(context context.Context, orgs []*Org) error
	WriteRepos(context context.Context, repos []*Repo) error
	WriteRepoComments(context context.Context, comments []*RepoComment) error
	WriteIssues(context context.Context, issues []*Issue) error
	WriteIssueComments(context context.Context, issueComments []*IssueComment) error
	WriteIssuePipelines(context context.Context, issueData []*IssuePipeline) error
	WritePullRequests(context context.Context, prs []*PullRequest) error
	WritePullRequestComments(context context.Context, prComments []*PullRequestComment) error
	WritePullRequestReviews(context context.Context, prReviews []*PullRequestReview) error
	WriteUsers(context context.Context, users []*User) error
	WriteLabels(context context.Context, labels []*Label) error
	WriteAllMembers(context context.Context, members []*Member) error
	WriteAllMaintainers(context context.Context, maintainers []*Maintainer) error
	WriteBotActivities(context context.Context, activities []*BotActivity) error

	ReadOrgByID(context context.Context, orgID string) (*Org, error)
	ReadOrgByLogin(context context.Context, login string) (*Org, error)
	ReadRepoByID(context context.Context, orgID string, repoID string) (*Repo, error)
	ReadRepoByName(context context.Context, orgID string, name string) (*Repo, error)
	ReadIssueByID(context context.Context, orgID string, repoID string, issueID string) (*Issue, error)
	ReadIssueByNumber(context context.Context, orgID string, repoID string, number int) (*Issue, error)
	ReadIssueCommentByID(context context.Context, orgID string, repoID string, issueID string, issueCommentID string) (*IssueComment, error)
	ReadIssuePipelineByNumber(context context.Context, orgID string, repoID string, number int) (*IssuePipeline, error)
	ReadLabelByID(context context.Context, orgID string, repoID string, labelID string) (*Label, error)
	ReadUserByID(context context.Context, userID string) (*User, error)
	ReadUserByLogin(context context.Context, login string) (*User, error)
	ReadPullRequestByID(context context.Context, orgID string, repoID string, prID string) (*PullRequest, error)
	ReadPullRequestCommentByID(context context.Context, orgID string, repoID string, prID string, prCommentID string) (*PullRequestComment, error)
	ReadPullRequestReviewByID(context context.Context, orgID string, repoID string, prID string, prReviewID string) (*PullRequestReview, error)
	ReadBotActivityByID(context context.Context, orgID string, repoID string) (*BotActivity, error)
	ReadMaintainerByID(context context.Context, orgID string, userID string) (*Maintainer, error)

	QueryMembersByOrg(context context.Context, orgID string, cb func(*Member) error) error
	QueryMaintainersByOrg(context context.Context, orgID string, cb func(*Maintainer) error) error
	QueryMaintainerInfo(context context.Context, maintainer *Maintainer) (*MaintainerInfo, error)
	QueryIssuesByRepo(context context.Context, orgID string, repoID string, cb func(*Issue) error) error
	QueryTestFlakeIssues(context context.Context, inactiveDays, createdDays int) ([]*Issue, error)
}
