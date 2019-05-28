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

// Package gh exposes a GitHub persistent object store.
package gh

import (
	"time"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/cache"
)

// Cached access over our GitHub object store.
type GitHubState struct {
	cache cache.ExpiringCache
	store storage.Store
}

// Issues and pull requests have the same node id (as per GitHub's API). Since we want to
// store both objects in the same cache & map using the node id as key, we apply this suffix
// to the PR keys to distinguish them from issues. The use of | in here is so that it is
// guaranteed unique from other node ids, since incoming node ids are base64 encoded and so
// don't contain pipe symbols.
const pullRequestIDSuffix = "|PR"

func NewGitHubState(store storage.Store, entryTTL time.Duration) *GitHubState {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if entryTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = entryTTL / 2
	}

	return &GitHubState{
		cache: cache.NewTTL(entryTTL, evictionInterval),
		store: store,
	}
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadOrg(org string) (*storage.Org, error) {
	if value, ok := ghs.cache.Get(org); ok {
		return value.(*storage.Org), nil
	}

	return ghs.store.ReadOrgByID(org)
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadRepo(org string, repo string) (*storage.Repo, error) {
	if value, ok := ghs.cache.Get(repo); ok {
		return value.(*storage.Repo), nil
	}

	return ghs.store.ReadRepoByID(org, repo)
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadUser(user string) (*storage.User, error) {
	if value, ok := ghs.cache.Get(user); ok {
		return value.(*storage.User), nil
	}

	return ghs.store.ReadUserByID(user)
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadLabel(org string, repo string, label string) (*storage.Label, error) {
	if value, ok := ghs.cache.Get(label); ok {
		return value.(*storage.Label), nil
	}

	return ghs.store.ReadLabelByID(org, repo, label)
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadIssue(org string, repo string, issue string) (*storage.Issue, error) {
	if value, ok := ghs.cache.Get(issue); ok {
		return value.(*storage.Issue), nil
	}

	return ghs.store.ReadIssueByID(org, repo, issue)
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadIssueComment(org string, repo string, issue string,
	issueComment string) (*storage.IssueComment, error) {
	if value, ok := ghs.cache.Get(issueComment); ok {
		return value.(*storage.IssueComment), nil
	}

	return ghs.store.ReadIssueCommentByID(org, repo, issue, issueComment)
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadPullRequest(org string, repo string, issue string) (*storage.PullRequest, error) {
	if value, ok := ghs.cache.Get(issue + pullRequestIDSuffix); ok {
		return value.(*storage.PullRequest), nil
	}

	return ghs.store.ReadPullRequestByID(org, repo, issue)
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadPullRequestReview(org string, repo string, issue string,
	prReview string) (*storage.PullRequestReview, error) {
	if value, ok := ghs.cache.Get(prReview); ok {
		return value.(*storage.PullRequestReview), nil
	}

	return ghs.store.ReadPullRequestReviewByID(org, repo, issue, prReview)
}
