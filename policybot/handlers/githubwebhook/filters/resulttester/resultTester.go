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
type testerState struct {
	tester *ResultTester
	ctx    context.Context
}

type ResultTester struct {
	store      storage.Store
	repos      map[string]bool
	cache      *cache.Cache
	ght        *gh.ThrottledClient
	bucketName string
}

var scope = log.RegisterScope("ResultTester", "Result tester for each pr test run", 0)

func NewResultTester(bucketName string, store storage.Store,
	cache *cache.Cache, ght *gh.ThrottledClient, orgs []config.Org) filters.Filter {
	r := &ResultTester{
		store:      store,
		repos:      make(map[string]bool),
		cache:      cache,
		ght:        ght,
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
func (ts *testerState) handle(githubObject interface{}) {
	switch p := githubObject.(type) {
	case webhook.CheckRunPayload:
		scope.Infof("Received CheckRunPayload: %s", p.Repository.FullName)
		if !ts.tester.repos[p.Repository.FullName] {
			scope.Infof("Ignoring ChechRun event from repo %s since it's not in a monitored repo", p.Repository.FullName)
			return
		}

		checkRunPayload := &p
		pullRequestPayload := checkRunPayload.CheckRun.CheckSuite.PullRequests[0]
		discoveredUsers := make(map[string]string, len(pullRequestPayload.PullRequest.Assignees)+len(pullRequestPayload.PullRequest.RequestedReviewers))

		orgID := checkRunPayload.Repository.Owner.NodeID
		repoID := checkRunPayload.Repository.NodeID
		prNum := pullRequestPayload.PullRequest.Number

		prResultTest, err := testresults.NewPrResultTester(ts.ctx, ts.tester.bucketName)
		if err != nil {
			scope.Errorf("Error: Unable to build result tester for bucket %s: %v", ts.tester.bucketName, err.Error())
			return
		}

		testResults, err := prResultTest.CheckTestResultsForPr(prNum, orgID, repoID)
		if err != nil {
			scope.Errorf("Error: Unable to get test result for PR %d in repo %s: %v", prNum, repoID, err)
			return
		}

		if err = ts.tester.cache.WriteTestResults(ts.ctx, testResults); err != nil {
			scope.Errorf("Error: Unable to write test results to Spanner: %v", err.Error())
		}

		ts.syncUsers(discoveredUsers)

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *ResultTester) Handle(context context.Context, githubObject interface{}) {
	ts := &testerState{
		ctx:    context,
		tester: r,
	}
	ts.handle(githubObject)
}

func (ts *testerState) syncUsers(discoveredUsers map[string]string) {
	var users []*storage.User
	for _, du := range discoveredUsers {
		user, err := ts.tester.cache.ReadUserByLogin(ts.ctx, du)
		if err != nil {
			scope.Warnf("unable to read user %s from storage: %v", du, err)
		}

		if user != nil {
			// we already know about this user
			continue
		}

		// didn't get user info from our storage layer, ask GitHub for details
		u, _, err := ts.tester.ght.Get(ts.ctx).Users.Get(ts.ctx, du)
		if err != nil {
			scope.Errorf("Unable to get info on user %s from GitHub: %v", du, err)
		} else {
			users = append(users, gh.UserFromAPI(u))
		}
	}

	if err := ts.tester.cache.WriteUsers(ts.ctx, users); err != nil {
		scope.Errorf("Unable to write users: %v", err)
	}
}
