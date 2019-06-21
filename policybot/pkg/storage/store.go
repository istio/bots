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
	WriteIssuePipelines(issueData []*IssuePipeline) error
	WritePullRequests(prs []*PullRequest) error
	WritePullRequestComments(prComments []*PullRequestComment) error
	WritePullRequestReviews(prReviews []*PullRequestReview) error
	WriteUsers(users []*User) error
	WriteLabels(labels []*Label) error
	WriteAllMembers(members []*Member) error
	WriteAllMaintainers(maintainers []*Maintainer) error
	WriteBotActivities(activities []*BotActivity) error
	WriteTestFlakeForPr(testFlakesForPr []*TestFlakeForPr) error
	WriteTestFlakes(flakes []*TestFlake) error
	WriteFlakeOccurrences(flakeOccurrences []*FlakeOccurrence) error

	ReadOrgByID(orgID string) (*Org, error)
	ReadOrgByLogin(login string) (*Org, error)
	ReadRepoByID(orgID string, repoID string) (*Repo, error)
	ReadRepoByName(orgID string, name string) (*Repo, error)
	ReadIssueByID(orgID string, repoID string, issueID string) (*Issue, error)
	ReadIssueByNumber(orgID string, repoID string, number int) (*Issue, error)
	ReadIssueCommentByID(orgID string, repoID string, issueID string, issueCommentID string) (*IssueComment, error)
	ReadIssuePipelineByNumber(orgID string, repoID string, number int) (*IssuePipeline, error)
	ReadLabelByID(orgID string, repoID string, labelID string) (*Label, error)
	ReadUserByID(userID string) (*User, error)
	ReadUserByLogin(login string) (*User, error)
	ReadPullRequestByID(orgID string, repoID string, prID string) (*PullRequest, error)
	ReadPullRequestCommentByID(orgID string, repoID string, prID string, prCommentID string) (*PullRequestComment, error)
	ReadPullRequestReviewByID(orgID string, repoID string, prID string, prReviewID string) (*PullRequestReview, error)
	ReadBotActivityByID(orgID string, repoID string) (*BotActivity, error)
	ReadTestFlakeForPrByName(orgID string, testName string, prNum int64, runNum int64) (*TestFlakeForPr, error)
	ReadTestFlakeByName(orgID string, repoID string, branchName string, testName string) (*TestFlake, error)

	QueryMembersByOrg(orgID string, cb func(*Member) error) error
	QueryMaintainersByOrg(orgID string, cb func(*Maintainer) error) error
	QueryMaintainerInfo(maintainer *Maintainer) (*MaintainerInfo, error)
	QueryIssuesByRepo(orgID string, repoID string, cb func(*Issue) error) error
	QueryTestFlakeForPrByTestName(testName string, cb func(*TestFlake) error) error
	QueryTestFlakeForPrByPrNumber(prNum int64, cb func(*TestFlake) error) error

	EnumUsers(cb func(*User) bool) error
	QueryAllUsers(cb func(*User) error) error
	QueryFlakeOccurrencesByFlake(orgID string, repoID string, branchName string, testName string, cb func(*FlakeOccurrence) error) error
}
