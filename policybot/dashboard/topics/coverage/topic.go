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

//go:generate ../../../scripts/gen_topic.sh

package coverage

import (
	"net/http"

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

type topic struct {
	store storage.Store
	cache *cache.Cache
}

func NewTopic(store storage.Store, cache *cache.Cache) dashboard.Topic {
	return &topic{
		store: store,
		cache: cache,
	}
}

func (t *topic) Title() string {
	return "Code Coverage"
}

func (t *topic) Description() string {
	return "Understand Istio code coverage."
}

func (t *topic) Name() string {
	return "coverage"
}

func (t *topic) Configure(htmlRouter *mux.Router, apiRouter *mux.Router, context dashboard.RenderContext, opt *dashboard.Options) {
	page := string(MustAsset("page.html"))

	htmlRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			context.RenderHTML(w, page)
		})

	apiRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			context.RenderJSON(w, http.StatusOK, nil)
		})
}
