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

// Take in a pr number from blob storage and examines the pr
// for all tests that are run and their results. The results are then written to storage.

package gh

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v26/github"

	"istio.io/pkg/cache"
)

var shaToPRCache = cache.NewTTL(time.Hour, time.Minute)

// GetPRForSHA fetches the associated pull request for a commit sha using
// the Github Search API. This method can return a cached pull request object,
// so it is only useful for fairly static in formation, such as the pull
// request number, the PR base information, etc.
func GetPRForSHA(context context.Context, gc *ThrottledClient, sha string) (*github.PullRequest, error) {
	val, ok := shaToPRCache.Get(sha)
	if ok {
		return val.(*github.PullRequest), nil
	}
	resp, _, err := gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Search.Issues(context, sha, nil)
	})
	if err != nil {
		return nil, err
	}
	issues := resp.(*github.IssuesSearchResult).Issues
	if len(issues) == 0 {
		return nil, fmt.Errorf("no pull requests found for commit %s", sha)
	}
	// The returned issue doesn't have nice fields for owner/repo.
	repoURL := issues[0].GetRepositoryURL()
	parts := strings.Split(repoURL, "/")
	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]
	resp, _, err = gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.PullRequests.Get(context, owner, repo, issues[0].GetNumber())
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching pull request for commit %s: %v", sha, err)
	}
	shaToPRCache.Set(sha, resp)
	return resp.(*github.PullRequest), nil
}
