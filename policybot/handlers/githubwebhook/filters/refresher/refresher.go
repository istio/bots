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

package refresher

import (
	"context"

	webhook "github.com/go-playground/webhooks/github"
	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type Refresher struct {
	repos map[string]bool
	cache *cache.Cache
	ght   *gh.ThrottledClient
}

var scope = log.RegisterScope("refresher", "Dynamic database refresher", 0)

func NewRefresher(cache *cache.Cache, ght *gh.ThrottledClient, orgs []config.Org) filters.Filter {
	r := &Refresher{
		repos: make(map[string]bool),
		cache: cache,
		ght:   ght,
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			r.repos[org.Name+"/"+repo.Name] = true
		}
	}

	return r
}

func (r *Refresher) Events() []webhook.Event {
	return []webhook.Event{
		webhook.IssuesEvent,
		webhook.IssueCommentEvent,
		webhook.PullRequestEvent,
		webhook.PullRequestReviewEvent,
		webhook.CommitCommentEvent,
	}
}

// accept an event arriving from GitHub
func (r *Refresher) Handle(context context.Context, githubObject interface{}) {
	switch p := githubObject.(type) {
	case webhook.IssuesPayload:
		scope.Infof("Received IssuePayload: %s, %d, %s", p.Repository.FullName, p.Issue.Number, p.Action)

		if !r.repos[p.Repository.FullName] {
			scope.Infof("Ignoring issue %d from repo %s since it's not in a monitored repo", p.Issue.Number, p.Repository.FullName)
			return
		}

		issue, discoveredUsers := gh.IssueFromHook(&p)
		issues := []*storage.Issue{issue}
		if err := r.cache.WriteIssues(context, issues); err != nil {
			scope.Errorf(err.Error())
		}
		r.syncUsers(context, discoveredUsers)

	case webhook.IssueCommentPayload:
		scope.Infof("Received IssueCommentPayload: %s, %d, %s", p.Repository.FullName, p.Issue.Number, p.Action)

		if !r.repos[p.Repository.FullName] {
			scope.Infof("Ignoring issue comment for issue %d from repo %s since it's not in a monitored repo", p.Issue.Number, p.Repository.FullName)
			return
		}

		issueComment, discoveredUsers := gh.IssueCommentFromHook(&p)
		issueComments := []*storage.IssueComment{issueComment}
		if err := r.cache.WriteIssueComments(context, issueComments); err != nil {

			// try again, this time as a PR comment
			var prComment *storage.PullRequestComment
			prComment, discoveredUsers = gh.PullRequestCommentFromHook(&p)
			prComments := []*storage.PullRequestComment{prComment}
			if err := r.cache.WritePullRequestComments(context, prComments); err != nil {
				scope.Errorf(err.Error())
			}
		}
		r.syncUsers(context, discoveredUsers)

	case webhook.PullRequestPayload:
		scope.Infof("Received PullRequestPayload: %s, %d, %s", p.Repository.FullName, p.Number, p.Action)

		if !r.repos[p.Repository.FullName] {
			scope.Infof("Ignoring PR %d from repo %s since it's not in a monitored repo", p.PullRequest.Number, p.Repository.FullName)
			return
		}

		pr, discoveredUsers := gh.PullRequestFromHook(&p)

		opt := &github.ListOptions{
			PerPage: 100,
		}

		// get the set of files comprising this PR since the payload didn't supply them
		var allFiles []string
		for {
			files, resp, err := r.ght.Get(context).PullRequests.ListFiles(context, p.Repository.Owner.Login, p.Repository.Name, int(p.Number), opt)
			if err != nil {
				scope.Errorf("Unable to list all files for pull request %d in repo %s: %v\n", p.Number, p.Repository.FullName, err)
				return
			}

			for _, f := range files {
				allFiles = append(allFiles, f.GetFilename())
			}

			if resp.NextPage == 0 {
				break
			}

			opt.Page = resp.NextPage
		}
		pr.Files = allFiles

		prs := []*storage.PullRequest{pr}
		if err := r.cache.WritePullRequests(context, prs); err != nil {
			scope.Errorf(err.Error())
		}
		r.syncUsers(context, discoveredUsers)

	case webhook.PullRequestReviewPayload:
		scope.Infof("Received PullRequestReviewPayload: %s, %d, %s", p.Repository.FullName, p.PullRequest.Number, p.Action)

		if !r.repos[p.Repository.FullName] {
			scope.Infof("Ignoring PR review for PR %d from repo %s since it's not in a monitored repo", p.PullRequest.Number, p.Repository.FullName)
			return
		}

		review, discoveredUsers := gh.PullRequestReviewFromHook(&p)
		reviews := []*storage.PullRequestReview{review}
		if err := r.cache.WritePullRequestReviews(context, reviews); err != nil {
			scope.Errorf(err.Error())
		}
		r.syncUsers(context, discoveredUsers)

	case webhook.CommitCommentPayload:
		scope.Infof("Received CommitCommentPayload: %s, %s", p.Repository.FullName, p.Action)

		if !r.repos[p.Repository.FullName] {
			scope.Infof("Ignoring repo comment from repo %s since it's not in a monitored repo", p.Repository.FullName)
			return
		}

		comment, discoveredUsers := gh.RepoCommentFromHook(&p)
		comments := []*storage.RepoComment{comment}
		if err := r.cache.WriteRepoComments(context, comments); err != nil {
			scope.Errorf(err.Error())
		}
		r.syncUsers(context, discoveredUsers)

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *Refresher) syncUsers(context context.Context, discoveredUsers map[string]string) {
	var users []*storage.User
	for _, du := range discoveredUsers {
		user, err := r.cache.ReadUserByLogin(context, du)
		if err != nil {
			scope.Warnf("unable to read user %s from storage: %v", du, err)
		}

		if user != nil {
			// we already know about this user
			continue
		}

		// didn't get user info from our storage layer, ask GitHub for details
		u, _, err := r.ght.Get(context).Users.Get(context, du)
		if err != nil {
			scope.Errorf("Unable to get info on user %s from GitHub: %v", du, err)
		} else {
			users = append(users, gh.UserFromAPI(u))
		}
	}

	if err := r.cache.WriteUsers(context, users); err != nil {
		scope.Errorf("Unable to write users: %v", err)
	}
}
