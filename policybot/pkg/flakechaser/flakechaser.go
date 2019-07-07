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

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("flakechaser", "The GitHub flaky test chaser.", 0)

// Chaser scans the test flakiness issues and neg issuer assignee when no updates occur for a while.
type Chaser struct {
	gc    *gh.ThrottledClient
	cache *cache.Cache
	store storage.Store
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
func New(gc *gh.ThrottledClient, store storage.Store, cache *cache.Cache, config config.FlakeChaser) *Chaser {
	enabledRepo := map[string]bool{}
	for _, repo := range config.Repos {
		enabledRepo[repo] = true
	}
	return &Chaser{
		gc:           gc,
		store:        store,
		cache:        cache,
		repos:        enabledRepo,
		inactiveDays: config.InactiveDays,
		createdDays:  config.CreatedDays,
		dryRun:       config.DryRun,
		message:      config.Message,
	}
}

// Chase does the nagging
func (c *Chaser) Chase(context context.Context) {
	issues, err := c.store.QueryTestFlakeIssues(context, c.inactiveDays, c.createdDays)
	if err != nil {
		scope.Errorf("Failed to read issue from storage: %v", err)
		return
	}

	scope.Infof("Found %v potential issues", len(issues))
	for _, issue := range issues {
		comment := &github.IssueComment{
			Body: &c.message,
		}
		repo, err := c.cache.ReadRepo(context, issue.OrgLogin, issue.RepoName)
		if err != nil {
			scope.Errorf("Failed to look up the repo: %v", err)
			continue
		}
		org, err := c.cache.ReadOrg(context, issue.OrgLogin)
		if err != nil {
			scope.Errorf("Failed to read the repo: %v", err)
			continue
		}
		repoURI := fmt.Sprintf("%v/%v", org.OrgLogin, repo.RepoName)
		if _, ok := c.repos[repoURI]; !ok {
			scope.Infof("Uninterested repo %v, skipping...", repoURI)
			continue
		}
		url := fmt.Sprintf("https://github.com/%v/%v/issues/%v", org.OrgLogin, repo.RepoName, issue.IssueNumber)
		scope.Infof("About to nag test flaky issue with %v", url)
		if c.dryRun {
			continue
		}

		_, _, err = c.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.CreateComment(
				context, org.OrgLogin, repo.RepoName, int(issue.IssueNumber), comment)
		})

		if err != nil {
			scope.Errorf("Failed to create flakes nagging comments: %v", err)
		}
	}
}
