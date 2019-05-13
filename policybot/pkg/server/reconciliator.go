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

package server

import (
	"context"
	"net/http"

	"github.com/google/go-github/v25/github"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"istio.io/pkg/log"

	"istio.io/bots/policybot/pkg/storage"
)

type reconciliator struct {
	ctx        context.Context
	store      storage.Store
	client     *github.Client
	orgs       []Org
	knownUsers map[string]*github.User
	newUsers   map[string]*github.User
	limiter    *rate.Limiter
}

var scope = log.RegisterScope("reconciliator", "The GitHub reconciliator", 0)

const (
	maxGitHubRequestsPerHour   = 5000.0
	maxGitHubRequestsPerSecond = maxGitHubRequestsPerHour / 3600.0
)

func newReconciliator(ctx context.Context, githubAccessToken string, orgs []Org, store storage.Store) *reconciliator {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)
	httpClient := oauth2.NewClient(ctx, src)
	client := github.NewClient(httpClient)

	r := &reconciliator{
		ctx:        ctx,
		store:      store,
		client:     client,
		orgs:       orgs,
		knownUsers: make(map[string]*github.User),
		newUsers:   make(map[string]*github.User),
		limiter:    rate.NewLimiter(maxGitHubRequestsPerSecond, 100),
	}

	return r
}

func (r *reconciliator) handle(_ http.ResponseWriter, _ *http.Request) {
	for _, org := range r.orgs {
		r.handleOrg(org)
	}
}

func (r *reconciliator) handleOrg(orgConfig Org) {
	scope.Infof("Reconciling org %s", orgConfig.Name)

	_ = r.limiter.Wait(r.ctx)
	org, _, err := r.client.Organizations.Get(r.ctx, orgConfig.Name)
	if err != nil {
		scope.Errorf("Unable to query information about organization %s from GitHub: %v", orgConfig.Name, err)
		return
	}

	repos := make([]*github.Repository, len(orgConfig.Repos))
	for i, repoConfig := range orgConfig.Repos {
		var err error

		_ = r.limiter.Wait(r.ctx)
		if repos[i], _, err = r.client.Repositories.Get(r.ctx, orgConfig.Name, repoConfig.Name); err != nil {
			scope.Errorf("Unable to query information about repository %s/%s from GitHub: %v", orgConfig.Name, repoConfig.Name, err)
		}
	}

	if err := r.store.WriteOrgAndRepos(org, repos); err != nil {
		scope.Errorf("Unable to update storage with info from org %s: %v", org.GetName(), err)
		return
	}

	for _, repo := range repos {
		if repo != nil {
			r.handleRepo(org, repo)
		}
	}
}

func (r *reconciliator) handleRepo(org *github.Organization, repo *github.Repository) {
	scope.Infof("Reconciling repo %s", repo.GetFullName())

	opt := &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	total := 0
	for {
		_ = r.limiter.Wait(r.ctx)
		issues, resp, err := r.client.Issues.ListByRepo(r.ctx, org.GetName(), repo.GetName(), opt)
		if err != nil {
			scope.Errorf("Unable to list all issues in repo %s/%s: %v\n", org.GetName(), repo.GetName(), err)
			return
		}

		for _, issue := range issues {
			r.handleIssue(org, repo, issue)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage

		total += len(issues)
		scope.Infof("Reconciled %d issues in repo %s", total, repo.GetFullName())
	}

	if len(r.newUsers) > 0 {
		users := make([]*github.User, len(r.newUsers), 0)
		for _, user := range r.newUsers {
			users = append(users, user)
		}

		if err := r.store.WriteUsers(users); err != nil {
			scope.Errorf("Unable to update user data: %v", err)
		} else {
			// move the new users to the known users list
			for k, v := range r.newUsers {
				delete(r.newUsers, k)
				r.knownUsers[k] = v
			}
		}
	}
}

func (r *reconciliator) recordUser(user *github.User) {
	if _, ok := r.knownUsers[user.GetNodeID()]; !ok {
		r.newUsers[user.GetNodeID()] = user
	}
}

func (r *reconciliator) handleIssue(org *github.Organization, repo *github.Repository, issue *github.Issue) {
	if existing, _ := r.store.ReadIssue(org, repo, issue.GetNodeID()); existing != nil {
		if issue.GetUpdatedAt() == existing.GetUpdatedAt() {
			// nothing's changed, no need to read the comments nor to update storage
			return
		}
	}

	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	r.recordUser(issue.User)
	for _, user := range issue.Assignees {
		r.recordUser(user)
	}

	allComments := []*github.IssueComment{}
	for {
		_ = r.limiter.Wait(r.ctx)
		comments, resp, err := r.client.Issues.ListComments(r.ctx, org.GetName(), repo.GetName(), issue.GetNumber(), opt)
		if err != nil {
			scope.Errorf("Unable to list all comments for issue %d in repo %s/%s: %v\n", issue.GetNumber(), org.GetName(), repo.GetName(), err)
			return
		}

		allComments = append(allComments, comments...)

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	for _, comment := range allComments {
		r.recordUser(comment.User)
	}

	if err := r.store.WriteIssueAndComments(org, repo, issue, allComments); err != nil {
		scope.Errorf("Unable to update storage with info from issue %d in repo %s: %v", issue.GetNumber(), repo.GetFullName(), err)
	}
}
