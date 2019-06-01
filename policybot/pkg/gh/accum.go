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
	labels             map[string]*storage.Label
	users              map[string]*storage.User
	orgs               map[string]*storage.Org
	repos              map[string]*storage.Repo
	issues             map[string]*storage.Issue
	issueComments      map[string]*storage.IssueComment
	pullRequests       map[string]*storage.PullRequest
	pullRequestReviews map[string]*storage.PullRequestReview
	members            map[string]*storage.Member
}

func (ghs *GitHubState) NewAccumulator() *Accumulator {
	return &Accumulator{
		ghs:                ghs,
		labels:             make(map[string]*storage.Label),
		users:              make(map[string]*storage.User),
		orgs:               make(map[string]*storage.Org),
		repos:              make(map[string]*storage.Repo),
		issues:             make(map[string]*storage.Issue),
		issueComments:      make(map[string]*storage.IssueComment),
		pullRequests:       make(map[string]*storage.PullRequest),
		pullRequestReviews: make(map[string]*storage.PullRequestReview),
		members:            make(map[string]*storage.Member),
	}
}

// Reset clears all state accumulated so the accumulator can be used anew
func (a *Accumulator) Reset() {
	for k := range a.labels {
		delete(a.labels, k)
	}

	for k := range a.users {
		delete(a.users, k)
	}

	for k := range a.orgs {
		delete(a.orgs, k)
	}

	for k := range a.repos {
		delete(a.repos, k)
	}

	for k := range a.issues {
		delete(a.issues, k)
	}

	for k := range a.issueComments {
		delete(a.issueComments, k)
	}

	for k := range a.pullRequests {
		delete(a.pullRequests, k)
	}

	for k := range a.pullRequestReviews {
		delete(a.pullRequestReviews, k)
	}

	for k := range a.members {
		delete(a.members, k)
	}
}
