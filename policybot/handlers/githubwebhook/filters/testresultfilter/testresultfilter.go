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

package testresultfilter

import (
	"context"
	"fmt"
	"time"

	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/gh"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/pkg/config"
	gatherer "istio.io/bots/policybot/pkg/resultgatherer"
	"istio.io/bots/policybot/pkg/storage/cache"
	ic "istio.io/pkg/cache"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type TestResultFilter struct {
	repos        map[string]gatherer.TestResultGatherer
	cache        *cache.Cache
	gc           *gh.ThrottledClient
	shaToPrCache ic.ExpiringCache
}

var scope = log.RegisterScope("testresultfilter", "Result filter for each pr test run", 0)

func NewTestResultFilter(cache *cache.Cache, orgs []config.Org, gc *gh.ThrottledClient, client blobstorage.Store) filters.Filter {
	r := &TestResultFilter{
		repos:        make(map[string]gatherer.TestResultGatherer),
		cache:        cache,
		gc:           gc,
		shaToPrCache: ic.NewTTL(3*time.Hour, 1*time.Minute),
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			r.repos[org.Name+"/"+repo.Name] = gatherer.TestResultGatherer{client, org.BucketName, org.PreSubmitTestPath, org.PostSubmitTestPath}
		}
	}

	return r
}

// accept an event arriving from GitHub
func (r *TestResultFilter) Handle(context context.Context, event interface{}) {
	switch p := event.(type) {
	case *github.PullRequestEvent:
		scope.Infof("Received PullRequestEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetNumber(), p.GetAction())
		gatherer, ok := r.repos[p.GetRepo().GetFullName()]
		if !ok {
			scope.Infof("Ignoring PR %d from repo %s since it's not in a monitored repo", p.PullRequest.Number, p.GetRepo().GetFullName())
			return
		}
		repoName := p.GetRepo().GetFullName()
		orgLogin := p.GetOrganization().GetLogin()
		prNum := p.GetNumber()
		testResults, err := gatherer.CheckTestResultsForPr(context, orgLogin, repoName, int64(prNum))
		if err != nil {
			scope.Errorf("Error: Unable to get test result for PR %d in repo %s: %v", prNum, repoName, err)
			return
		}

		if err = r.cache.WriteTestResults(context, testResults); err != nil {
			scope.Errorf(err.Error())
		}

	case *github.StatusEvent:
		scope.Infof("Received StatusEvent: %s", p.GetRepo().GetFullName())
		val, ok := r.repos[p.GetRepo().GetFullName()]
		if !ok {
			scope.Infof("Ignoring StatusEvent from repo %s since it's not in a monitored repo", p.GetRepo().GetFullName())
			return
		}

		if p.GetState() == "pending" {
			scope.Infof("Ignoring StatusEvent from repo %s because it's pending", p.GetRepo().GetFullName())
			return
		}

		orgLogin := p.GetRepo().GetOwner().GetLogin()
		repoName := p.GetRepo().GetName()

		sha := p.GetCommit().GetSHA()
		prNum, err := r.getPrNumForSha(context, sha)
		if err != nil {
			scope.Errorf("Error fetching pull request info for commit %s: %v", sha, err)
		}
		scope.Infof("Commit %s corresponds to pull request %d.", sha, prNum)

		testResults, err := val.CheckTestResultsForPr(context, orgLogin, repoName, prNum)
		if err != nil {
			scope.Errorf("Error: Unable to get test result for PR %d in repo %s: %v", prNum, repoName, err)
			return
		}

		if err = r.cache.WriteTestResults(context, testResults); err != nil {
			scope.Errorf(err.Error())
		}

	default:
		// not what we're looking for
		scope.Debugf("Unknown payload received: %T %+v", p, p)
		return
	}
}

func (r *TestResultFilter) getPrNumForSha(context context.Context, sha string) (int64, error) {
	val, ok := r.shaToPrCache.Get(sha)
	if ok {
		return val.(int64), nil
	}
	resp, _, err := r.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Search.Issues(context, sha, nil)
	})
	if err != nil {
		return 0, err
	}
	issues := resp.(*github.IssuesSearchResult).Issues
	if len(issues) == 0 {
		return 0, fmt.Errorf("No pull requests found for commit %s", sha)
	}
	prNum := int64(issues[0].GetNumber())
	r.shaToPrCache.Set(sha, prNum)
	return prNum, nil
}
