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
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"istio.io/bots/policybot/mgrs/lifecyclemgr"
	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/storage/spanner"
)

func lifecycleMgrCmd() *cobra.Command {
	cmd, _ := cmdutil.Run("lifecyclemgr", "Runs the issue and pull request lifecycle manager", 0,
		cmdutil.ConfigPath|cmdutil.ConfigRepo|cmdutil.GitHubToken, runLifecycleMgr)

	return cmd
}

func runLifecycleMgr(reg *config.Registry, secrets *cmdutil.Secrets) error {
	core := reg.Core()

	store, err := spanner.NewStore(context.Background(), core.SpannerDatabase)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	gc := gh.NewThrottledClient(context.Background(), secrets.GitHubToken)
	c := cache.New(store, time.Duration(core.CacheTTL))
	mgr := lifecyclemgr.New(gc, store, c, reg)
	return mgr.ManageAll(context.Background(), false)
}
