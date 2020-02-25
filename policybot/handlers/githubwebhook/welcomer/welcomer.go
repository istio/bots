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

package welcomer

import (
	"context"
	"time"

	"github.com/google/go-github/v26/github"

	"istio.io/pkg/log"

	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

// Inserts comments into PRs for new or infrequently seen contributors.
type Welcomer struct {
	store storage.Store
	cache *cache.Cache
	gc    *gh.ThrottledClient
	reg   *config.Registry
}

const welcomeSignature = "\n\n_Courtesy of your friendly welcome wagon_."

var scope = log.RegisterScope("welcomer", "The Istio welcome wagon", 0)

func NewWelcomer(gc *gh.ThrottledClient, store storage.Store, cache *cache.Cache, reg *config.Registry) githubwebhook.Filter {
	return &Welcomer{
		store: store,
		cache: cache,
		gc:    gc,
		reg:   reg,
	}
}

// process an event arriving from GitHub
func (w *Welcomer) Handle(context context.Context, event interface{}) {
	prp, ok := event.(*github.PullRequestEvent)
	if !ok {
		// not what we're looking for
		scope.Debugf("Unknown event received: %T %+v", event, event)
		return
	}

	scope.Infof("Received PullRequestEvent: %s, %d, %s", prp.GetRepo().GetFullName(), prp.GetPullRequest().GetNumber(), prp.GetAction())

	action := prp.GetAction()
	if action != "opened" {
		scope.Infof("Ignoring event for PR %d from repo %s since it doesn't have a supported action: %s", prp.GetNumber(), prp.GetRepo().GetFullName(), action)
		return
	}

	// see if the PR is in a repo we're monitoring
	welcome, ok := w.reg.SingleRecord(recordType, prp.GetRepo().GetFullName())
	if !ok {
		scope.Errorf("Ignoring event for PR %d from repo %s since there are no matching welcome message", prp.GetNumber(), prp.GetRepo().GetFullName())
		return
	}

	// NOTE: this assumes the PR state has already been stored by the refresher filter
	pr, err := w.cache.ReadPullRequest(context, prp.GetRepo().GetOwner().GetLogin(), prp.GetRepo().GetName(), prp.GetPullRequest().GetNumber())
	if err != nil {
		scope.Errorf("Unable to retrieve data from storage for PR %d from repo %s: %v", prp.GetNumber(), prp.GetRepo().GetFullName(), err)
		return
	}

	scope.Infof("Processing PR %d from repo %s", prp.GetNumber(), prp.GetRepo().GetFullName())

	w.processPR(context, pr, welcome.(*welcomeRecord))
}

// process a PR
func (w *Welcomer) processPR(context context.Context, pr *storage.PullRequest, welcome *welcomeRecord) {
	latest := time.Time{}

	if err := w.store.QueryPullRequestsByUser(context, pr.OrgLogin, pr.RepoName, pr.Author, func(prResult *storage.PullRequest) error {
		if pr.PullRequestNumber != prResult.PullRequestNumber && prResult.CreatedAt.After(latest) {
			latest = prResult.CreatedAt
		}

		return nil
	}); err != nil {
		scope.Errorf("Unable to query storage for PRs in repo %s/%s: %v", pr.OrgLogin, pr.RepoName, err)
	}

	if time.Since(latest) > time.Hour*24*time.Duration(welcome.ResendDays) {
		if err := w.gc.AddOrReplaceBotComment(context, pr.OrgLogin, pr.RepoName, int(pr.PullRequestNumber), pr.Author, welcome.Message,
			welcomeSignature); err != nil {
			scope.Errorf("Unable to add comment to PR %d in repo %s/%s: %v", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
		}
	}
}
