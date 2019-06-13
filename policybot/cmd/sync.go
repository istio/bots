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
	"github.com/spf13/cobra"
	"google.golang.org/grpc/grpclog"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/server"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

func syncCmd() *cobra.Command {
	ca := config.DefaultArgs()

	ca.StartupOptions.GitHubToken = env.RegisterStringVar("GITHUB_TOKEN", ca.StartupOptions.GitHubToken, githubToken).Get()
	ca.StartupOptions.ZenHubToken = env.RegisterStringVar("ZENHUB_TOKEN", ca.StartupOptions.ZenHubToken, zenhubToken).Get()
	ca.StartupOptions.GCPCredentials = env.RegisterStringVar("GCP_CREDS", ca.StartupOptions.GCPCredentials, gcpCreds).Get()
	ca.StartupOptions.ConfigRepo = env.RegisterStringVar("CONFIG_REPO", ca.StartupOptions.ConfigRepo, configRepo).Get()
	ca.StartupOptions.ConfigFile = env.RegisterStringVar("CONFIG_FILE", ca.StartupOptions.ConfigFile, configFile).Get()

	loggingOptions := log.DefaultOptions()
	var filters string

	syncerCmd := &cobra.Command{
		Use:   "sync",
		Short: "Manually run the GitHub syncer",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.Configure(loggingOptions); err != nil {
				log.Errorf("Unable to configure logging: %v", err)
			}

			// neutralize gRPC logging since it spews out useless junk
			var dummy = dummyIoWriter{}
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

			cmd.SilenceUsage = true
			return server.Sync(ca, filters)
		},
	}

	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigRepo, "configRepo", "", ca.StartupOptions.ConfigRepo, configRepo)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigFile, "configFile", "", ca.StartupOptions.ConfigFile, configFile)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubToken, "github_token", "", ca.StartupOptions.GitHubToken, githubToken)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ZenHubToken, "zenhub_token", "", ca.StartupOptions.ZenHubToken, zenhubToken)
	syncerCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GCPCredentials, "gcp_creds", "", ca.StartupOptions.GCPCredentials, gcpCreds)

	syncerCmd.PersistentFlags().StringVarP(&filters,
		"filter", "", "", "Comma-separated filters to limit what is synced, one or more of [issues, prs, labels, maintainers, members, zenhub]")

	loggingOptions.AttachCobraFlags(syncerCmd)

	return syncerCmd
}
