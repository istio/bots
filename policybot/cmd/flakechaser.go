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

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/flakechaser"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

func flakeChaserCmd() *cobra.Command {
	ca := config.DefaultArgs()

	ca.StartupOptions.GitHubToken = env.RegisterStringVar("GITHUB_TOKEN", ca.StartupOptions.GitHubToken, githubToken).Get()
	ca.StartupOptions.GCPCredentials = env.RegisterStringVar("GCP_CREDS", ca.StartupOptions.GCPCredentials, gcpCreds).Get()
	ca.StartupOptions.ConfigRepo = env.RegisterStringVar("CONFIG_REPO", ca.StartupOptions.ConfigRepo, configRepo).Get()
	ca.StartupOptions.ConfigFile = env.RegisterStringVar("CONFIG_FILE", ca.StartupOptions.ConfigFile, configFile).Get()

	loggingOptions := log.DefaultOptions()

	chaserCmd := &cobra.Command{
		Use:   "flakechaser",
		Short: "Manually run the test flake chaser",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.Configure(loggingOptions); err != nil {
				log.Errorf("Unable to configure logging: %v", err)
			}

			// neutralize gRPC logging since it spews out useless junk
			var dummy = dummyIoWriter{}
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

			cmd.SilenceUsage = true
			return runFlakeChaser(ca)
		},
	}

	chaserCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigRepo, "configRepo", "", ca.StartupOptions.ConfigRepo, configRepo)
	chaserCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigFile, "configFile", "", ca.StartupOptions.ConfigFile, configFile)
	chaserCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubToken, "github_token", "", ca.StartupOptions.GitHubToken, githubToken)
	chaserCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GCPCredentials, "gcp_creds", "", ca.StartupOptions.GCPCredentials, gcpCreds)

	loggingOptions.AttachCobraFlags(chaserCmd)

	return chaserCmd
}

// Runs the flake chase.
func runFlakeChaser(a *config.Args) error {
	// load the config file
	if err := a.Fetch(); err != nil {
		return fmt.Errorf("unable to load configuration file: %v", err)
	}

	creds, err := base64.StdEncoding.DecodeString(a.StartupOptions.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	ght := gh.NewThrottledClient(context.Background(), a.StartupOptions.GitHubToken)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	cache := cache.New(store, a.CacheTTL)

	h := flakechaser.New(ght, store, cache, a.FlakeChaser)
	h.Chase(context.Background())
	return nil
}
