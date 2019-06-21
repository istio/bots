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

package flakechaser

import (
	"net/http"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/flakechaser"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("flakechaser", "The GitHub flaky test chaser.", 0)

type handler struct {
	chaser *flakechaser.Chaser
}

// New creates a flake chaser.
func New(ght *gh.ThrottledClient, store storage.Store, cache *cache.Cache, config config.FlakeChaser) http.Handler {
	return &handler{
		chaser: flakechaser.New(ght, store, cache, config),
	}
}

// Handle kicks of the chaser
func (h *handler) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	scope.Infof("Handle request for flake chaser")
	h.chaser.Chase(r.Context())
}
