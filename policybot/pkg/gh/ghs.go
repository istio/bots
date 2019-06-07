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
	store                  storage.Store
	orgCache               cache.ExpiringCache
	repoCache              cache.ExpiringCache
	issueCache             cache.ExpiringCache
	issueCommentCache      cache.ExpiringCache
	labelCache             cache.ExpiringCache
	userCache              cache.ExpiringCache
	pullRequestCache       cache.ExpiringCache
	pullRequestReviewCache cache.ExpiringCache
}

func NewGitHubState(store storage.Store, entryTTL time.Duration) *GitHubState {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if entryTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = entryTTL / 2
	}

	return &GitHubState{
		store:                  store,
		orgCache:               cache.NewTTL(entryTTL, evictionInterval),
		repoCache:              cache.NewTTL(entryTTL, evictionInterval),
		issueCache:             cache.NewTTL(entryTTL, evictionInterval),
		issueCommentCache:      cache.NewTTL(entryTTL, evictionInterval),
		labelCache:             cache.NewTTL(entryTTL, evictionInterval),
		userCache:              cache.NewTTL(entryTTL, evictionInterval),
		pullRequestCache:       cache.NewTTL(entryTTL, evictionInterval),
		pullRequestReviewCache: cache.NewTTL(entryTTL, evictionInterval),
	}
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadOrg(org string) (*storage.Org, error) {
	if value, ok := ghs.orgCache.Get(org); ok {
		return value.(*storage.Org), nil
	}

	result, err := ghs.store.ReadOrgByID(org)
	if err == nil {
		ghs.orgCache.Set(org, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadRepo(org string, repo string) (*storage.Repo, error) {
	if value, ok := ghs.repoCache.Get(repo); ok {
		return value.(*storage.Repo), nil
	}

	result, err := ghs.store.ReadRepoByID(org, repo)
	if err == nil {
		ghs.repoCache.Set(repo, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadUser(user string) (*storage.User, error) {
	if value, ok := ghs.userCache.Get(user); ok {
		return value.(*storage.User), nil
	}

	result, err := ghs.store.ReadUserByID(user)
	if err == nil {
		ghs.userCache.Set(user, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadLabel(org string, repo string, label string) (*storage.Label, error) {
	if value, ok := ghs.labelCache.Get(label); ok {
		return value.(*storage.Label), nil
	}

	result, err := ghs.store.ReadLabelByID(org, repo, label)
	if err == nil {
		ghs.labelCache.Set(label, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadIssue(org string, repo string, issue string) (*storage.Issue, error) {
	if value, ok := ghs.issueCache.Get(issue); ok {
		return value.(*storage.Issue), nil
	}

	result, err := ghs.store.ReadIssueByID(org, repo, issue)
	if err == nil {
		ghs.issueCache.Set(issue, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadIssueComment(org string, repo string, issue string,
	issueComment string) (*storage.IssueComment, error) {
	if value, ok := ghs.issueCommentCache.Get(issueComment); ok {
		return value.(*storage.IssueComment), nil
	}

	result, err := ghs.store.ReadIssueCommentByID(org, repo, issue, issueComment)
	if err == nil {
		ghs.issueCommentCache.Set(issueComment, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadPullRequest(org string, repo string, issue string) (*storage.PullRequest, error) {
	if value, ok := ghs.pullRequestCache.Get(issue); ok {
		return value.(*storage.PullRequest), nil
	}

	result, err := ghs.store.ReadPullRequestByID(org, repo, issue)
	if err == nil {
		ghs.pullRequestReviewCache.Set(issue, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadPullRequestReview(org string, repo string, issue string,
	prReview string) (*storage.PullRequestReview, error) {
	if value, ok := ghs.pullRequestReviewCache.Get(prReview); ok {
		return value.(*storage.PullRequestReview), nil
	}

	result, err := ghs.store.ReadPullRequestReviewByID(org, repo, issue, prReview)
	if err == nil {
		ghs.pullRequestReviewCache.Set(prReview, result)
	}

	return result, err
}

// ReadTestFlakyIssues returns issue based on the SQL query.
func (ghs *GitHubState) ReadTestFlakyIssues(inactiveDays, createdDays int) ([]*storage.Issue, error) {
	return ghs.store.ReadTestFlakyIssues(inactiveDays, createdDays)
}
