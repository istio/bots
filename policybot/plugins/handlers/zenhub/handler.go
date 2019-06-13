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

package zenhub

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/zh"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("zenhub", "The ZenHub webhook handler", 0)

// Decodes and dispatches ZenHub webhook calls
type handler struct {
	store storage.Store
	cache *cache.Cache
}

func NewHandler(store storage.Store, cache *cache.Cache) http.Handler {
	return &handler{
		store: store,
		cache: cache,
	}
}

type typer struct {
	Type string `json:"type"`
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		scope.Errorf("Unable to read body from ZenHub event: %v", err)
		return
	}

	data := &typer{}
	if err = json.Unmarshal(body, data); err != nil {
		scope.Errorf("Unable to parse ZenHub event body: %v", err)
		return
	}

	switch data.Type {
	case "issue_transfer":
		scope.Infof("Received IssueTransferEvent from ZenHub")

		result := &zh.IssueTransferEvent{}
		if err = json.Unmarshal(body, result); err != nil {
			log.Errorf("Unable to decode ZenHub issue transfer event: %v", err)
			return
		}

		h.storePipeline(result.Organization, result.Repo, result.IssueNumber, result.ToPipelineName)

	case "issue_reprioritized_event":
		scope.Infof("Received IssueReprioritizedEvent from ZenHub")

		result := &zh.IssueReprioritizedEvent{}
		if err = json.Unmarshal(body, result); err != nil {
			log.Errorf("Unable to decode ZenHub issue reprioritization event: %v", err)
			return
		}

		h.storePipeline(result.Organization, result.Repo, result.IssueNumber, result.ToPipelineName)
	}
}

func (h *handler) storePipeline(org string, repo string, issueNumber int, pipeline string) {
	o, err := h.cache.ReadOrgByLogin(org)
	if err != nil {
		scope.Errorf("Unable to get info on organization %s: %v", org, err)
		return
	} else if o == nil {
		scope.Errorf("Organization %s was not found", org)
		return
	}

	r, err := h.cache.ReadRepoByName(o.OrgID, repo)
	if err != nil {
		scope.Errorf("Unable to get info on repo %s/%s: %v", org, repo, err)
		return
	} else if r == nil {
		scope.Errorf("Repo %s/%s was not found", org, repo)
		return
	}

	issuePipeline := &storage.IssuePipeline{
		OrgID:       r.OrgID,
		RepoID:      r.RepoID,
		IssueNumber: int64(issueNumber),
		Pipeline:    pipeline,
	}

	if err := h.store.WriteIssuePipelines([]*storage.IssuePipeline{issuePipeline}); err != nil {
		scope.Errorf("Unable to write pipeline to storage: %v", err)
	}
}
