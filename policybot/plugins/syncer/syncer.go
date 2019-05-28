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

package syncer

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/go-github/v25/github"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

// Syncer is responsible for synchronizing issues and pull request from GitHub to our local store
type Syncer struct {
	ctx   context.Context
	ghs   *gh.GitHubState
	ght   *util.GitHubThrottle
	orgs  []config.Org
	store storage.Store
}

var (
	scope = log.RegisterScope("syncer", "The GitHub issue & PR syncer", 0)

	issuesSynced = stats.Int64(
		"policybot/syncer/issues_synced_total", "The number of issues having been synchronized.", stats.UnitDimensionless)
)

func init() {
	_ = view.Register(&view.View{
		Name:        issuesSynced.Name(),
		Description: issuesSynced.Description(),
		Measure:     issuesSynced,
		TagKeys:     []tag.Key{},
		Aggregation: view.LastValue(),
	})
}

func NewSyncer(ctx context.Context, ght *util.GitHubThrottle, ghs *gh.GitHubState, store storage.Store, orgs []config.Org) *Syncer {
	return &Syncer{
		ctx:   ctx,
		ght:   ght,
		ghs:   ghs,
		orgs:  orgs,
		store: store,
	}
}

func (s *Syncer) Handle(_ http.ResponseWriter, _ *http.Request) {
	a := s.ghs.NewAccumulator()
	for _, org := range s.orgs {
		s.handleOrg(a, org)
	}
}

func (s *Syncer) handleOrg(a *gh.Accumulator, orgConfig config.Org) {
	scope.Infof("Syncing org %s", orgConfig.Name)

	org, _, err := s.ght.Get().Organizations.Get(s.ctx, orgConfig.Name)
	if err != nil {
		scope.Errorf("Unable to query information about organization %s from GitHub: %v", orgConfig.Name, err)
		return
	}

	o := a.OrgFromAPI(org)

	for _, repoConfig := range orgConfig.Repos {
		if repo, _, err := s.ght.Get().Repositories.Get(s.ctx, orgConfig.Name, repoConfig.Name); err != nil {
			scope.Errorf("Unable to query information about repository %s/%s from GitHub: %v", orgConfig.Name, repoConfig.Name, err)
		} else {
			s.handleRepo(a, o, a.RepoFromAPI(repo))
		}
	}

	if err := a.Commit(); err != nil {
		scope.Errorf("Unable to commit data to storage: %v", err)
	}
}

func (s *Syncer) handleRepo(a *gh.Accumulator, org *storage.Org, repo *storage.Repo) {
	scope.Infof("Syncing repo %s/%s", org.Login, repo.Name)

	s.handleIssues(a, org, repo)
	s.handlePullRequests(a, org, repo)
}

type commentBundle struct {
	comments []*github.IssueComment
	issue    string
}

