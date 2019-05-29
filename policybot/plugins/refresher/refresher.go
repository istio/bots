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

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type Refresher struct {
	ghs   *gh.GitHubState
	repos map[string]bool
}

var scope = log.RegisterScope("refresher", "Dynamic database refresher", 0)

func NewRefresher(ghs *gh.GitHubState, orgs []config.Org) *Refresher {
	r := &Refresher{
		ghs:   ghs,
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
		scope.Debugf("Received IssuePayload: %+v", p)
		r.refresh(p.Repository.FullName, func(a *gh.Accumulator) interface{} {
			return a.IssueFromHook(&p)
		})

	case webhook.IssueCommentPayload:
		scope.Debugf("Received IssueCommentPayload: %+v", p)
		r.refresh(p.Repository.FullName, func(a *gh.Accumulator) interface{} {
			return a.IssueCommentFromHook(&p)
		})

	case webhook.PullRequestPayload:
		scope.Debugf("Received PullRequestPayload: %+v", p)
		r.refresh(p.Repository.FullName, func(a *gh.Accumulator) interface{} {
			pr, _ := a.PullRequestFromHook(&p)
			return pr
		})

	case webhook.PullRequestReviewPayload:
		scope.Debugf("Received PullRequestReviewPayload: %+v", p)
		r.refresh(p.Repository.FullName, func(a *gh.Accumulator) interface{} {
			return a.PullRequestReviewFromHook(&p)
		})

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *Refresher) refresh(repo string, conv func(a *gh.Accumulator) interface{}) {
	if _, ok := r.repos[repo]; !ok {
		// not a repo we're tracking
		return
	}

	a := r.ghs.NewAccumulator()
	conv(a)

	if err := a.Commit(); err != nil {
		scope.Errorf("Unable to update storage: %v", err)
		return
	}

	scope.Infof("Updated storage for repo %s", repo)
}
