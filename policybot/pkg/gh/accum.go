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
	"istio.io/bots/policybot/pkg/storage"
)

// Accumulates objects in anticipation of bulk non-transactional commits to the cache and to the DB.
type Accumulator struct {
	ghs                *GitHubState
	objects            map[string]interface{}
	labels             []*storage.Label
	users              []*storage.User
	orgs               []*storage.Org
	repos              []*storage.Repo
	issues             []*storage.Issue
	issueComments      []*storage.IssueComment
	pullRequests       []*storage.PullRequest
	pullRequestReviews []*storage.PullRequestReview
}

func (ghs *GitHubState) NewAccumulator() *Accumulator {
	return &Accumulator{
		ghs:                ghs,
		objects:            make(map[string]interface{}),
		labels:             make([]*storage.Label, 0),
		users:              make([]*storage.User, 0),
		orgs:               make([]*storage.Org, 0),
		repos:              make([]*storage.Repo, 0),
		issues:             make([]*storage.Issue, 0),
		issueComments:      make([]*storage.IssueComment, 0),
		pullRequests:       make([]*storage.PullRequest, 0),
		pullRequestReviews: make([]*storage.PullRequestReview, 0),
	}
}

// Reset clears all state accumulated so the accumulator can be used anew
func (a *Accumulator) Reset() {
	a.labels = a.labels[:0]
	a.users = a.users[:0]
	a.orgs = a.orgs[:0]
	a.repos = a.repos[:0]
	a.issues = a.issues[:0]
	a.issueComments = a.issueComments[:0]
	a.pullRequests = a.pullRequests[:0]
	a.pullRequestReviews = a.pullRequestReviews[:0]

	for k := range a.objects {
		delete(a.objects, k)
	}
}
