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
	store                         storage.Store
	orgCache                      cache.ExpiringCache
	repoCache                     cache.ExpiringCache
	issueCache                    cache.ExpiringCache
	issueCommentCache             cache.ExpiringCache
	labelCache                    cache.ExpiringCache
	userCache                     cache.ExpiringCache
	userByLoginCache              cache.ExpiringCache
	pullRequestCache              cache.ExpiringCache
	pullRequestReviewCommentCache cache.ExpiringCache
	pullRequestReviewCache        cache.ExpiringCache
	pipelineCache                 cache.ExpiringCache
	maintainerCache               cache.ExpiringCache
	memberCache                   cache.ExpiringCache
	repoCommentCache              cache.ExpiringCache
	testResultCache               cache.ExpiringCache
}

func New(store storage.Store, entryTTL time.Duration) *Cache {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if entryTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = entryTTL / 2
	}

	return &Cache{
		store:                         store,
		orgCache:                      cache.NewTTL(entryTTL, evictionInterval),
		repoCache:                     cache.NewTTL(entryTTL, evictionInterval),
		issueCache:                    cache.NewTTL(entryTTL, evictionInterval),
		issueCommentCache:             cache.NewTTL(entryTTL, evictionInterval),
		labelCache:                    cache.NewTTL(entryTTL, evictionInterval),
		userCache:                     cache.NewTTL(entryTTL, evictionInterval),
		userByLoginCache:              cache.NewTTL(entryTTL, evictionInterval),
		pullRequestCache:              cache.NewTTL(entryTTL, evictionInterval),
		pullRequestReviewCommentCache: cache.NewTTL(entryTTL, evictionInterval),
		pullRequestReviewCache:        cache.NewTTL(entryTTL, evictionInterval),
		pipelineCache:                 cache.NewTTL(entryTTL, evictionInterval),
		maintainerCache:               cache.NewTTL(entryTTL, evictionInterval),
		memberCache:                   cache.NewTTL(entryTTL, evictionInterval),
		repoCommentCache:              cache.NewTTL(entryTTL, evictionInterval),
		testResultCache:               cache.NewTTL(entryTTL, evictionInterval),
	}
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadOrg(context context.Context, orgLogin string) (*storage.Org, error) {
	key := orgLogin
	if value, ok := c.orgCache.Get(key); ok {
		return value.(*storage.Org), nil
	}

	result, err := c.store.ReadOrg(context, orgLogin)
	if err == nil {
		c.orgCache.Set(key, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadRepo(context context.Context, orgLogin string, repoName string) (*storage.Repo, error) {
	key := orgLogin + repoName
	if value, ok := c.repoCache.Get(key); ok {
		return value.(*storage.Repo), nil
	}

	result, err := c.store.ReadRepo(context, orgLogin, repoName)
	if err == nil {
		c.repoCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteRepoComments(context context.Context, comments []*storage.RepoComment) error {
	err := c.store.WriteRepoComments(context, comments)
	if err == nil {
		for _, comment := range comments {
			c.repoCommentCache.Set(comment.OrgLogin+comment.RepoName+strconv.Itoa(int(comment.CommentID)), comment)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadUser(context context.Context, userLogin string) (*storage.User, error) {
	key := userLogin
	if value, ok := c.userCache.Get(key); ok {
		return value.(*storage.User), nil
	}

	result, err := c.store.ReadUser(context, userLogin)
	if err == nil {
		c.userCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteUsers(context context.Context, users []*storage.User) error {
	err := c.store.WriteUsers(context, users)
	if err == nil {
		for _, user := range users {
			c.userCache.Set(user.UserLogin, user)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadLabel(context context.Context, orgLogin string, repoName string, labelName string) (*storage.Label, error) {
	key := orgLogin + repoName + labelName
	if value, ok := c.labelCache.Get(key); ok {
		return value.(*storage.Label), nil
	}

	result, err := c.store.ReadLabel(context, orgLogin, repoName, labelName)
	if err == nil {
		c.labelCache.Set(key, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadIssue(context context.Context, orgLogin string, repoName string, issueNumber int) (*storage.Issue, error) {
	key := orgLogin + repoName + strconv.Itoa(issueNumber)
	if value, ok := c.issueCache.Get(key); ok {
		return value.(*storage.Issue), nil
	}

	result, err := c.store.ReadIssue(context, orgLogin, repoName, issueNumber)
	if err == nil {
		c.issueCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteIssues(context context.Context, issues []*storage.Issue) error {
	err := c.store.WriteIssues(context, issues)
	if err == nil {
		for _, issue := range issues {
			c.issueCache.Set(issue.OrgLogin+issue.RepoName+strconv.Itoa(int(issue.IssueNumber)), issue)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadIssueComment(context context.Context, orgLogin string, repoName string, issueNumber int,
	issueCommentID int) (*storage.IssueComment, error) {
	key := orgLogin + repoName + strconv.Itoa(issueNumber) + strconv.Itoa(issueCommentID)
	if value, ok := c.issueCommentCache.Get(key); ok {
		return value.(*storage.IssueComment), nil
	}

	result, err := c.store.ReadIssueComment(context, orgLogin, repoName, issueNumber, issueCommentID)
	if err == nil {
		c.issueCommentCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteIssueComments(context context.Context, issueComments []*storage.IssueComment) error {
	err := c.store.WriteIssueComments(context, issueComments)
	if err == nil {
		for _, comment := range issueComments {
			c.issueCommentCache.Set(comment.OrgLogin+comment.RepoName+strconv.Itoa(int(comment.IssueNumber))+strconv.Itoa(int(comment.IssueCommentID)),
				comment)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadPullRequest(context context.Context, orgLogin string, repoName string, prNumber int) (*storage.PullRequest, error) {
	key := orgLogin + repoName + strconv.Itoa(prNumber)
	if value, ok := c.pullRequestCache.Get(key); ok {
		return value.(*storage.PullRequest), nil
	}

	result, err := c.store.ReadPullRequest(context, orgLogin, repoName, prNumber)
	if err == nil {
		c.pullRequestReviewCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WritePullRequests(context context.Context, prs []*storage.PullRequest) error {
	err := c.store.WritePullRequests(context, prs)
	if err == nil {
		for _, pr := range prs {
			c.pullRequestCache.Set(pr.OrgLogin+pr.RepoName+strconv.Itoa(int(pr.PullRequestNumber)), pr)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadPullRequestReviewComment(context context.Context, orgLogin string, repoName string, prNumber int,
	prCommentID int) (*storage.PullRequestReviewComment, error) {
	key := orgLogin + repoName + strconv.Itoa(prNumber) + strconv.Itoa(prCommentID)
	if value, ok := c.pullRequestReviewCommentCache.Get(key); ok {
		return value.(*storage.PullRequestReviewComment), nil
	}

	result, err := c.store.ReadPullRequestReviewComment(context, orgLogin, repoName, prNumber, prCommentID)
	if err == nil {
		c.pullRequestReviewCommentCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WritePullRequestReviewComments(context context.Context, prComments []*storage.PullRequestReviewComment) error {
	err := c.store.WritePullRequestReviewComments(context, prComments)
	if err == nil {
		for _, comment := range prComments {
			c.pullRequestReviewCommentCache.Set(comment.OrgLogin+
				comment.RepoName+
				strconv.Itoa(int(comment.PullRequestNumber))+
				strconv.Itoa(int(comment.PullRequestReviewCommentID)), comment)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadPullRequestReview(context context.Context, orgLogin string, repoName string, prNumber int,
	prReviewID int) (*storage.PullRequestReview, error) {
	key := orgLogin + repoName + strconv.Itoa(prNumber) + strconv.Itoa(prReviewID)
	if value, ok := c.pullRequestReviewCache.Get(key); ok {
		return value.(*storage.PullRequestReview), nil
	}

	result, err := c.store.ReadPullRequestReview(context, orgLogin, repoName, prNumber, prReviewID)
	if err == nil {
		c.pullRequestReviewCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WritePullRequestReviews(context context.Context, prReviews []*storage.PullRequestReview) error {
	err := c.store.WritePullRequestReviews(context, prReviews)
	if err == nil {
		for _, review := range prReviews {
			c.pullRequestReviewCache.Set(review.OrgLogin+
				review.RepoName+
				strconv.Itoa(int(review.PullRequestNumber))+
				strconv.Itoa(int(review.PullRequestReviewID)), review)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadIssuePipeline(context context.Context, orgLogin string, repoName string, issueNumber int) (*storage.IssuePipeline, error) {
	key := orgLogin + repoName + strconv.Itoa(issueNumber)
	if value, ok := c.pipelineCache.Get(key); ok {
		return value.(*storage.IssuePipeline), nil
	}

	result, err := c.store.ReadIssuePipeline(context, orgLogin, repoName, issueNumber)
	if err == nil {
		c.pipelineCache.Set(key, result)
	}

	return result, err
}

func (c *Cache) ReadTestResult(context context.Context,
	orgLogin string, repoName string, testName string, prNum int64, runNumber int64) (*storage.TestResult, error) {
	key := orgLogin + repoName + testName + strconv.FormatInt(prNum, 10) + strconv.FormatInt(runNumber, 10)
	if value, ok := c.testResultCache.Get(key); ok {
		return value.(*storage.TestResult), nil
	}

	result, err := c.store.ReadTestResult(context, orgLogin, repoName, testName, prNum, runNumber)
	if err == nil {
		c.testResultCache.Set(key, result)
	}

	return result, err
}

// Writes to DB and if successful, updates the cache
func (c *Cache) WriteTestResults(context context.Context, testResults []*storage.TestResult) error {
	err := c.store.WriteTestResults(context, testResults)
	if err == nil {
		for _, testResult := range testResults {
			orgID := testResult.OrgLogin
			repoID := testResult.RepoName
			testName := testResult.TestName
			prNum := testResult.PullRequestNumber
			runNum := testResult.RunNumber
			key := orgID + repoID + testName + strconv.FormatInt(prNum, 10) + strconv.FormatInt(runNum, 10)

			c.testResultCache.Set(key, testResult)
		}
	}

	return err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadMaintainer(context context.Context, orgLogin string, userLogin string) (*storage.Maintainer, error) {
	key := orgLogin + userLogin
	if value, ok := c.maintainerCache.Get(key); ok {
		return value.(*storage.Maintainer), nil
	}

	result, err := c.store.ReadMaintainer(context, orgLogin, userLogin)
	if err == nil {
		c.maintainerCache.Set(key, result)
	}

	return result, err
}

// Reads from cache and if not found reads from DB
func (c *Cache) ReadMember(context context.Context, orgLogin string, userLogin string) (*storage.Member, error) {
	key := orgLogin + userLogin
	if value, ok := c.memberCache.Get(key); ok {
		return value.(*storage.Member), nil
	}

	result, err := c.store.ReadMember(context, orgLogin, userLogin)
	if err == nil {
		c.memberCache.Set(key, result)
	}

	return result, err
}
