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
	"time"

	"github.com/google/go-github/v25/github"

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
	store storage.Store
	orgs  []config.Org
}

var scope = log.RegisterScope("syncer", "The GitHub issue & PR syncer", 0)

func NewSyncer(ctx context.Context, ght *util.GitHubThrottle, ghs *gh.GitHubState, store storage.Store, orgs []config.Org) *Syncer {
	return &Syncer{
		ctx:   ctx,
		ght:   ght,
		ghs:   ghs,
		store: store,
		orgs:  orgs,
	}
}

func (s *Syncer) Handle(_ http.ResponseWriter, _ *http.Request) {
	s.Sync()
}

func (s *Syncer) Sync() {
	a := s.ghs.NewAccumulator()

	start := time.Now().UTC()
	priorStart := time.Time{}
	if activity, err := s.store.ReadBotActivity(); err == nil {
		priorStart = activity.LastSyncStart
	}

	for _, org := range s.orgs {
		s.handleOrg(a, org, priorStart)
	}

	end := time.Now()
	_ = s.store.WriteBotActivity(&storage.BotActivity{LastSyncStart: start, LastSyncEnd: end})
}

func (s *Syncer) handleOrg(a *gh.Accumulator, orgConfig config.Org, startTime time.Time) {
	scope.Infof("Syncing org %s", orgConfig.Name)

	org, _, err := s.ght.Get().Organizations.Get(s.ctx, orgConfig.Name)
	if err != nil {
		scope.Errorf("Unable to query information about organization %s from GitHub: %v", orgConfig.Name, err)
		return
	}

	o := a.OrgFromAPI(org)

	s.handleMembers(a, o)

	for _, repoConfig := range orgConfig.Repos {
		if repo, _, err := s.ght.Get().Repositories.Get(s.ctx, orgConfig.Name, repoConfig.Name); err != nil {
			scope.Errorf("Unable to query information about repository %s/%s from GitHub: %v", orgConfig.Name, repoConfig.Name, err)
		} else {
			s.handleRepo(a, o, a.RepoFromAPI(repo), startTime)
		}
	}

	if err := a.Commit(); err != nil {
		scope.Errorf("Unable to commit data to storage: %v", err)
	}
}

func (s *Syncer) handleMembers(a *gh.Accumulator, org *storage.Org) {
	members := s.fetchMembers(org)
	for _, member := range members {
		_ = a.MemberFromAPI(org, member)
	}
}

func (s *Syncer) fetchMembers(org *storage.Org) []*github.User {
	opt := &github.ListMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var result []*github.User
	for {
		scope.Debugf("Getting members of org %s", org.Login)

		members, resp, err := s.ght.Get().Organizations.ListMembers(s.ctx, org.Login, opt)
		if err != nil {
			scope.Errorf("Unable to list all members of org %s: %v", org.Login, err)
			return result
		}

		result = append(result, members...)

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	return result
}

func (s *Syncer) handleRepo(a *gh.Accumulator, org *storage.Org, repo *storage.Repo, startTime time.Time) {
	scope.Infof("Syncing repo %s/%s", org.Login, repo.Name)

	s.handleIssues(a, org, repo, startTime)
	s.handlePullRequests(a, org, repo)
}

func (s *Syncer) handleIssues(a *gh.Accumulator, org *storage.Org, repo *storage.Repo, startTime time.Time) {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		Since: startTime,
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
		commentsMap := sync.Map{}

		for _, issue := range issues {

			// if this issue is already known to us and is up to date, skip further processing
			if existing, _ := s.ghs.ReadIssue(org.OrgID, repo.RepoID, issue.GetNodeID()); existing != nil {
				if existing.UpdatedAt == issue.GetUpdatedAt() {
					continue
				}
			}

			wg.Add(1)
			capture := issue
			go func() {
				comm := s.fetchComments(org, repo, capture.GetNumber(), startTime)
				commentsMap.Store(capture.GetNodeID(), comm)
				wg.Done()
			}()
		}
		wg.Wait()

		for _, issue := range issues {
			_ = a.IssueFromAPI(org.OrgID, repo.RepoID, issue)

			if comments, ok := commentsMap.Load(issue.GetNodeID()); ok {
				for _, comment := range comments.([]*github.IssueComment) {
					_ = a.IssueCommentFromAPI(org.OrgID, repo.RepoID, issue.GetNodeID(), comment)
				}
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

func (s *Syncer) fetchComments(org *storage.Org, repo *storage.Repo, issueNumber int, startTime time.Time) []*github.IssueComment {
	opt := &github.IssueListCommentsOptions{
		Since: startTime,
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
		reviewsMap := sync.Map{}
		filesMap := sync.Map{}

		for _, pr := range prs {
			// if this pr is already known to us and is up to date, skip further processing
			if existing, _ := s.ghs.ReadPullRequest(org.OrgID, repo.RepoID, pr.GetNodeID()); existing != nil {
				if existing.UpdatedAt == pr.GetUpdatedAt() {
					continue
				}
			}

			wg.Add(2)

			capture := pr
			go func() {
				reviews := s.fetchReviews(org, repo, capture.GetNumber())
				reviewsMap.Store(capture.GetNodeID(), reviews)
				wg.Done()
			}()

			go func() {
				files := s.fetchFiles(org, repo, capture.GetNumber())
				filesMap.Store(capture.GetNodeID(), files)
				wg.Done()
			}()
		}
		wg.Wait()

		for _, pr := range prs {
			if reviews, ok := reviewsMap.Load(pr.GetNodeID()); ok {
				for _, review := range reviews.([]*github.PullRequestReview) {
					_ = a.PullRequestReviewFromAPI(org.OrgID, repo.RepoID, pr.GetNodeID(), review)
				}
			}

			var files []string
			if filesRaw, ok := filesMap.Load(pr.GetNodeID()); ok {
				files = filesRaw.([]string)
				for i := range files {
					files[i] = org.Login + "/" + repo.Name + "/" + files[i]
				}
			}

			_ = a.PullRequestFromAPI(org.OrgID, repo.RepoID, pr, files)
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

func (s *Syncer) fetchFiles(org *storage.Org, repo *storage.Repo, prNumber int) []string {
	scope.Debugf("Getting file list for pr %d from repo %s/%s", prNumber, org.Login, repo.Name)

	opt := &github.ListOptions{
		PerPage: 100,
	}

	var allFiles []string
	for {
		files, resp, err := s.ght.Get().PullRequests.ListFiles(s.ctx, org.Login, repo.Name, prNumber, opt)
		if err != nil {
			scope.Errorf("Unable to list all files for pull request %d in repo %s/%s: %v\n", prNumber, org.Login, repo.Name, err)
			return allFiles
		}

		for _, f := range files {
			allFiles = append(allFiles, f.GetFilename())
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allFiles
}

func (s *Syncer) fetchReviews(org *storage.Org, repo *storage.Repo, prNumber int) []*github.PullRequestReview {
	scope.Debugf("Getting reviews for pr %d from repo %s/%s", prNumber, org.Login, repo.Name)

	opt := &github.ListOptions{
		PerPage: 100,
	}

	var result []*github.PullRequestReview
	for {
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
