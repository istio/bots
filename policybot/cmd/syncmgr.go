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

	"github.com/spf13/cobra"

	"istio.io/bots/policybot/mgrs/syncmgr"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/zh"
)

func syncMgrCmd() *cobra.Command {
	cmd, ca := config.Run("syncmgr", "Run the GitHub+ZenHub state syncer", 0,
		config.ConfigFile|config.ConfigRepo|config.ZenhubToken|config.GitHubToken|config.GCPCreds, runSyncMgr)

	cmd.PersistentFlags().StringVarP(&ca.SyncerFilter,
		"filter", "", "", "Comma-separated filters to limit what is synced, one or more of "+
			"[issues, prs, labels, maintainers, members, zenhub, repocomments, events, testresults]")

	return cmd
}

// Runs the syncer.
func runSyncMgr(a *config.Args, args []string) error {
	flags, err := syncmgr.ConvFilterFlags(a.SyncerFilter)
	if err != nil {
		return err
	}

	creds, err := base64.StdEncoding.DecodeString(a.Secrets.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	gc := gh.NewThrottledClient(context.Background(), a.Secrets.GitHubToken)
	zc := zh.NewThrottledClient(a.Secrets.ZenHubToken)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	mgr, err := syncmgr.New(gc, creds, a.GCPProject, zc, store, a.Orgs, a.Robots)
	if err != nil {
		return fmt.Errorf("unable to create syncer: %v", err)
	}

	return mgr.Sync(context.Background(), flags)
}
