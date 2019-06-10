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
	"istio.io/pkg/ctrlz"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

const (
	githubSecret   = "Secret for the GitHub webhook"
	githubToken    = "Token to access the GitHub API"
	gcpCreds       = "Base64-encoded credentials to access GCP"
	configRepo     = "GitHub org/repo/branch where to fetch policybot config"
	configFile     = "Path to a configuration file"
	sendgridAPIKey = "API Key for sendgrid.com"
	zenhubToken    = "Token to access the ZenHub API"
	port           = "TCP port to listen to for incoming traffic"
)

func serverCmd() *cobra.Command {
	ca := config.DefaultArgs()

	ca.StartupOptions.GitHubSecret = env.RegisterStringVar("GITHUB_SECRET", ca.StartupOptions.GitHubSecret, githubSecret).Get()
	ca.StartupOptions.GitHubToken = env.RegisterStringVar("GITHUB_TOKEN", ca.StartupOptions.GitHubToken, githubToken).Get()
	ca.StartupOptions.ZenHubToken = env.RegisterStringVar("ZENHUB_TOKEN", ca.StartupOptions.ZenHubToken, zenhubToken).Get()
	ca.StartupOptions.GCPCredentials = env.RegisterStringVar("GCP_CREDS", ca.StartupOptions.GCPCredentials, gcpCreds).Get()
	ca.StartupOptions.ConfigRepo = env.RegisterStringVar("CONFIG_REPO", ca.StartupOptions.ConfigRepo, configRepo).Get()
	ca.StartupOptions.ConfigFile = env.RegisterStringVar("CONFIG_FILE", ca.StartupOptions.ConfigFile, configFile).Get()
	ca.StartupOptions.SendGridAPIKey = env.RegisterStringVar("SENDGRID_APIKEY", ca.StartupOptions.SendGridAPIKey, sendgridAPIKey).Get()
	ca.StartupOptions.Port = env.RegisterIntVar("PORT", ca.StartupOptions.Port, port).Get()

	loggingOptions := log.DefaultOptions()
	introspectionOptions := ctrlz.DefaultOptions()

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Starts the policybot server",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.Configure(loggingOptions); err != nil {
				log.Errorf("Unable to configure logging: %v", err)
			}

			// neutralize gRPC logging since it spews out useless junk
			var dummy = dummyIoWriter{}
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

			if cs, err := ctrlz.Run(introspectionOptions, nil); err == nil {
				defer cs.Close()
			} else {
				log.Errorf("Unable to initialize ControlZ: %v", err)
			}

			return server.RunServer(ca)
		},
	}

	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigRepo, "configRepo", "", ca.StartupOptions.ConfigRepo, configRepo)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigFile, "configFile", "", ca.StartupOptions.ConfigFile, configFile)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubSecret, "github_secret", "", ca.StartupOptions.GitHubSecret, githubSecret)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubToken, "github_token", "", ca.StartupOptions.GitHubToken, githubToken)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GCPCredentials, "gcp_creds", "", ca.StartupOptions.GCPCredentials, gcpCreds)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.SendGridAPIKey, "sendgrid_apikey", "", ca.StartupOptions.SendGridAPIKey, sendgridAPIKey)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ZenHubToken, "zenhub_token", "", ca.StartupOptions.ZenHubToken, zenhubToken)
	serverCmd.PersistentFlags().IntVarP(&ca.StartupOptions.Port, "port", "", ca.StartupOptions.Port, port)

	loggingOptions.AttachCobraFlags(serverCmd)
	introspectionOptions.AttachCobraFlags(serverCmd)

	return serverCmd
}

type dummyIoWriter struct{}

func (dummyIoWriter) Write([]byte) (int, error) { return 0, nil }
