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

package gh

import (
	"fmt"

	"istio.io/bots/policybot/pkg/storage"

	google_spanner "cloud.google.com/go/spanner"
)

func (ghs *GitHubState) ReadOrg(org string) (*storage.Org, error) {
	if value, ok := ghs.cache.Get(org); ok {
		return value.(*storage.Org), nil
	}

	return ghs.store.ReadOrgByID(org)
}

func (ghs *GitHubState) ReadRepo(org string, repo string) (*storage.Repo, error) {
	if value, ok := ghs.cache.Get(repo); ok {
		return value.(*storage.Repo), nil
	}

	return ghs.store.ReadRepoByID(org, repo)
}

func (ghs *GitHubState) ReadUser(user string) (*storage.User, error) {
	if value, ok := ghs.cache.Get(user); ok {
		return value.(*storage.User), nil
	}

	return ghs.store.ReadUserByID(user)
}

func (ghs *GitHubState) ReadLabel(org string, repo string, label string) (*storage.Label, error) {
	if value, ok := ghs.cache.Get(label); ok {
		return value.(*storage.Label), nil
	}

	return ghs.store.ReadLabelByID(org, repo, label)
}

func (ghs *GitHubState) ReadIssue(org string, repo string, issue string) (*storage.Issue, error) {
	if value, ok := ghs.cache.Get(issue); ok {
		return value.(*storage.Issue), nil
	}

	return ghs.store.ReadIssueByID(org, repo, issue)
}

func (ghs *GitHubState) ReadIssueComment(org string, repo string, issue string,
	issueComment string) (*storage.IssueComment, error) {
	if value, ok := ghs.cache.Get(issueComment); ok {
		return value.(*storage.IssueComment), nil
	}

	return ghs.store.ReadIssueCommentByID(org, repo, issue, issueComment)
}

func (ghs *GitHubState) ReadIssueBySQL(sql string) ([]*storage.Issue, error) {
	issues := []*storage.Issue{}
	getIssue := func(row *google_spanner.Row) error {
		issue := &storage.Issue{}
		err := row.Columns(&issue.OrgID, &issue.IssueID, &issue.Title, &issue.UpdatedAt)
		if err != nil {
			fmt.Println("jianfeih debug error in fetching the issue", err)
			return err
		}
		fmt.Println("jianfeih debug issue %v", issue)
		issues = append(issues, &issue)
		return nil
	}
	if err := ghs.store.ReadIssueBySQL(sql, getIssue); err != nil {
		return nil, err
	}
	return issues, nil
}
