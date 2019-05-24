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

// Package gh exposes a GitHub persistent object store.
package gh

import (
	"time"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/cache"
)

// Cached access over our GitHub object store.
type GitHubState struct {
	cache cache.ExpiringCache
	store storage.Store
}

func NewGitHubState(store storage.Store, entryTTL time.Duration) *GitHubState {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if entryTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = entryTTL / 2
	}

	return &GitHubState{
		cache: cache.NewTTL(entryTTL, evictionInterval),
		store: store,
	}
}
