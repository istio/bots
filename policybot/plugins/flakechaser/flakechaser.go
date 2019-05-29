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
	"net/http"

	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

const (
	// TODO, rewrite query to be within 60 days of now to reduce the query size.
	// created timestamp comparision instead...
	// https://cloud.google.com/spanner/docs/functions-and-operators#timestamp_diff
	query = `SELECT OrgID, IssueID, Title, UpdatedAt from Issues
WHERE TIMESTAMP("2019-04-25 15:30:00", "America/Los_Angeles") < UpdatedAt
	AND REGEXP_CONTAINS(title, 'flake');`
)

var scope = log.RegisterScope("flakechaser", "Listens for changes in policybot config", 0)

// Chaser scans the test flakiness issues and neg issuer assignee when no updates occur for a while.
type Chaser struct {
	ght  *util.GitHubThrottle
	ghs  *gh.GitHubState
	repo string
}

// New creates a flake chaser.
func New(ght *util.GitHubThrottle, ghs *gh.GitHubState, repo string) (*Chaser, error) {
	return &Chaser{
		repo: repo,
		ght:  ght,
		ghs:  ghs,
	}, nil
}

// Handle implements http interface, will be invoked periodically to fullfil the test flakes comments.
func (c *Chaser) Handle(_ http.ResponseWriter, _ *http.Request) {
	scope.Infof("Handle request for flake chaser")
	// TODO, add handler function to post updates.
	issues, err := c.ghs.ReadIssueBySQL(query, nil)
	if err != nil {
		scope.Errorf("Failed to read issue from Spanner: %v", err)
		return
	}
}
