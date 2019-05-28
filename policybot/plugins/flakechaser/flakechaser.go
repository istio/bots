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

package flakechaser

import (
	"istio.io/bots/policybot/pkg/util"
	"net/http"
)

// Chaser scans the test flakiness issues and neg issuer assignee when no updates occur for a while.
type Chaser struct {
	ght    *util.GitHubThrottle
	repo string
}

// New creates a flake chaser.
func New(ght *util.GitHubThrottle, repo string) (*Chaser, error) {
	return &Chaser{
		repo: repo,
		ght: ght,
	}, nil
}

// Handle implements http interface, will be invoked periodically to fullfil the test flakes comments.
func (c *Chaser) Handle(_ http.ResponseWriter, _ *http.Request) {
}
