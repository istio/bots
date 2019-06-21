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
	orgTable                = "Orgs"
	repoTable               = "Repos"
	repoCommentTable        = "RepoComments"
	userTable               = "Users"
	labelTable              = "Labels"
	issueTable              = "Issues"
	issueCommentTable       = "IssueComments"
	issuePipelineTable      = "IssuePipelines"
	pullRequestTable        = "PullRequests"
	pullRequestCommentTable = "PullRequestComments"
	pullRequestReviewTable  = "PullRequestReviews"
	memberTable             = "Members"
	botActivityTable        = "BotActivity"
	maintainerTable         = "Maintainers"
)

// All the DB indices we use
const (
	orgLoginIndex    = "OrgsLogin"
	repoNameIndex    = "ReposName"
	issueNumberIndex = "IssuesNumber"
	userLoginIndex   = "UsersLogin"
)

// Shape of the rows in the indices
type (
	orgLoginRow struct {
		OrgID string
		Login string
	}

	repoNameRow struct {
		OrgID  string
		RepoID string
		Name   string
	}

	issueNumberRow struct {
		OrgID   string
		RepoID  string
		IssueID string
		Number  int64
	}

	userLoginRow struct {
		UserID string
		Login  string
	}
)

// Holds the column names for each table or index in the database (filled in at startup)
var (
	orgColumns                []string
	orgLoginColumns           []string
	repoColumns               []string
	repoNameColumns           []string
	userColumns               []string
	userLoginColumns          []string
	labelColumns              []string
	issueColumns              []string
	issueNumberColumns        []string
	issueCommentColumns       []string
	issuePipelineColumns      []string
	pullRequestColumns        []string
	pullRequestCommentColumns []string
	pullRequestReviewColumns  []string
	botActivityColumns        []string
	maintainerColumns         []string
)

// Bunch of functions to from keys for the tables and indices in the DB

func orgKey(orgID string) spanner.Key {
	return spanner.Key{orgID}
}

func orgLoginKey(login string) spanner.Key {
	return spanner.Key{login}
}

func repoKey(orgID string, repoID string) spanner.Key {
	return spanner.Key{orgID, repoID}
}

func repoNameKey(orgID string, name string) spanner.Key {
	return spanner.Key{orgID, name}
}

func userKey(userID string) spanner.Key {
	return spanner.Key{userID}
}

func userLoginKey(login string) spanner.Key {
	return spanner.Key{login}
}

func labelKey(orgID string, repoID string, labelID string) spanner.Key {
	return spanner.Key{orgID, repoID, labelID}
}

func issueKey(orgID string, repoID string, issueID string) spanner.Key {
	return spanner.Key{orgID, repoID, issueID}
}

func issueNumberKey(repoID string, number int) spanner.Key {
	return spanner.Key{repoID, int64(number)}
}

func issueCommentKey(orgID string, repoID string, issueID string, commentID string) spanner.Key {
	return spanner.Key{orgID, repoID, issueID, commentID}
}

func issuePipelineKey(orgID string, repoID string, number int) spanner.Key {
	return spanner.Key{orgID, repoID, number}
}

func pullRequestKey(orgID string, repoID string, prID string) spanner.Key {
	return spanner.Key{orgID, repoID, prID}
}

func pullRequestCommentKey(orgID string, repoID string, prID string, commentID string) spanner.Key {
	return spanner.Key{orgID, repoID, prID, commentID}
}

func pullRequestReviewKey(orgID string, repoID string, prID string, reviewID string) spanner.Key {
	return spanner.Key{orgID, repoID, prID, reviewID}
}

func botActivityKey(orgID string, repoID string) spanner.Key {
	return spanner.Key{orgID, repoID}
}

func maintainerKey(orgID string, userID string) spanner.Key {
	return spanner.Key{orgID, userID}
}

func init() {
	orgColumns = getFields(storage.Org{})
	orgLoginColumns = getFields(orgLoginRow{})
	repoColumns = getFields(storage.Repo{})
	repoNameColumns = getFields(repoNameRow{})
	userColumns = getFields(storage.User{})
	userLoginColumns = getFields(userLoginRow{})
	labelColumns = getFields(storage.Label{})
	issueColumns = getFields(storage.Issue{})
	issueNumberColumns = getFields(issueNumberRow{})
	issueCommentColumns = getFields(storage.IssueComment{})
	issuePipelineColumns = getFields(storage.IssuePipeline{})
	pullRequestColumns = getFields(storage.PullRequest{})
	pullRequestReviewColumns = getFields(storage.PullRequestReview{})
	botActivityColumns = getFields(storage.BotActivity{})
	maintainerColumns = getFields(storage.Maintainer{})
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
