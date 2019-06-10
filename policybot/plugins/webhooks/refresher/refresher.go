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
	"net/http"

	"istio.io/bots/policybot/pkg/fw"

	"istio.io/bots/policybot/pkg/storage"

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type Refresher struct {
	store storage.Store
	repos map[string]bool
}

var scope = log.RegisterScope("refresher", "Dynamic database refresher", 0)

func NewRefresher(store storage.Store, orgs []config.Org) fw.Webhook {
	r := &Refresher{
		store: store,
		repos: make(map[string]bool),
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

		issues := []*storage.Issue{gh.IssueFromHook(&p)}
		if err := r.store.WriteIssues(issues); err != nil {
			scope.Errorf(err.Error())
		}

	case webhook.IssueCommentPayload:
		scope.Infof("Received IssueCommentPayload: %s, %d, %s", p.Repository.FullName, p.Issue.Number, p.Action)

		issueComments := []*storage.IssueComment{gh.IssueCommentFromHook(&p)}
		if err := r.store.WriteIssueComments(issueComments); err != nil {
			scope.Errorf(err.Error())
		}

	case webhook.PullRequestPayload:
		scope.Infof("Received PullRequestPayload: %s, %d, %s", p.Repository.FullName, p.Number, p.Action)

		prs := []*storage.PullRequest{gh.PullRequestFromHook(&p)}
		if err := r.store.WritePullRequests(prs); err != nil {
			scope.Errorf(err.Error())
		}

	case webhook.PullRequestReviewPayload:
		scope.Infof("Received PullRequestReviewPayload: %s, %d, %s", p.Repository.FullName, p.PullRequest.Number, p.Action)

		reviews := []*storage.PullRequestReview{gh.PullRequestReviewFromHook(&p)}
		if err := r.store.WritePullRequestReviews(reviews); err != nil {
			scope.Errorf(err.Error())
		}

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}
