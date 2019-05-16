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

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/pkg/log"

	"istio.io/bots/policybot/pkg/storage"
)

type deltaCollector struct {
	store  storage.Store
	hook   *webhook.Webhook
	nagger *testNagger
}

func newDeltaCollector(githubSecret string, store storage.Store, nagger *testNagger) (*deltaCollector, error) {
	hook, err := webhook.New(webhook.Options.Secret(githubSecret))
	if err != nil {
		return nil, fmt.Errorf("unable to create webhook: %v", err)
	}

	return &deltaCollector{
		store:  store,
		hook:   hook,
		nagger: nagger,
	}, nil
}

func (dc *deltaCollector) handle(w http.ResponseWriter, r *http.Request) {
	payload, err := dc.hook.Parse(r,
		webhook.IssueCommentEvent,
		webhook.IssuesEvent,
		webhook.PullRequestEvent,
		webhook.PullRequestReviewEvent,
		webhook.PullRequestReviewCommentEvent,
		webhook.PushEvent)
	if err != nil {
		if err != webhook.ErrEventNotFound {
			log.Errorf("Unable to parse GitHub webhook trigger: %v", err)
		}
		return
	}

	switch p := payload.(type) {
	case webhook.IssueCommentPayload:
	case webhook.IssuesPayload:
	case webhook.PullRequestReviewCommentPayload:
	case webhook.PullRequestPayload:
		dc.nagger.handleNewPR(&p)
	case webhook.PullRequestReviewPayload:
	default:
		log.Errorf("Unrecognized payload type: %T, %+v", payload, payload)
	}
}
