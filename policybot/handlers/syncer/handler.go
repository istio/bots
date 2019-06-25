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

package syncer

import (
	"context"
	"net/http"

	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/syncer"
	"istio.io/bots/policybot/pkg/zh"
)

type handler struct {
	syncer *syncer.Syncer
}

func NewHandler(ctx context.Context, ght *gh.ThrottledClient, cache *cache.Cache,
	zht *zh.ThrottledClient, store storage.Store, bs blobstorage.Store, orgs []config.Org) http.Handler {
	return &handler{
		syncer: syncer.New(ght, cache, zht, store, bs, orgs),
	}
}

func (h *handler) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	flags, err := syncer.ConvFilterFlags(r.URL.Query().Get("filter"))
	if err != nil {
		// TODO: render error
		_ = err
		return
	}

	if err = h.syncer.Sync(r.Context(), flags); err != nil {
		// TODO: render error
		_ = err
	}
}
