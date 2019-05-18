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

const (
	orgTable          = "Orgs"
	repoTable         = "Repos"
	repoStatsTable    = "RepoStats"
	userTable         = "Users"
	labelTable        = "Labels"
	issueTable        = "Issues"
	issueCommentTable = "IssueComments"

	orgLoginIndex    = "OrgsLogin"
	repoNameIndex    = "ReposName"
	issueNumberIndex = "IssuesNumber"
)

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

	repoStatsRow struct {
		RepoID      string
		NagsAdded   int64
		NagsRemoved int64
	}

	issueNumberRow struct {
		OrgID   string
		RepoID  string
		IssueID string
		Number  int64
	}
)

var (
	orgColumns          []string
	orgLoginColumns     []string
	repoColumns         []string
	repoNameColumns     []string
	repoStatsColumns    []string
	userColumns         []string
	labelColumns        []string
	issueColumns        []string
	issueNumberColumns  []string
	issueCommentColumns []string
)

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

func repoStatsKey(repoID string) spanner.Key {
	return spanner.Key{repoID}
}

func userKey(userID string) spanner.Key {
	return spanner.Key{userID}
}

func labelKey(orgID string, repoID string, labelID string) spanner.Key {
	return spanner.Key{orgID, repoID, labelID}
}

func issueKey(orgID string, repoID string, issueID string) spanner.Key {
	return spanner.Key{orgID, repoID, issueID}
}

func issueNumberKey(orgID string, repoID string, number int) spanner.Key {
	return spanner.Key{orgID, repoID, int64(number)}
}

func issueCommentKey(orgID string, repoID string, issueID string, commentID string) spanner.Key {
	return spanner.Key{orgID, repoID, issueID, commentID}
}

func init() {
	orgColumns = getFields(storage.Org{})
	orgLoginColumns = getFields(orgLoginRow{})
	repoColumns = getFields(storage.Repo{})
	repoNameColumns = getFields(repoNameRow{})
	repoStatsColumns = getFields(repoStatsRow{})
	userColumns = getFields(storage.User{})
	labelColumns = getFields(storage.Label{})
	issueColumns = getFields(storage.Issue{})
	issueNumberColumns = getFields(issueNumberRow{})
	issueCommentColumns = getFields(storage.IssueComment{})
}

func getFields(o interface{}) []string {
	t := reflect.TypeOf(o)
	result := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		result[i] = t.Field(i).Name
	}

	return result
}
