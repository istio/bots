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

type Store interface {
	io.Closer
	WriteOrgs(orgs []*Org) error
	WriteRepos(repos []*Repo) error
	WriteIssues(issues []*Issue) error
	WriteIssueComments(issueComments []*IssueComment) error
	WriteUsers(users []*User) error
	WriteLabels(labels []*Label) error

	ReadOrgByID(org string) (*Org, error)
	ReadOrgByLogin(login string) (*Org, error)
	ReadRepoByID(org string, repo string) (*Repo, error)
	ReadRepoByName(org string, name string) (*Repo, error)
	ReadIssueByID(org string, repo string, issue string) (*Issue, error)
	ReadIssueByNumber(org string, repo string, number int) (*Issue, error)
	ReadIssueCommentByID(org string, repo string, issue string, issueComment string) (*IssueComment, error)
	ReadLabelByID(org string, repo string, label string) (*Label, error)
	ReadUserByID(user string) (*User, error)

	//	FindUnengagedIssues(repo *gh.Repo, cb func(issue *gh.Issue)) error

	RecordTestNagAdded(repo string) error
	RecordTestNagRemoved(repo string) error
}
