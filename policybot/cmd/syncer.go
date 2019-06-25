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
	"google.golang.org/grpc/grpclog"

	"istio.io/bots/policybot/pkg/blobstorage/gcs"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/syncer"
	"istio.io/bots/policybot/pkg/zh"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

func syncerCmd() *cobra.Command {
	ca := config.DefaultArgs()

	ca.StartupOptions.GitHubToken = env.RegisterStringVar("GITHUB_TOKEN", ca.StartupOptions.GitHubToken, githubToken).Get()
	ca.StartupOptions.ZenHubToken = env.RegisterStringVar("ZENHUB_TOKEN", ca.StartupOptions.ZenHubToken, zenhubToken).Get()
	ca.StartupOptions.GCPCredentials = env.RegisterStringVar("GCP_CREDS", ca.StartupOptions.GCPCredentials, gcpCreds).Get()
	ca.StartupOptions.ConfigRepo = env.RegisterStringVar("CONFIG_REPO", ca.StartupOptions.ConfigRepo, configRepo).Get()
	ca.StartupOptions.ConfigFile = env.RegisterStringVar("CONFIG_FILE", ca.StartupOptions.ConfigFile, configFile).Get()

	loggingOptions := log.DefaultOptions()
	var filters string

	syncerCmd := &cobra.Command{
		Use:   "syncer",
		Short: "Manually run the GitHub/ZenHub state syncer",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.Configure(loggingOptions); err != nil {
				log.Errorf("Unable to configure logging: %v", err)
			}

			// neutralize gRPC logging since it spews out useless junk
			var dummy = dummyIoWriter{}
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

			cmd.SilenceUsage = true
			return runSyncer(ca, filters)
		},
	}

	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigRepo, "configRepo", "", ca.StartupOptions.ConfigRepo, configRepo)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigFile, "configFile", "", ca.StartupOptions.ConfigFile, configFile)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubToken, "github_token", "", ca.StartupOptions.GitHubToken, githubToken)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ZenHubToken, "zenhub_token", "", ca.StartupOptions.ZenHubToken, zenhubToken)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GCPCredentials, "gcp_creds", "", ca.StartupOptions.GCPCredentials, gcpCreds)

	syncerCmd.PersistentFlags().StringVarP(&filters,
		"filter", "", "", "Comma-separated filters to limit what is synced, one or more of [issues, prs, labels, maintainers, members, zenhub, repocomments]")

	loggingOptions.AttachCobraFlags(syncerCmd)

	return syncerCmd
}

// Runs the syncer.
func runSyncer(a *config.Args, filters string) error {
	flags, err := syncer.ConvFilterFlags(filters)
	if err != nil {
		return err
	}

	// load the config file
	if err := a.Fetch(); err != nil {
		return fmt.Errorf("unable to load configuration file: %v", err)
	}

	creds, err := base64.StdEncoding.DecodeString(a.StartupOptions.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	ght := gh.NewThrottledClient(context.Background(), a.StartupOptions.GitHubToken)
	zht := zh.NewThrottledClient(a.StartupOptions.ZenHubToken)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	bs, err := gcs.NewStore(context.Background(), creds)
	if err != nil {
		return fmt.Errorf("unable to create blob storage lsyer: %v", err)
	}
	defer bs.Close()

	cache := cache.New(store, a.CacheTTL)

	h := syncer.New(ght, cache, zht, store, bs, a.Orgs)
	return h.Sync(context.Background(), flags)
}
