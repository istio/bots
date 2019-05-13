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

package server

import (
	"fmt"
	"net/http"

	"github.com/go-playground/webhooks/github"

	"istio.io/pkg/log"

	"istio.io/bots/policybot/pkg/storage"
)

type deltaCollector struct {
	store storage.Store
	hook  *github.Webhook
}

func newDeltaCollector(githubSecret string, store storage.Store) (*deltaCollector, error) {
	hook, err := github.New(github.Options.Secret(githubSecret))
	if err != nil {
		return nil, fmt.Errorf("unable to create to create webhook: %v", err)
	}

	return &deltaCollector{
		store: store,
		hook:  hook,
	}, nil
}

func (dc *deltaCollector) handle(w http.ResponseWriter, r *http.Request) {
	payload, err := dc.hook.Parse(r,
		github.IssueCommentEvent,
		github.IssuesEvent,
		github.PullRequestEvent,
		github.PullRequestReviewEvent,
		github.PullRequestReviewCommentEvent,
		github.PushEvent)
	if err != nil {
		if err != github.ErrEventNotFound {
			log.Errorf("Unable to parse GitHub webhook trigger: %v", err)
		}
		return
	}

	switch payload.(type) {
	case github.IssueCommentPayload:
		log.Info("IssueCommentPayload")
	case github.IssuesPayload:
		log.Info("IssuePayload")
	case github.PullRequestReviewCommentPayload:
		log.Info("PullRequestReviewCommentPayload")
	case github.PullRequestPayload:
		log.Info("PullRequestPayload")
	case github.PullRequestReviewPayload:
		log.Info("PullRequestReviewPayload")
	}
}
