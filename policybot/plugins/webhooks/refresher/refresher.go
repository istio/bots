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
	"net/http"

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/fw"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type Refresher struct {
	store storage.Store
	repos map[string]bool
	cache *cache.Cache
	ght   *gh.ThrottledClient
	ctx   context.Context
}

var scope = log.RegisterScope("refresher", "Dynamic database refresher", 0)

func NewRefresher(ctx context.Context, store storage.Store, cache *cache.Cache, ght *gh.ThrottledClient, orgs []config.Org) fw.Webhook {
	r := &Refresher{
		store: store,
		repos: make(map[string]bool),
		cache: cache,
		ght:   ght,
		ctx:   ctx,
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
	}
}

// accept an event arriving from GitHub
func (r *Refresher) Handle(_ http.ResponseWriter, githubObject interface{}) {
	switch p := githubObject.(type) {
	case webhook.IssuesPayload:
		scope.Infof("Received IssuePayload: %s, %d, %s", p.Repository.FullName, p.Issue.Number, p.Action)

		issue, discoveredUsers := gh.IssueFromHook(&p)
		issues := []*storage.Issue{issue}
		if err := r.store.WriteIssues(issues); err != nil {
			scope.Errorf(err.Error())
		}
		r.syncUsers(discoveredUsers)

	case webhook.IssueCommentPayload:
		scope.Infof("Received IssueCommentPayload: %s, %d, %s", p.Repository.FullName, p.Issue.Number, p.Action)

		issueComment, discoveredUsers := gh.IssueCommentFromHook(&p)
		issueComments := []*storage.IssueComment{issueComment}
		if err := r.store.WriteIssueComments(issueComments); err != nil {

			// try again, this time as a PR comment
			var prComment *storage.PullRequestComment
			prComment, discoveredUsers = gh.PullRequestCommentFromHook(&p)
			prComments := []*storage.PullRequestComment{prComment}
			if err := r.store.WritePullRequestComments(prComments); err != nil {
				scope.Errorf(err.Error())
			}
		}
		r.syncUsers(discoveredUsers)

	case webhook.PullRequestPayload:
		scope.Infof("Received PullRequestPayload: %s, %d, %s", p.Repository.FullName, p.Number, p.Action)

		pr, discoveredUsers := gh.PullRequestFromHook(&p)
		prs := []*storage.PullRequest{pr}
		if err := r.store.WritePullRequests(prs); err != nil {
			scope.Errorf(err.Error())
		}
		r.syncUsers(discoveredUsers)

	case webhook.PullRequestReviewPayload:
		scope.Infof("Received PullRequestReviewPayload: %s, %d, %s", p.Repository.FullName, p.PullRequest.Number, p.Action)

		review, discoveredUsers := gh.PullRequestReviewFromHook(&p)
		reviews := []*storage.PullRequestReview{review}
		if err := r.store.WritePullRequestReviews(reviews); err != nil {
			scope.Errorf(err.Error())
		}
		r.syncUsers(discoveredUsers)

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *Refresher) syncUsers(discoveredUsers map[string]string) {
	var users []*storage.User
	for _, du := range discoveredUsers {
		user, err := r.cache.ReadUserByLogin(du)
		if err != nil {
			scope.Warnf("unable to read user %s from storage: %v", du, err)
		}

		if user != nil {
			// we already know about this user
			continue
		}

		// didn't get user info from our storage layer, ask GiHub for details
		u, _, err := r.ght.Get().Users.Get(r.ctx, du)
		if err != nil {
			scope.Errorf("Unable to get info on user %s from GitHub: %v", du, err)
		} else {
			users = append(users, gh.UserFromAPI(u))
		}
	}

	if err := r.store.WriteUsers(users); err != nil {
		scope.Errorf("Unable to write users: %v", err)
	}
}
