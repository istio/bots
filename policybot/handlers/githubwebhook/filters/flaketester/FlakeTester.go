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

package flaketester

import (
	"context"
	"net/http"

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/testflakes"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type FlakeTester struct {
	store storage.Store
	repos map[string]bool
	cache *cache.Cache
	ght   *gh.ThrottledClient
	ctx   context.Context
}

var scope = log.RegisterScope("refresher", "Dynamic database refresher", 0)

func NewFlakeTester(ctx context.Context, store storage.Store, cache *cache.Cache, ght *gh.ThrottledClient, orgs []config.Org) filters.Filter {
	r := &FlakeTester{
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

func (r *FlakeTester) Events() []webhook.Event {
	return []webhook.Event{
		webhook.IssuesEvent,
		webhook.IssueCommentEvent,
		webhook.PullRequestEvent,
		webhook.PullRequestReviewEvent,
		webhook.PushEvent,
	}
}

// accept an event arriving from GitHub
func (r *FlakeTester) Handle(_ http.ResponseWriter, githubObject interface{}) {
	switch p := githubObject.(type) {
	case webhook.CheckRunPayload:
		scope.Infof("Received CheckSuitePayload: %s", p.Repository.FullName)
		if !r.repos[p.Repository.FullName] {
			scope.Infof("Ignoring ChechSuite event from repo %s since it's not in a monitored repo", p.Repository.FullName)
			return
		}
		testFlake, discoveredUsers := gh.TestFlakeFromHook(&p)
		orgID := testFlake.OrgID
		prNum := testFlake.PrNum

		prFlakeTest, err := testflakes.NewPrFlakeTest()
		if err != nil {
			scope.Errorf(err.Error())
		}
		testFlakes, errr := prFlakeTest.CheckTestFlakesForPr(prNum)

		if errr != nil {
			scope.Errorf(errr.Error())
			return
		}

		testFlakes = prFlakeTest.SetOrgID(orgID, testFlakes)

		erro := r.store.WriteTestFlakes(testFlakes)
		if erro != nil {
			scope.Errorf(erro.Error())
		}

		r.syncUsers(discoveredUsers)

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *FlakeTester) syncUsers(discoveredUsers map[string]string) {
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

		// didn't get user info from our storage layer, ask GitHub for details
		u, _, err := r.ght.Get().Users.Get(r.ctx, du)
		if err != nil {
			scope.Errorf("Unable to get info on user %s from GitHub: %v", du, err)
		} else {
			users = append(users, gh.UserFromAPI(u))
		}
	}

	if err := r.cache.WriteUsers(users); err != nil {
		scope.Errorf("Unable to write users: %v", err)
	}
}
