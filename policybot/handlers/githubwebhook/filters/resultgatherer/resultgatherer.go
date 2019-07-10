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

package resultgatherer

import (
	"context"

	s "cloud.google.com/go/storage"
	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/pkg/config"
	gatherer "istio.io/bots/policybot/pkg/resultgatherer"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type ResultGatherer struct {
	store              storage.Store
	repos              map[string]bool
	cache              *cache.Cache
	testResultGatherer *gatherer.TestResultGatherer
}

var scope = log.RegisterScope("ResultGatherer", "Result gatherer for each pr test run", 0)

func NewResultGatherer(store storage.Store,
	cache *cache.Cache, orgs []config.Org, bucketName string) filters.Filter {
	ctx := context.Background()

	client, err := s.NewClient(ctx)
	if err != nil {
		return nil
	}
	testResultGatherer, err := gatherer.NewTestResultGatherer(client, bucketName)
	if err != nil {
		scope.Errorf(err.Error())
		return nil
	}
	r := &ResultGatherer{
		store:              store,
		repos:              make(map[string]bool),
		cache:              cache,
		testResultGatherer: testResultGatherer,
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			r.repos[org.Name+"/"+repo.Name] = true
		}
	}

	return r
}

// accept an event arriving from GitHub
func (r *ResultGatherer) Handle(context context.Context, event interface{}) {
	switch p := event.(type) {
	case *github.PullRequestEvent:
		scope.Infof("Received PullRequestEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetNumber(), p.GetAction())

		if !r.repos[p.GetRepo().GetFullName()] {
			scope.Infof("Ignoring PR %d from repo %s since it's not in a monitored repo", p.PullRequest.Number, p.GetRepo().GetFullName())
			return
		}

		discoveredUsers := make([]*storage.User, 0, len(p.GetPullRequest().Assignees)+len(p.GetPullRequest().RequestedReviewers))

		repoName := p.GetRepo().GetFullName()
		orgLogin := p.GetOrganization().GetLogin()
		prNum := p.GetNumber()
		testResults, err := r.testResultGatherer.CheckTestResultsForPr(context, orgLogin, repoName, int64(prNum))
		if err != nil {
			scope.Errorf("Error: Unable to get test result for PR %d in repo %s: %v", prNum, repoName, err)
			return
		}

		if err = r.cache.WriteTestResults(context, testResults); err != nil {
			scope.Errorf(err.Error())
		}

		r.syncUsers(context, discoveredUsers)

	case *github.CheckRunEvent:
		scope.Infof("Received CheckRunEvent: %s", p.GetRepo().GetName())
		if !r.repos[p.GetRepo().GetName()] {
			scope.Infof("Ignoring CheckRun event from repo %s since it's not in a monitored repo", p.GetRepo().GetFullName())
			return
		}

		orgLogin := p.GetOrg().GetLogin()
		repoName := p.GetRepo().GetName()

		pullRequest := p.GetCheckRun().GetCheckSuite().PullRequests[0]
		discoveredUsers := make([]*storage.User, 0, len(pullRequest.Assignees)+len(pullRequest.RequestedReviewers))
		prNum := pullRequest.Number

		testResults, err := r.testResultGatherer.CheckTestResultsForPr(context, orgLogin, repoName, int64(*prNum))
		if err != nil {
			scope.Errorf("Error: Unable to get test result for PR %d in repo %s: %v", prNum, repoName, err)
			return
		}

		if err = r.cache.WriteTestResults(context, testResults); err != nil {
			scope.Errorf(err.Error())
		}

		r.syncUsers(context, discoveredUsers)

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *ResultGatherer) syncUsers(context context.Context, users []*storage.User) {
	if err := r.cache.WriteUsers(context, users); err != nil {
		scope.Errorf("Unable to write users: %v", err)
	}
}
