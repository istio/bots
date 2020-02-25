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

package flakemgr

import (
	"context"
	"fmt"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("flakemgr", "The GitHub flaky test chaser.", 0)

// FlakeManager scans the test flakiness issues and neg issuer assignee when no updates occur for a while.
type FlakeManager struct {
	gc    *gh.ThrottledClient
	cache *cache.Cache
	store storage.Store
	reg   *config.Registry
}

const nagSignature = "\n\n_Courtesy of your friendly test flake nag_."

// New creates a flake manager.
func New(gc *gh.ThrottledClient, store storage.Store, cache *cache.Cache, reg *config.Registry) *FlakeManager {
	return &FlakeManager{
		gc:    gc,
		cache: cache,
		store: store,
		reg:   reg,
	}
}

// Nag does the nagging
func (fm *FlakeManager) Nag(context context.Context, dryRun bool) error {
	for _, repo := range fm.reg.Repos() {
		for _, r := range fm.reg.Records(recordType, repo.OrgAndRepo) {
			nag := r.(*flakeNagRecord)

			if err := fm.handleNag(context, repo, nag, dryRun); err != nil {
				return err
			}
		}
	}

	return nil
}

func (fm *FlakeManager) handleNag(context context.Context, repo gh.RepoDesc, nag *flakeNagRecord, dryRun bool) error {
	issues, err := fm.store.QueryTestFlakeIssues(context, repo.OrgLogin, repo.RepoName, nag.InactiveDays, nag.CreatedDays)
	if err != nil {
		return fmt.Errorf("unable to read test flake issues from storage: %v", err)
	}

	scope.Infof("Found %v potential flake issues for repo %v", len(issues), repo)

	for _, issue := range issues {
		if dryRun {
			scope.Infof("Would have nagged issue %d from repo %v", issue.IssueNumber, repo)
			continue
		}

		if err := fm.gc.AddOrReplaceBotComment(context, repo.OrgLogin, repo.RepoName, int(issue.IssueNumber), issue.Author, nag.Message, nagSignature); err != nil {
			return fmt.Errorf("unable to create nagging comment for issue %d in repo %v: %v", issue.IssueNumber, repo, err)
		}

		scope.Infof("Nagged issue %d from repo %v", issue.IssueNumber, repo)
	}

	return nil
}
