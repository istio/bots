// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.orgID/licenses/LICENSE-2.0
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
	orgByLoginCache        cache.ExpiringCache
	repoCache              cache.ExpiringCache
	issueCache             cache.ExpiringCache
	issueCommentCache      cache.ExpiringCache
	labelCache             cache.ExpiringCache
	userCache              cache.ExpiringCache
	userByLoginCache       cache.ExpiringCache
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
		orgByLoginCache:        cache.NewTTL(entryTTL, evictionInterval),
		repoCache:              cache.NewTTL(entryTTL, evictionInterval),
		issueCache:             cache.NewTTL(entryTTL, evictionInterval),
		issueCommentCache:      cache.NewTTL(entryTTL, evictionInterval),
		labelCache:             cache.NewTTL(entryTTL, evictionInterval),
		userCache:              cache.NewTTL(entryTTL, evictionInterval),
		userByLoginCache:       cache.NewTTL(entryTTL, evictionInterval),
		pullRequestCache:       cache.NewTTL(entryTTL, evictionInterval),
		pullRequestReviewCache: cache.NewTTL(entryTTL, evictionInterval),
	}
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadOrg(orgID string) (*storage.Org, error) {
	if value, ok := ghs.orgCache.Get(orgID); ok {
		return value.(*storage.Org), nil
	}

	result, err := ghs.store.ReadOrgByID(orgID)
	if err == nil {
		ghs.orgCache.Set(orgID, result)
		if result != nil {
			ghs.orgByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadOrgByLogin(login string) (*storage.Org, error) {
	if value, ok := ghs.orgByLoginCache.Get(login); ok {
		return value.(*storage.Org), nil
	}

	result, err := ghs.store.ReadOrgByLogin(login)
	if err == nil {
		if result != nil {
			ghs.orgCache.Set(result.OrgID, result)
			ghs.orgByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadRepo(orgID string, repoID string) (*storage.Repo, error) {
	if value, ok := ghs.repoCache.Get(repoID); ok {
		return value.(*storage.Repo), nil
	}

	result, err := ghs.store.ReadRepoByID(orgID, repoID)
	if err == nil {
		ghs.repoCache.Set(repoID, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadUser(userID string) (*storage.User, error) {
	if value, ok := ghs.userCache.Get(userID); ok {
		return value.(*storage.User), nil
	}

	result, err := ghs.store.ReadUserByID(userID)
	if err == nil {
		ghs.userCache.Set(userID, result)
		if result != nil {
			ghs.userByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadUserByLogin(login string) (*storage.User, error) {
	if value, ok := ghs.userByLoginCache.Get(login); ok {
		return value.(*storage.User), nil
	}

	result, err := ghs.store.ReadUserByLogin(login)
	if err == nil {
		if result != nil {
			ghs.userCache.Set(result.UserID, result)
			ghs.userByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadLabel(orgID string, repoID string, labelID string) (*storage.Label, error) {
	if value, ok := ghs.labelCache.Get(labelID); ok {
		return value.(*storage.Label), nil
	}

	result, err := ghs.store.ReadLabelByID(orgID, repoID, labelID)
	if err == nil {
		ghs.labelCache.Set(labelID, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadIssue(orgID string, repoID string, issueID string) (*storage.Issue, error) {
	if value, ok := ghs.issueCache.Get(issueID); ok {
		return value.(*storage.Issue), nil
	}

	result, err := ghs.store.ReadIssueByID(orgID, repoID, issueID)
	if err == nil {
		ghs.issueCache.Set(issueID, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadIssueComment(orgID string, repoID string, issueID string,
	issueCommentID string) (*storage.IssueComment, error) {
	if value, ok := ghs.issueCommentCache.Get(issueCommentID); ok {
		return value.(*storage.IssueComment), nil
	}

	result, err := ghs.store.ReadIssueCommentByID(orgID, repoID, issueID, issueCommentID)
	if err == nil {
		ghs.issueCommentCache.Set(issueCommentID, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadPullRequest(orgID string, repoID string, prID string) (*storage.PullRequest, error) {
	if value, ok := ghs.pullRequestCache.Get(prID); ok {
		return value.(*storage.PullRequest), nil
	}

	result, err := ghs.store.ReadPullRequestByID(orgID, repoID, prID)
	if err == nil {
		ghs.pullRequestReviewCache.Set(prID, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (ghs *GitHubState) ReadPullRequestReview(orgID string, repoID string, issueID string,
	prReviewID string) (*storage.PullRequestReview, error) {
	if value, ok := ghs.pullRequestReviewCache.Get(prReviewID); ok {
		return value.(*storage.PullRequestReview), nil
	}

	result, err := ghs.store.ReadPullRequestReviewByID(orgID, repoID, issueID, prReviewID)
	if err == nil {
		ghs.pullRequestReviewCache.Set(prReviewID, result)
	}

	return result, err
}
