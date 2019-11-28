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

package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"istio.io/bots/policybot/mgrs/flakemgr"
	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/storage/spanner"
)

func flakeMgrCmd() *cobra.Command {
	cmd, _ := cmdutil.Run("flakemgr", "Run the test flake manager", 0,
		cmdutil.ConfigPath|cmdutil.ConfigRepo|cmdutil.GitHubToken|cmdutil.GCPCreds, runFlakeMgr)

	return cmd
}

// Runs the flake manager.
func runFlakeMgr(reg *config.Registry, secrets *cmdutil.Secrets) error {
	creds, err := base64.StdEncoding.DecodeString(secrets.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	core := reg.Core()

	store, err := spanner.NewStore(context.Background(), core.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	gc := gh.NewThrottledClient(context.Background(), secrets.GitHubToken)
	c := cache.New(store, time.Duration(core.CacheTTL))
	mgr := flakemgr.New(gc, store, c, reg)
	return mgr.Nag(context.Background(), false)
}
