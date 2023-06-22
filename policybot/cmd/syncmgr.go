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

	"cloud.google.com/go/bigquery"
	"github.com/spf13/cobra"

	"istio.io/bots/policybot/mgrs/syncmgr"
	"istio.io/bots/policybot/pkg/blobstorage/gcs"
	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/spanner"
)

func syncMgrCmd() *cobra.Command {
	syncFilter := ""

	cmd, _ := cmdutil.Run("syncmgr", "Run the GitHub state syncer", 0,
		cmdutil.ConfigPath|cmdutil.ConfigRepo|cmdutil.GitHubToken, func(reg *config.Registry, secrets *cmdutil.Secrets) error {
			return runSyncMgr(reg, secrets, syncFilter)
		})

	cmd.PersistentFlags().StringVarP(&syncFilter,
		"filter", "", "", "Comma-separated filters to limit what is synced, one or more of "+
			"[issues, prs, labels, maintainers, members, repocomments, events, testresults]")

	return cmd
}

// Runs the sync manager.
func runSyncMgr(reg *config.Registry, secrets *cmdutil.Secrets, syncFilter string) error {
	flags, err := syncmgr.ConvFilterFlags(syncFilter)
	if err != nil {
		return err
	}

	core := reg.Core()

	store, err := spanner.NewStore(context.Background(), core.SpannerDatabase)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	bq, err := bigquery.NewClient(context.Background(), core.GCPProject)
	if err != nil {
		return fmt.Errorf("unable to create BigQuery client: %v", err)
	}
	defer bq.Close()

	bs, err := gcs.NewStore(context.Background())
	if err != nil {
		return fmt.Errorf("unable to create gcs client: %v", err)
	}
	defer bs.Close()

	gc := gh.NewThrottledClient(context.Background(), secrets.GitHubToken)
	mgr := syncmgr.New(gc, store, bq, bs, reg, core.Robots)
	return mgr.Sync(context.Background(), flags, false)
}
