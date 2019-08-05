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
	"time"

	"github.com/google/go-github/v26/github"
	"istio.io/pkg/cache"
)

var shaToPRCache = cache.NewTTL(3*time.Hour, 1*time.Minute)

// GetPRNumberForSHA fetches the associated pull request for a commit sha using
// the Github Search API.
func GetPRNumberForSHA(context context.Context, gc *ThrottledClient, sha string) (int64, error) {
	val, ok := shaToPRCache.Get(sha)
	if ok {
		return val.(int64), nil
	}
	resp, _, err := gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Search.Issues(context, sha, nil)
	})
	if err != nil {
		return 0, err
	}
	issues := resp.(*github.IssuesSearchResult).Issues
	if len(issues) == 0 {
		return 0, fmt.Errorf("no pull requests found for commit %s", sha)
	}
	prNum := int64(issues[0].GetNumber())
	shaToPRCache.Set(sha, prNum)
	return prNum, nil
}
