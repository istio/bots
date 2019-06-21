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

// Package cache exposes a caching layer on top of the core store abstraction.
package cache

import (
	"context"
	"strconv"
	"time"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/cache"
)

// Cached access over our database.
type Cache struct {
	store                   storage.Store
	orgCache                cache.ExpiringCache
	orgByLoginCache         cache.ExpiringCache
	repoCache               cache.ExpiringCache
	repoByNameCache         cache.ExpiringCache
	issueCache              cache.ExpiringCache
	issueCommentCache       cache.ExpiringCache
	labelCache              cache.ExpiringCache
	userCache               cache.ExpiringCache
	userByLoginCache        cache.ExpiringCache
	pullRequestCache        cache.ExpiringCache
	pullRequestCommentCache cache.ExpiringCache
	pullRequestReviewCache  cache.ExpiringCache
	pipelineCache           cache.ExpiringCache
	maintainerCache         cache.ExpiringCache
	repoCommentCache        cache.ExpiringCache
}

func New(store storage.Store, entryTTL time.Duration) *Cache {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if entryTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = entryTTL / 2
	}

	return &Cache{
		store:                   store,
		orgCache:                cache.NewTTL(entryTTL, evictionInterval),
		orgByLoginCache:         cache.NewTTL(entryTTL, evictionInterval),
		repoCache:               cache.NewTTL(entryTTL, evictionInterval),
		repoByNameCache:         cache.NewTTL(entryTTL, evictionInterval),
		issueCache:              cache.NewTTL(entryTTL, evictionInterval),
		issueCommentCache:       cache.NewTTL(entryTTL, evictionInterval),
		labelCache:              cache.NewTTL(entryTTL, evictionInterval),
		userCache:               cache.NewTTL(entryTTL, evictionInterval),
		userByLoginCache:        cache.NewTTL(entryTTL, evictionInterval),
		pullRequestCache:        cache.NewTTL(entryTTL, evictionInterval),
		pullRequestCommentCache: cache.NewTTL(entryTTL, evictionInterval),
		pullRequestReviewCache:  cache.NewTTL(entryTTL, evictionInterval),
		pipelineCache:           cache.NewTTL(entryTTL, evictionInterval),
		maintainerCache:         cache.NewTTL(entryTTL, evictionInterval),
		repoCommentCache:        cache.NewTTL(entryTTL, evictionInterval),
	}
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadOrg(context context.Context, orgID string) (*storage.Org, error) {
	if value, ok := c.orgCache.Get(orgID); ok {
		return value.(*storage.Org), nil
	}

	result, err := c.store.ReadOrgByID(context, orgID)
	if err == nil {
		c.orgCache.Set(orgID, result)
		if result != nil {
			c.orgByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadOrgByLogin(context context.Context, login string) (*storage.Org, error) {
	if value, ok := c.orgByLoginCache.Get(login); ok {
		return value.(*storage.Org), nil
	}

	result, err := c.store.ReadOrgByLogin(context, login)
	if err == nil {
		if result != nil {
			c.orgCache.Set(result.OrgID, result)
			c.orgByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadRepo(context context.Context, orgID string, repoID string) (*storage.Repo, error) {
	if value, ok := c.repoCache.Get(repoID); ok {
		return value.(*storage.Repo), nil
	}

	result, err := c.store.ReadRepoByID(context, orgID, repoID)
	if err == nil {
		c.repoCache.Set(repoID, result)
		if result != nil {
			c.repoByNameCache.Set(orgID+result.Name, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadRepoByName(context context.Context, orgID string, repo string) (*storage.Repo, error) {
	key := orgID + repo
	if value, ok := c.repoByNameCache.Get(key); ok {
		return value.(*storage.Repo), nil
	}

	result, err := c.store.ReadRepoByName(context, orgID, repo)
	if err == nil {
		c.repoByNameCache.Set(key, result)
		if result != nil {
			c.repoCache.Set(result.RepoID, result)
		}
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteRepoComments(context context.Context, comments []*storage.RepoComment) error {
	err := c.store.WriteRepoComments(context, comments)
	if err == nil {
		for _, comment := range comments {
			c.repoCommentCache.Set(comment.CommentID, comment)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadUser(context context.Context, userID string) (*storage.User, error) {
	if value, ok := c.userCache.Get(userID); ok {
		return value.(*storage.User), nil
	}

	result, err := c.store.ReadUserByID(context, userID)
	if err == nil {
		c.userCache.Set(userID, result)
		if result != nil {
			c.userByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadUserByLogin(context context.Context, login string) (*storage.User, error) {
	if value, ok := c.userByLoginCache.Get(login); ok {
		return value.(*storage.User), nil
	}

	result, err := c.store.ReadUserByLogin(context, login)
	if err == nil {
		if result != nil {
			c.userCache.Set(result.UserID, result)
			c.userByLoginCache.Set(result.Login, result)
		}
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteUsers(context context.Context, users []*storage.User) error {
	err := c.store.WriteUsers(context, users)
	if err == nil {
		for _, user := range users {
			c.userCache.Set(user.UserID, user)
			c.userByLoginCache.Set(user.Login, user)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadLabel(context context.Context, orgID string, repoID string, labelID string) (*storage.Label, error) {
	if value, ok := c.labelCache.Get(labelID); ok {
		return value.(*storage.Label), nil
	}

	result, err := c.store.ReadLabelByID(context, orgID, repoID, labelID)
	if err == nil {
		c.labelCache.Set(labelID, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadIssue(context context.Context, orgID string, repoID string, issueID string) (*storage.Issue, error) {
	if value, ok := c.issueCache.Get(issueID); ok {
		return value.(*storage.Issue), nil
	}

	result, err := c.store.ReadIssueByID(context, orgID, repoID, issueID)
	if err == nil {
		c.issueCache.Set(issueID, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteIssues(context context.Context, issues []*storage.Issue) error {
	err := c.store.WriteIssues(context, issues)
	if err == nil {
		for _, issue := range issues {
			c.issueCache.Set(issue.IssueID, issue)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadIssueComment(context context.Context, orgID string, repoID string, issueID string,
	issueCommentID string) (*storage.IssueComment, error) {
	if value, ok := c.issueCommentCache.Get(issueCommentID); ok {
		return value.(*storage.IssueComment), nil
	}

	result, err := c.store.ReadIssueCommentByID(context, orgID, repoID, issueID, issueCommentID)
	if err == nil {
		c.issueCommentCache.Set(issueCommentID, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteIssueComments(context context.Context, issueComments []*storage.IssueComment) error {
	err := c.store.WriteIssueComments(context, issueComments)
	if err == nil {
		for _, comment := range issueComments {
			c.issueCommentCache.Set(comment.IssueCommentID, comment)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadPullRequest(context context.Context, orgID string, repoID string, prID string) (*storage.PullRequest, error) {
	if value, ok := c.pullRequestCache.Get(prID); ok {
		return value.(*storage.PullRequest), nil
	}

	result, err := c.store.ReadPullRequestByID(context, orgID, repoID, prID)
	if err == nil {
		c.pullRequestReviewCache.Set(prID, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WritePullRequests(context context.Context, prs []*storage.PullRequest) error {
	err := c.store.WritePullRequests(context, prs)
	if err == nil {
		for _, pr := range prs {
			c.pullRequestCache.Set(pr.PullRequestID, pr)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadPullRequestComment(context context.Context, orgID string, repoID string, prID string,
	prCommentID string) (*storage.PullRequestComment, error) {
	if value, ok := c.pullRequestCommentCache.Get(prCommentID); ok {
		return value.(*storage.PullRequestComment), nil
	}

	result, err := c.store.ReadPullRequestCommentByID(context, orgID, repoID, prID, prCommentID)
	if err == nil {
		c.pullRequestCommentCache.Set(prCommentID, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WritePullRequestComments(context context.Context, prComments []*storage.PullRequestComment) error {
	err := c.store.WritePullRequestComments(context, prComments)
	if err == nil {
		for _, comment := range prComments {
			c.pullRequestCommentCache.Set(comment.PullRequestCommentID, comment)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadPullRequestReview(context context.Context, orgID string, repoID string, prID string,
	prReviewID string) (*storage.PullRequestReview, error) {
	if value, ok := c.pullRequestReviewCache.Get(prReviewID); ok {
		return value.(*storage.PullRequestReview), nil
	}

	result, err := c.store.ReadPullRequestReviewByID(context, orgID, repoID, prID, prReviewID)
	if err == nil {
		c.pullRequestReviewCache.Set(prReviewID, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WritePullRequestReviews(context context.Context, prReviews []*storage.PullRequestReview) error {
	err := c.store.WritePullRequestReviews(context, prReviews)
	if err == nil {
		for _, review := range prReviews {
			c.pullRequestReviewCache.Set(review.PullRequestReviewID, review)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadIssuePipeline(context context.Context, orgID string, repoID string, issueNumber int) (*storage.IssuePipeline, error) {
	key := orgID + repoID + strconv.Itoa(issueNumber)
	if value, ok := c.pipelineCache.Get(key); ok {
		return value.(*storage.IssuePipeline), nil
	}

	result, err := c.store.ReadIssuePipelineByNumber(context, orgID, repoID, issueNumber)
	if err == nil {
		c.pipelineCache.Set(key, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadMaintainer(context context.Context, orgID string, userID string) (*storage.Maintainer, error) {
	key := orgID + userID
	if value, ok := c.maintainerCache.Get(key); ok {
		return value.(*storage.Maintainer), nil
	}

	result, err := c.store.ReadMaintainerByID(context, orgID, userID)
	if err == nil {
		c.maintainerCache.Set(key, result)
	}

	return result, err
}
