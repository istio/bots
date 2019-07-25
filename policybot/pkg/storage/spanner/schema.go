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

package spanner

import (
	"reflect"

	"cloud.google.com/go/spanner"

	"istio.io/bots/policybot/pkg/storage"
)

// Details of the DB schema internal to the Spanner-based implementation

// All the DB tables we use
const (
	orgTable                           = "Orgs"
	repoTable                          = "Repos"
	repoCommentTable                   = "RepoComments"
	userTable                          = "Users"
	labelTable                         = "Labels"
	issueTable                         = "Issues"
	issueCommentTable                  = "IssueComments"
	issuePipelineTable                 = "IssuePipelines"
	pullRequestTable                   = "PullRequests"
	pullRequestReviewCommentTable      = "PullRequestReviewComments"
	pullRequestReviewTable             = "PullRequestReviews"
	memberTable                        = "Members"
	botActivityTable                   = "BotActivity"
	maintainerTable                    = "Maintainers"
	issueEventTable                    = "IssueEvents"
	issueCommentEventTable             = "IssueCommentEvents"
	pullRequestEventTable              = "PullRequestEvents"
	pullRequestReviewCommentEventTable = "PullRequestReviewCommentEvents"
	pullRequestReviewEventTable        = "PullRequestReviewEvents"
	repoCommentEventTable              = "RepoCommentEvents"
	testResultTable                    = "TestResults"
)

// Holds the column names for each table or index in the database (filled in at startup)
var (
	orgColumns                      []string
	repoColumns                     []string
	userColumns                     []string
	labelColumns                    []string
	issueColumns                    []string
	issueCommentColumns             []string
	issuePipelineColumns            []string
	pullRequestColumns              []string
	pullRequestReviewCommentColumns []string
	pullRequestReviewColumns        []string
	botActivityColumns              []string
	maintainerColumns               []string
	memberColumns                   []string
	testResultColumns               []string
)

// Bunch of functions to from keys for the tables and indices in the DB

func orgKey(orgLogin string) spanner.Key {
	return spanner.Key{orgLogin}
}

func repoKey(orgLogin string, repoName string) spanner.Key {
	return spanner.Key{orgLogin, repoName}
}

func userKey(userLogin string) spanner.Key {
	return spanner.Key{userLogin}
}

func labelKey(orgLogin string, repoName string, labelName string) spanner.Key {
	return spanner.Key{orgLogin, repoName, labelName}
}

func issueKey(orgLogin string, repoName string, issueNumber int64) spanner.Key {
	return spanner.Key{orgLogin, repoName, issueNumber}
}

func issueCommentKey(orgLogin string, repoName string, issueNumber int64, commentID int64) spanner.Key {
	return spanner.Key{orgLogin, repoName, issueNumber, commentID}
}

func issuePipelineKey(orgLogin string, repoName string, issueNumber int64) spanner.Key {
	return spanner.Key{orgLogin, repoName, issueNumber}
}

func pullRequestKey(orgLogin string, repoName string, prNumber int64) spanner.Key {
	return spanner.Key{orgLogin, repoName, prNumber}
}

func pullRequestReviewCommentKey(orgLogin string, repoName string, prNumber int64, commentID int64) spanner.Key {
	return spanner.Key{orgLogin, repoName, prNumber, commentID}
}

func pullRequestReviewKey(orgLogin string, repoName string, prNumber int64, reviewID int64) spanner.Key {
	return spanner.Key{orgLogin, repoName, prNumber, reviewID}
}

func botActivityKey(orgLogin string, repoName string) spanner.Key {
	return spanner.Key{orgLogin, repoName}
}

func maintainerKey(orgLogin string, userLogin string) spanner.Key {
	return spanner.Key{orgLogin, userLogin}
}

func memberKey(orgLogin string, userLogin string) spanner.Key {
	return spanner.Key{orgLogin, userLogin}
}

func testResultKey(orgLogin string, repoName string, testName string, prNum int64, runNumber int64) spanner.Key {
	return spanner.Key{orgLogin, repoName, testName, prNum, runNumber}
}

func init() {
	orgColumns = getFields(storage.Org{})
	repoColumns = getFields(storage.Repo{})
	userColumns = getFields(storage.User{})
	labelColumns = getFields(storage.Label{})
	issueColumns = getFields(storage.Issue{})
	issueCommentColumns = getFields(storage.IssueComment{})
	issuePipelineColumns = getFields(storage.IssuePipeline{})
	pullRequestColumns = getFields(storage.PullRequest{})
	pullRequestReviewColumns = getFields(storage.PullRequestReview{})
	botActivityColumns = getFields(storage.BotActivity{})
	maintainerColumns = getFields(storage.Maintainer{})
	memberColumns = getFields(storage.Member{})
	testResultColumns = getFields(storage.TestResult{})
}

// Produces a string array representing all the fields in the input object
func getFields(o interface{}) []string {
	t := reflect.TypeOf(o)
	result := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		result[i] = t.Field(i).Name
	}

	return result
}
