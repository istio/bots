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

package util

import (
	"context"

	"github.com/google/go-github/v25/github"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

const (
	// TODO: need to enforce this
	// maxGitHubRequestsPerHour   = 5000
	maxGitHubRequestsPerSecond = 5
)

type GitHubThrottle struct {
	ctx     context.Context
	client  *github.Client
	limiter *rate.Limiter
}

func NewGitHubThrottle(ctx context.Context, githubToken string) *GitHubThrottle {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)

	return &GitHubThrottle{
		ctx:     ctx,
		client:  github.NewClient(oauth2.NewClient(ctx, src)),
		limiter: rate.NewLimiter(maxGitHubRequestsPerSecond, 100),
	}
}

// Get the GitHub client in a throttled fashion, so we don't exceed GitHub's 5000 request/hour limit
func (ght *GitHubThrottle) Get() *github.Client {
	_ = ght.limiter.Wait(ght.ctx)
	return ght.client
}
