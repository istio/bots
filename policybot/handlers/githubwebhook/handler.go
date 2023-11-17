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

package githubwebhook

import (
	"context"
	"net/http"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/util"
	"istio.io/istio/pkg/log"
)

var scope = log.RegisterScope("githubwebhook", "GitHub webhook handler")

// Decodes and dispatches GitHub webhook calls
type handler struct {
	secret  []byte
	filters []Filter
}

// The interface to a GitHub webhook filter.
//
// Note that individual filters are invoked for any events incoming to the
// bot.
type Filter interface {
	Handle(context context.Context, event interface{})
}

func NewHandler(githubWebhookSecret string, filters ...Filter) http.Handler {
	return &handler{
		secret:  []byte(githubWebhookSecret),
		filters: filters,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, h.secret)
	if err != nil {
		util.RenderError(w, err)
		return
	}

	scope.Debugf("Received GitHub event: %v", github.WebHookType(r))

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		util.RenderError(w, err)
		return
	}

	// dispatch to all the registered filters
	for _, filter := range h.filters {
		filter.Handle(r.Context(), event)
	}
}
