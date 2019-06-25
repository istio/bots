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

package zenhubwebhook

import (
	"context"
	"net/http"
	"strconv"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
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

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		scope.Errorf("Unable to parse ZenHub webhook data: %v", err)
		return
	}

	t := r.Form.Get("type")
	if t == "issue_transfer" || t == "issue_reprioritized" {
		scope.Infof("Received %s from ZenHub", t)

		num, _ := strconv.Atoi(r.Form.Get("issue_number"))
		h.storePipeline(
			r.Context(),
			r.Form.Get("organization"),
			r.Form.Get("repo"),
			num,
			r.Form.Get("to_pipeline_name"))
	}
}

func (h *handler) storePipeline(context context.Context, org string, repo string, issueNumber int, pipeline string) {
	o, err := h.cache.ReadOrgByLogin(context, org)
	if err != nil {
		scope.Errorf("Unable to get info on organization %s: %v", org, err)
		return
	} else if o == nil {
		scope.Errorf("Organization %s was not found", org)
		return
	}

	r, err := h.cache.ReadRepoByName(context, o.OrgID, repo)
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

	if err := h.store.WriteIssuePipelines(context, []*storage.IssuePipeline{issuePipeline}); err != nil {
		scope.Errorf("Unable to write pipeline to storage: %v", err)
	}
}
