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

package resulttester

import (
	"context"
	"net/http"
	"context"

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/testresults"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type ResultTester struct {
	store      storage.Store
	repos      map[string]bool
	cache      *cache.Cache
	ght        *gh.ThrottledClient
	ctx        context.Context
	bucketName string
}

var scope = log.RegisterScope("ResultTester", "Result tester for each pr test run", 0)

func NewResultTester(ctx context.Context, bucketName string, store storage.Store,
	cache *cache.Cache, ght *gh.ThrottledClient, orgs []config.Org) filters.Filter {
	r := &ResultTester{
		store:      store,
		repos:      make(map[string]bool),
		cache:      cache,
		ght:        ght,
		ctx:        ctx,
		bucketName: bucketName,
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			r.repos[org.Name+"/"+repo.Name] = true
		}
	}

	return r
}

func (r *ResultTester) Events() []webhook.Event {
	return []webhook.Event{
		webhook.CheckRunEvent,
	}
}

// accept an event arriving from GitHub
func (r *ResultTester) Handle(context context.Context, githubObject interface{}) {
	switch p := githubObject.(type) {
	case webhook.CheckRunPayload:
		scope.Infof("Received CheckRunPayload: %s", p.Repository.FullName)
		if !r.repos[p.Repository.FullName] {
			scope.Infof("Ignoring ChechRun event from repo %s since it's not in a monitored repo", p.Repository.FullName)
			return
		}
		testResult, discoveredUsers := gh.TestResultFromHook(&p)
		orgID := testResult.OrgID
		prNum := testResult.PrNum

		prResultTest, err := testresults.NewPrResultTester(r.bucketName)
		if err != nil {
			scope.Errorf(err.Error())
			return
		}
		testResults, errr := prResultTest.CheckTestResultsForPr(prNum, orgID)

		if errr != nil {
			scope.Errorf(errr.Error())
			return
		}

		erro := r.store.WriteTestResults(context, testResults)
		if erro != nil {
			scope.Errorf(erro.Error())
		}

		r.syncUsers(context, discoveredUsers)

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *ResultTester) syncUsers(context context.Context, discoveredUsers map[string]string) {
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