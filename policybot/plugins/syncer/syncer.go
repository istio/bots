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
	accum *gh.Accumulator
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
		accum: ghs.NewAccumulator(),
		store: store,
	}
}

func (s *Syncer) Handle(_ http.ResponseWriter, _ *http.Request) {
	for _, org := range s.orgs {
		s.handleOrg(org)
	}
}

func (s *Syncer) handleOrg(orgConfig config.Org) {
	scope.Infof("Syncing org %s", orgConfig.Name)

	org, _, err := s.ght.Get().Organizations.Get(s.ctx, orgConfig.Name)
	if err != nil {
		scope.Errorf("Unable to query information about organization %s from GitHub: %v", orgConfig.Name, err)
		return
	}

	o := s.accum.OrgFromAPI(org)

	for _, repoConfig := range orgConfig.Repos {
		if repo, _, err := s.ght.Get().Repositories.Get(s.ctx, orgConfig.Name, repoConfig.Name); err != nil {
			scope.Errorf("Unable to query information about repository %s/%s from GitHub: %v", orgConfig.Name, repoConfig.Name, err)
		} else {
			s.handleRepo(o, s.accum.RepoFromAPI(repo))
		}
	}

	if err := s.accum.Commit(); err != nil {
		scope.Errorf("Unable to commit data to storage: %v", err)
	}
}

type commentBundle struct {
	comments []*github.IssueComment
	issue    string
}

func (s *Syncer) handleRepo(org *storage.Org, repo *storage.Repo) {
	scope.Infof("Syncing repo %s/%s", org.Login, repo.Name)

	opt := &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	total := 0
	for {
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
					continue
				}
			}

			_ = s.accum.IssueFromAPI(org.OrgID, repo.RepoID, issue)

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
				s.accum.IssueCommentFromAPI(org.OrgID, repo.RepoID, bundle.issue, comment)
			}
		}

		if err := s.accum.Commit(); err != nil {
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
