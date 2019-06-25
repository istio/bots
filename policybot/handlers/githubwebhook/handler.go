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
	"fmt"
	"net/http"

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/pkg/log"
)

// Decodes and dispatches GitHub webhook calls
type handler struct {
	wh      *webhook.Webhook
	filters []filters.Filter
	events  []webhook.Event
}

func NewHandler(githubWebhookSecret string, filters ...filters.Filter) (http.Handler, error) {
	wh, err := webhook.New(webhook.Options.Secret(githubWebhookSecret))
	if err != nil {
		return nil, fmt.Errorf("unable to create webhook: %v", err)
	}

	m := make(map[webhook.Event]bool)
	for _, p := range filters {
		for _, e := range p.Events() {
			m[e] = true
		}
	}

	events := make([]webhook.Event, 0, len(m))
	for e := range m {
		events = append(events, e)
	}

	return &handler{
		wh:      wh,
		filters: filters,
		events:  events,
	}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := h.wh.Parse(r, h.events...)
	if err != nil {
		if err != webhook.ErrEventNotFound {
			log.Errorf("Unable to parse GitHub webhook trigger: %v", err)
		}
		return
	}

	// dispatch to all the registered plugins
	for _, filter := range h.filters {
		filter.Handle(r.Context(), payload)
	}
}
