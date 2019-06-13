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

package zh

import (
	"strconv"
	"time"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/cache"
)

type ZenHubState struct {
	store         storage.Store
	pipelineCache cache.ExpiringCache
}

func NewZenHubState(store storage.Store, entryTTL time.Duration) *ZenHubState {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if entryTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = entryTTL / 2
	}

	return &ZenHubState{
		store:         store,
		pipelineCache: cache.NewTTL(entryTTL, evictionInterval),
	}
}

// Reads from cache and if not found reads from DB
func (zhs *ZenHubState) ReadIssuePipeline(orgID string, repoID string, issueNumber int) (*storage.IssuePipeline, error) {
	key := orgID + repoID + strconv.Itoa(issueNumber)
	if value, ok := zhs.pipelineCache.Get(key); ok {
		return value.(*storage.IssuePipeline), nil
	}

	result, err := zhs.store.ReadIssuePipelineByNumber(orgID, repoID, issueNumber)
	if err == nil {
		zhs.pipelineCache.Set(key, result)
	}

	return result, err
}