func (s *Syncer) handleIssues(a *gh.Accumulator, org *storage.Org, repo *storage.Repo) {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	total := 0
	for {
		scope.Debugf("Getting issues from repo %s/%s", org.Login, repo.Name)

		issues, resp, err := s.ght.Get().Issues.ListByRepo(s.ctx, org.Login, repo.Name, opt)
		if err != nil {
			scope.Errorf("Unable to list all issues in repo %s/%s: %v\n", org.Login, repo.Name, err)
			return
		}

		wg := sync.WaitGroup{}
		wg.Add(len(issues))

		lock := sync.Mutex{}
		var bundles []commentBundle

		for _, issue := range issues {

			// if this issue is already known to us and is up to date, skip further processing
			if existing, _ := s.ghs.ReadIssue(org.OrgID, repo.RepoID, issue.GetNodeID()); existing != nil {
				if existing.UpdatedAt == issue.GetUpdatedAt() {
					wg.Done()
					continue
				}
			}

			_ = a.IssueFromAPI(org.OrgID, repo.RepoID, issue)

			capture := issue
			go func() {
				comm := s.fetchComments(org, repo, capture.GetNumber())

				lock.Lock()
				bundles = append(bundles, commentBundle{comm, capture.GetNodeID()})
				lock.Unlock()

				wg.Done()
			}()
		}
		wg.Wait()

		for _, bundle := range bundles {
			for _, comment := range bundle.comments {
				_ = a.IssueCommentFromAPI(org.OrgID, repo.RepoID, bundle.issue, comment)
			}
		}

		if err := a.Commit(); err != nil {
			scope.Errorf("Unable to commit data to storage: %v", err)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage

		total += len(issues)
		scope.Infof("Synced %d issues in repo %s/%s", total, org.Login, repo.Name)
	}
}

func (s *Syncer) fetchComments(org *storage.Org, repo *storage.Repo, issueNumber int) []*github.IssueComment {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var result []*github.IssueComment
	for {
		scope.Debugf("Getting issue comments for issue %d from repo %s/%s", issueNumber, org.Login, repo.Name)

		comments, resp, err := s.ght.Get().Issues.ListComments(s.ctx, org.Login, repo.Name, issueNumber, opt)
		if err != nil {
			scope.Errorf("Unable to list all comments for issue %d in repo %s/%s: %v", issueNumber, org.Login, repo.Name, err)
			return result
		}

		result = append(result, comments...)

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	return result
}

type reviewBundle struct {
	reviews []*github.PullRequestReview
	pr      string
}

func (s *Syncer) handlePullRequests(a *gh.Accumulator, org *storage.Org, repo *storage.Repo) {
	opt := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	total := 0
	for {
		scope.Debugf("Getting pull requests from repo %s/%s", org.Login, repo.Name)

		prs, resp, err := s.ght.Get().PullRequests.List(s.ctx, org.Login, repo.Name, opt)
		if err != nil {
			scope.Errorf("Unable to list all pull requests in repo %s/%s: %v\n", org.Login, repo.Name, err)
			return
		}

		wg := sync.WaitGroup{}
		wg.Add(len(prs))

		lock := sync.Mutex{}
		var bundles []reviewBundle

		for _, pr := range prs {

			// if this pr is already known to us and is up to date, skip further processing
			if existing, _ := s.ghs.ReadPullRequest(org.OrgID, repo.RepoID, pr.GetNodeID()); existing != nil {
				if existing.UpdatedAt == pr.GetUpdatedAt() {
					wg.Done()
					continue
				}
			}

			_ = a.PullRequestFromAPI(org.OrgID, repo.RepoID, pr)

			capture := pr
			go func() {
				reviews := s.fetchReviews(org, repo, capture.GetNumber())

				lock.Lock()
				bundles = append(bundles, reviewBundle{reviews, capture.GetNodeID()})
				lock.Unlock()

				wg.Done()
			}()
		}
		wg.Wait()

		for _, bundle := range bundles {
			for _, review := range bundle.reviews {
				_ = a.PullRequestReviewFromAPI(org.OrgID, repo.RepoID, bundle.pr, review)
			}
		}

		if err := a.Commit(); err != nil {
			scope.Errorf("Unable to commit data to storage: %v", err)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage

		total += len(prs)
		scope.Infof("Synced %d pull requests in repo %s/%s", total, org.Login, repo.Name)
	}
}

func (s *Syncer) fetchReviews(org *storage.Org, repo *storage.Repo, prNumber int) []*github.PullRequestReview {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	var result []*github.PullRequestReview
	for {
		scope.Debugf("Getting reviews for pr %d from repo %s/%s", prNumber, org.Login, repo.Name)

		reviews, resp, err := s.ght.Get().PullRequests.ListReviews(s.ctx, org.Login, repo.Name, prNumber, opt)
		if err != nil {
			scope.Errorf("Unable to list all comments for pr %d in repo %s/%s: %v", prNumber, org.Login, repo.Name, err)
			return result
		}

		result = append(result, reviews...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return result
}
