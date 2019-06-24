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

package commithub

import (
	"html/template"
	"net/http"

	"istio.io/bots/policybot/pkg/fw"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

type Topic struct {
	store storage.Store
	cache *cache.Cache
}

func NewTopic(store storage.Store, cache *cache.Cache) fw.Topic {
	return &Topic{
		store: store,
		cache: cache,
	}
}

func (t *Topic) Title() string {
	return "Commit Hub"
}

func (t *Topic) Description() string {
	return "Interact with pull requests and commits."
}

func (t *Topic) Prefix() string {
	return "commithub"
}

func (t *Topic) Activate(context fw.TopicContext) {
	tmpl := template.Must(context.Layout().Parse(commitHubTemplate))

	_ = context.HTMLRouter().StrictSlash(true).NewRoute().Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fw.RenderHTML(w, tmpl, nil)
	})

	_ = context.JSONRouter().StrictSlash(true).NewRoute().Methods("GET").Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fw.RenderJSON(w, http.StatusOK, nil)
	})
}
