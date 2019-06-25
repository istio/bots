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

package gh

import (
	"context"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

const (
	// TODO: need to enforce this
	// maxGitHubRequestsPerHour   = 5000.0     // per-hour max, to stay under rate limit
	maxGitHubRequestsPerSecond = 1.1 // per-second max, to stay under abuse detection limit
	maxGitHubBurst             = 10  // max burst size, to stay under abuse detection limit
)

// ThrottledClient is used to throttle our use of the GitHub API in order to
// prevent hitting rate limits or abuse limits.
type ThrottledClient struct {
	client  *github.Client
	limiter *rate.Limiter
}

func NewThrottledClient(context context.Context, githubToken string) *ThrottledClient {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)

	return &ThrottledClient{
		client:  github.NewClient(oauth2.NewClient(context, src)),
		limiter: rate.NewLimiter(maxGitHubRequestsPerSecond, maxGitHubBurst),
	}
}

// Get the GitHub client in a throttled fashion, so we don't exceed GitHub's usage limits. This will block
// until it is safe to make the call to GitHub.
func (ght *ThrottledClient) Get(context context.Context) *github.Client {
	_ = ght.limiter.Wait(context)
	return ght.client
}
