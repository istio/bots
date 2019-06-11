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

	"istio.io/bots/policybot/pkg/config"

	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("flakechaser", "The GitHub flaky test chaser.", 0)

// Chaser scans the test flakiness issues and neg issuer assignee when no updates occur for a while.
type Chaser struct {
	ght  *util.GitHubThrottle
	ghs  *gh.GitHubState
	repo string
	// we select issues hasn't bee updated for last `inactiveDays`
	inactiveDays int
	// we only consider issues that are created within last `createdDays`.
	createdDays int
	// dryRun if true, will not make comments on the github.
	dryRun bool
}

// New creates a flake chaser.
func New(ght *util.GitHubThrottle, ghs *gh.GitHubState, config config.FlakeChaser) *Chaser {
	c := &Chaser{
		ght:          ght,
		ghs:          ghs,
		repo:         "istio",
		inactiveDays: config.InactiveDays,
		createdDays:  config.CreatedDays,
		dryRun:       config.DryRun,
	}
	scope.Infof("Flake chaser configuration %+v, config %+v", c, config)
	return c
}

// Handle implements http interface, will be invoked periodically to fulfil the test flakes comments.
func (c *Chaser) Handle(_ http.ResponseWriter, _ *http.Request) {
	flakeComments := `Hey, there's no updates for this test flakes for 3 days.`
	scope.Infof("Handle request for flake chaser")
	issues, err := c.ghs.ReadTestFlakyIssues(c.inactiveDays, c.createdDays)
	if err != nil {
		scope.Errorf("Failed to read issue from storage: %v", err)
		return
	}
	for _, issue := range issues {
		comment := &github.IssueComment{
			Body: &flakeComments,
		}
		repo, err := c.ghs.ReadRepo(issue.OrgID, issue.RepoID)
		if err != nil {
			scope.Errorf("Failed to look up the repo: %v", err)
			continue
		}
		org, err := c.ghs.ReadRepo(issue.OrgID, issue.RepoID)
		if err != nil {
			scope.Errorf("Failed to read the repo: %v", err)
			continue
		}
		if repo.Name != c.repo {
			continue
		}
		url := fmt.Sprintf("https://github.com/%v/%v/issues/%v", org.Name, repo.Name, issue.Number)
		scope.Infof("About to nag test flaky issue with %v", url)
		if c.dryRun {
			continue
		}
		_, _, err = c.ght.Get().Issues.CreateComment(
			context.Background(), org.Name, repo.Name, int(issue.Number), comment)
		if err != nil {
			scope.Errorf("Failed to create flakes nagging comments: %v", err)
		}
	}
}
