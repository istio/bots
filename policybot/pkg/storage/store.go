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
	WritePullRequestReviewComments(context context.Context, prComments []*PullRequestReviewComment) error
	WritePullRequestReviews(context context.Context, prReviews []*PullRequestReview) error
	WriteUsers(context context.Context, users []*User) error
	WriteLabels(context context.Context, labels []*Label) error
	WriteAllMembers(context context.Context, members []*Member) error
	WriteAllMaintainers(context context.Context, maintainers []*Maintainer) error
	WriteBotActivities(context context.Context, activities []*BotActivity) error
	WriteTestResults(context context.Context, testResults []*TestResult) error
	WriteIssueEvents(context context.Context, events []*IssueEvent) error
	WriteIssueCommentEvents(context context.Context, events []*IssueCommentEvent) error
	WritePullRequestEvents(context context.Context, events []*PullRequestEvent) error
	WritePullRequestReviewCommentEvents(context context.Context, events []*PullRequestReviewCommentEvent) error
	WritePullRequestReviewEvents(context context.Context, events []*PullRequestReviewEvent) error
	WriteRepoCommentEvents(context context.Context, events []*RepoCommentEvent) error

	UpdateBotActivity(context context.Context, orgLogin string, repoName string, cb func(*BotActivity) error) error

	ReadOrg(context context.Context, orgLogin string) (*Org, error)
	ReadRepo(context context.Context, orgLogin string, repoName string) (*Repo, error)
	ReadIssue(context context.Context, orgLogin string, repoName string, number int) (*Issue, error)
	ReadIssueComment(context context.Context, orgLogin string, repoName string, issueNumber int, issueCommentID int) (*IssueComment, error)
	ReadIssuePipeline(context context.Context, orgLogin string, repoName string, issueNumber int) (*IssuePipeline, error)
	ReadLabel(context context.Context, orgLogin string, repoName string, labelName string) (*Label, error)
	ReadUser(context context.Context, userLogin string) (*User, error)
	ReadPullRequest(context context.Context, orgLogin string, repoName string, prNumber int) (*PullRequest, error)
	ReadPullRequestReviewComment(context context.Context, orgLogin string, repoName string, prNumber int, prCommentID int) (*PullRequestReviewComment, error)
	ReadPullRequestReview(context context.Context, orgLogin string, repoName string, prNumber int, prReviewID int) (*PullRequestReview, error)
	ReadBotActivity(context context.Context, orgLogin string, repoName string) (*BotActivity, error)
	ReadMaintainer(context context.Context, orgLogin string, userLogin string) (*Maintainer, error)
	ReadMember(context context.Context, orgLogin string, userLogin string) (*Member, error)
	ReadTestResult(context context.Context, orgLogin string, repoName string, testName string, pullRequestNumber int64, runNumber int64) (*TestResult, error)

	QueryMembersByOrg(context context.Context, orgLogin string, cb func(*Member) error) error
	QueryMaintainersByOrg(context context.Context, orgLogin string, cb func(*Maintainer) error) error
	QueryMaintainerActivity(context context.Context, maintainer *Maintainer) (*ActivityInfo, error)
	QueryMemberActivity(context context.Context, member *Member, repoNames []string) (*ActivityInfo, error)
	QueryIssuesByRepo(context context.Context, orgLogin string, repoName string, cb func(*Issue) error) error
	QueryTestResultByPrNumber(context context.Context, orgLogin string, repoName string, pullRequestNumber int64, cb func(*TestResult) error) error
	QueryTestResultByUndone(context context.Context, orgLogin string, repoName string, cb func(*TestResult) error) error
	QueryAllTestResults(context context.Context, orgLogin string, repoName string, cb func(*TestResult) error) error
	QueryTestResultByTestName(context context.Context, orgLogin string, repoName string, testName string, cb func(*TestResult) error) error

	// TODO: needs to be org-specific and/or repo-specific, needs to use a callback instead of returning a slice
	QueryTestFlakeIssues(context context.Context, inactiveDays, createdDays int) ([]*Issue, error)
}
