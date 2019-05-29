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
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v25/github"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

const (
	// flakeIssueQuery selects all the issues that haven't been updated for more than 3 days
	flakeIssueQuery = `SELECT * from Issues
	WHERE TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), UpdatedAt, DAY) > 3 AND 
				TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), CreatedAt, DAY) < 180 AND
				( REGEXP_CONTAINS(title, 'flake') OR 
					REGEXP_CONTAINS(body, 'flake') );`
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
	flakeComments := `Hey, there's no updates for this test flakes for 3 days.`
	scope.Infof("Handle request for flake chaser")
	issues, err := c.ghs.ReadIssueBySQL(flakeIssueQuery)
	if err != nil {
		scope.Errorf("Failed to read issue from Spanner: %v", err)
		return
	}
	for _, issue := range issues {
		comment := &github.IssueComment{
			Body: &flakeComments,
		}
		fmt.Printf("jianfeih debug handling issue %v", issue)
		// TODO: resolve the RepoName and OrgName from the ID.
		_, _, err := c.ght.Get().Issues.CreateComment(
			context.Background(), issue.OrgID, issue.RepoID, int(issue.Number), comment)
		if err != nil {
			scope.Errorf("Failed to create flakes nagging comments: %v", err)
		}
		return
	}
}
