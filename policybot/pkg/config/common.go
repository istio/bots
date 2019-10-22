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

package config

import (
	"fmt"

	"istio.io/pkg/ctrlz"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/grpclog"

	"istio.io/pkg/log"

	"istio.io/pkg/env"
)

const (
	configRepo              = "GitHub org/repo/branch where to fetch policybot config"
	configFile              = "Path to a configuration file"
	githubWebhookSecret     = "Secret for the GitHub webhook"
	githubToken             = "Token to access the GitHub API"
	gcpCreds                = "Base64-encoded credentials to access GCP"
	sendgridAPIKey          = "API Key for sendgrid.com"
	zenhubToken             = "Token to access the ZenHub API"
	githubOAuthClientSecret = "Client secret for GitHub OAuth2 flow"
	githubOAuthClientID     = "Client ID for GitHub OAuth2 flow"
)

type CommonFlags int

const (
	ConfigRepo              CommonFlags = 1 << 0
	ConfigFile                          = 1 << 1
	GitHubWebhookSecret                 = 1 << 2
	GitHubToken                         = 1 << 3
	GCPCreds                            = 1 << 4
	SendgridAPIKey                      = 1 << 5
	ZenhubToken                         = 1 << 6
	GithubOAuthClientSecret             = 1 << 7
	GithubOAuthClientID                 = 1 << 8
	ControlZ                            = 1 << 9
)

func Run(name string, desc string, numArgs int, flags CommonFlags, cb func(ca *Args, args []string) error) (*cobra.Command, *Args) {
	ca := DefaultArgs()
	cmd := &cobra.Command{}

	if flags&ConfigRepo != 0 {
		ca.ConfigRepo = env.RegisterStringVar("CONFIG_REPO", ca.ConfigRepo, configRepo).Get()
		cmd.PersistentFlags().StringVarP(&ca.ConfigRepo,
			"config_repo", "", ca.ConfigRepo, configRepo)
	}

	if flags&ConfigFile != 0 {
		ca.ConfigFile = env.RegisterStringVar("CONFIG_FILE", ca.ConfigFile, configFile).Get()
		cmd.PersistentFlags().StringVarP(&ca.ConfigFile,
			"config_file", "", ca.ConfigFile, configFile)
	}

	if flags&GitHubWebhookSecret != 0 {
		ca.Secrets.GitHubWebhookSecret = env.RegisterStringVar("GITHUB_WEBHOOK_SECRET", ca.Secrets.GitHubWebhookSecret, githubWebhookSecret).Get()
		cmd.PersistentFlags().StringVarP(&ca.Secrets.GitHubWebhookSecret,
			"github_webhook_secret", "", ca.Secrets.GitHubWebhookSecret, githubWebhookSecret)
	}

	if flags&GitHubToken != 0 {
		ca.Secrets.GitHubToken = env.RegisterStringVar("GITHUB_TOKEN", ca.Secrets.GitHubToken, githubToken).Get()
		cmd.PersistentFlags().StringVarP(&ca.Secrets.GitHubToken,
			"github_token", "", ca.Secrets.GitHubToken, githubToken)
	}

	if flags&GCPCreds != 0 {
		ca.Secrets.GCPCredentials = env.RegisterStringVar("GCP_CREDS", ca.Secrets.GCPCredentials, gcpCreds).Get()
		cmd.PersistentFlags().StringVarP(&ca.Secrets.GCPCredentials,
			"gcp_creds", "", ca.Secrets.GCPCredentials, gcpCreds)
	}

	if flags&SendgridAPIKey != 0 {
		ca.Secrets.SendGridAPIKey = env.RegisterStringVar("SENDGRID_APIKEY", ca.Secrets.SendGridAPIKey, sendgridAPIKey).Get()
		cmd.PersistentFlags().StringVarP(&ca.Secrets.SendGridAPIKey,
			"sendgrid_apikey", "", ca.Secrets.SendGridAPIKey, sendgridAPIKey)
	}

	if flags&ZenhubToken != 0 {
		ca.Secrets.ZenHubToken = env.RegisterStringVar("ZENHUB_TOKEN", ca.Secrets.ZenHubToken, zenhubToken).Get()
		cmd.PersistentFlags().StringVarP(&ca.Secrets.ZenHubToken,
			"zenhub_token", "", ca.Secrets.ZenHubToken, zenhubToken)
	}

	if flags&GithubOAuthClientSecret != 0 {
		ca.Secrets.GitHubOAuthClientSecret =
			env.RegisterStringVar("GITHUB_OAUTH_CLIENT_SECRET", ca.Secrets.GitHubOAuthClientSecret, githubOAuthClientSecret).Get()
		cmd.PersistentFlags().StringVarP(&ca.Secrets.GitHubOAuthClientSecret,
			"github_oauth_client_secret", "", ca.Secrets.GitHubOAuthClientSecret, githubOAuthClientSecret)
	}

	if flags&GithubOAuthClientID != 0 {
		ca.Secrets.GitHubOAuthClientID =
			env.RegisterStringVar("GITHUB_OAUTH_CLIENT_ID", ca.Secrets.GitHubOAuthClientID, githubOAuthClientID).Get()

		cmd.PersistentFlags().StringVarP(&ca.Secrets.GitHubOAuthClientID,
			"github_oauth_client_id", "", ca.Secrets.GitHubOAuthClientID, githubOAuthClientID)
	}

	loggingOptions := log.DefaultOptions()
	introspectionOptions := ctrlz.DefaultOptions()

	cmd.Use = name
	cmd.Short = desc
	cmd.Args = cobra.ExactArgs(numArgs)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := log.Configure(loggingOptions); err != nil {
			log.Errorf("Unable to configure logging: %v", err)
		}

		if flags&ControlZ != 0 {
			if cs, err := ctrlz.Run(introspectionOptions, nil); err == nil {
				defer cs.Close()
			} else {
				log.Errorf("Unable to initialize ControlZ: %v", err)
			}
		}

		// neutralize gRPC logging since it spews out useless junk
		var dummy = dummyIoWriter{}
		grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

		cmd.SilenceUsage = true

		// load the config file
		if err := ca.Fetch(); err != nil {
			return fmt.Errorf("unable to load configuration file: %v", err)
		}

		return cb(ca, args)
	}

	loggingOptions.AttachCobraFlags(cmd)

	if flags&ControlZ != 0 {
		introspectionOptions.AttachCobraFlags(cmd)
	}

	return cmd, ca
}

type dummyIoWriter struct{}

func (dummyIoWriter) Write([]byte) (int, error) { return 0, nil }
