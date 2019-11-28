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

	"github.com/spf13/cobra"

	"istio.io/bots/policybot/mgrs/milestonemgr"
	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
)

func milestoneMgrCmd() *cobra.Command {
	cmd, _ := cmdutil.Run("milestonemgr", "Run the milestone manager", 0,
		cmdutil.ConfigPath|cmdutil.ConfigRepo|cmdutil.GitHubToken, runMilestoneMgr)

	return cmd
}

func runMilestoneMgr(reg *config.Registry, secrets *cmdutil.Secrets) error {
	gc := gh.NewThrottledClient(context.Background(), secrets.GitHubToken)
	mgr := milestonemgr.New(gc, reg)
	return mgr.MakeConfiguredMilestones(context.Background(), false)
}
