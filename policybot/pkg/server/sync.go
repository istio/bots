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

package server

import (
	"context"
	"encoding/base64"
	"fmt"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/bots/policybot/plugins/handlers/syncer"
)

// Runs the syncer.
//
// If config comes from a container-based file, this will try to run the server, but if
// problems occur (probably due to bad config), then the function returns with an error.
//
// If config comes from a repo-based file, this will also try to run the server, but if an error
// occurs, it will refetch the config every minute and try again. And so in that case, this
// function never returns.
func Sync(a *config.Args, filters string) error {
	// load the config file
	if err := fetchConfig(a); err != nil {
		return fmt.Errorf("unable to load configuration file: %v", err)
	}

	creds, err := base64.StdEncoding.DecodeString(a.StartupOptions.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	ght := util.NewGitHubThrottle(context.Background(), a.StartupOptions.GitHubToken)
	zht := util.NewZenHubThrottle(context.Background(), a.StartupOptions.ZenHubToken)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	ghs := gh.NewGitHubState(store, a.CacheTTL)

	h := syncer.NewHandler(context.Background(), ght, ghs, zht, store, a.Orgs).(*syncer.Syncer)
	return h.Sync(filters)
}
