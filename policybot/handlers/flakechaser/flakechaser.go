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
	"istio.io/bots/policybot/pkg/storage/cache"

	"istio.io/bots/policybot/pkg/gh"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("flakechaser", "The GitHub flaky test chaser.", 0)

// Chaser scans the test flakiness issues and neg issuer assignee when no updates occur for a while.
type Chaser struct {
	ght   *gh.ThrottledClient
	cache *cache.Cache
	repos map[string]bool
	// we select issues hasn't bee updated for last `inactiveDays`
	inactiveDays int
	// we only consider issues that are created within last `createdDays`.
	createdDays int
	// dryRun if true, will not make comments on the github.
	dryRun bool
	// message is what the bot will post on the github issue.
	message string
}

// New creates a flake chaser.
func New(ght *gh.ThrottledClient, cache *cache.Cache, config config.FlakeChaser) *Chaser {
	enabledRepo := map[string]bool{}
	for _, repo := range config.Repos {
		enabledRepo[repo] = true
	}
	c := &Chaser{
		ght:          ght,
		cache:        cache,
		repos:        enabledRepo,
		inactiveDays: config.InactiveDays,
		createdDays:  config.CreatedDays,
		dryRun:       config.DryRun,
		message:      config.Message,
	}
	return c
}

// Handle implements http interface, will be invoked periodically to fulfill the test flakes comments.
func (c *Chaser) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	scope.Infof("Handle request for flake chaser")
	issues, err := c.cache.ReadTestFlakyIssues(c.inactiveDays, c.createdDays)
	if err != nil {
		scope.Errorf("Failed to read issue from storage: %v", err)
		return
	}
	scope.Infof("Found %v potential issues", len(issues))
	for _, issue := range issues {
		comment := &github.IssueComment{
			Body: &c.message,
		}
		repo, err := c.cache.ReadRepo(issue.OrgID, issue.RepoID)
		if err != nil {
			scope.Errorf("Failed to look up the repo: %v", err)
			continue
		}
		org, err := c.cache.ReadOrg(issue.OrgID)
		if err != nil {
			scope.Errorf("Failed to read the repo: %v", err)
			continue
		}
		repoURI := fmt.Sprintf("%v/%v", org.Login, repo.Name)
		if _, ok := c.repos[repoURI]; !ok {
			scope.Infof("Uninterested repo %v, skipping...", repoURI)
			continue
		}
		url := fmt.Sprintf("https://github.com/%v/%v/issues/%v", repo.Name, repo.Name, issue.Number)
		scope.Infof("About to nag test flaky issue with %v", url)
		if c.dryRun {
			continue
		}
		_, _, err = c.ght.Get().Issues.CreateComment(
			context.Background(), org.Login, repo.Name, int(issue.Number), comment)
		if err != nil {
			scope.Errorf("Failed to create flakes nagging comments: %v", err)
		}
	}
}
