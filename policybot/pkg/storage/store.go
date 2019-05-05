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

	"github.com/google/go-github/v25/github"
)

type Store interface {
	io.Closer
	WriteOrgAndRepos(org *github.Organization, repos []*github.Repository) error
	WriteIssueAndComments(org *github.Organization, repo *github.Repository, issue *github.Issue, comments []*github.IssueComment) error
	WriteUsers(user []*github.User) error
	ReadIssue(org *github.Organization, repo *github.Repository, id string) (*github.Issue, error)
}
