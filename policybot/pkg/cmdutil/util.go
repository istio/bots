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

package cmdutil

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/grpclog"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/pkg/ctrlz"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

const (
	configRepo              = "GitHub org/repo/branch where to fetch policybot config"
	configPath              = "Path to a directory of configuration files"
	githubWebhookSecret     = "Secret for the GitHub webhook"
	githubToken             = "Token to access the GitHub API"
	gcpCreds                = "Base64-encoded credentials to access GCP"
	githubOAuthClientSecret = "Client secret for GitHub OAuth2 flow"
	githubOAuthClientID     = "Client ID for GitHub OAuth2 flow"
)

type CommonFlags int

const (
	ConfigRepo              CommonFlags = 1 << 0
	ConfigPath                          = 1 << 1
	GitHubWebhookSecret                 = 1 << 2
	GitHubToken                         = 1 << 3
	GithubOAuthClientSecret             = 1 << 7
	GithubOAuthClientID                 = 1 << 8
	ControlZ                            = 1 << 9
)

func Run(name string, desc string, numArgs int, flags CommonFlags, cb func(reg *config.Registry, secrets *Secrets) error) (*cobra.Command, *config.Registry) {
	secrets := Secrets{}
	cmd := &cobra.Command{}
	cfgPath := ""
	cfgRepo := ""

	if flags&ConfigRepo != 0 {
		cfgRepo = env.RegisterStringVar("CONFIG_REPO", cfgRepo, configRepo).Get()
		cmd.PersistentFlags().StringVarP(&cfgRepo, "config_repo", "", cfgRepo, configRepo)
	}

	if flags&ConfigPath != 0 {
		cfgPath = env.RegisterStringVar("CONFIG_PATH", cfgPath, configPath).Get()
		cmd.PersistentFlags().StringVarP(&cfgPath, "config_path", "", cfgPath, configPath)
	}

	if flags&GitHubWebhookSecret != 0 {
		secrets.GitHubWebhookSecret = env.RegisterStringVar("GITHUB_WEBHOOK_SECRET", secrets.GitHubWebhookSecret, githubWebhookSecret).Get()
		cmd.PersistentFlags().StringVarP(&secrets.GitHubWebhookSecret,
			"github_webhook_secret", "", secrets.GitHubWebhookSecret, githubWebhookSecret)
	}

	if flags&GitHubToken != 0 {
		secrets.GitHubToken = env.RegisterStringVar("GITHUB_TOKEN", secrets.GitHubToken, githubToken).Get()
		cmd.PersistentFlags().StringVarP(&secrets.GitHubToken,
			"github_token", "", secrets.GitHubToken, githubToken)
	}

	if flags&GithubOAuthClientSecret != 0 {
		secrets.GitHubOAuthClientSecret = env.RegisterStringVar("GITHUB_OAUTH_CLIENT_SECRET", secrets.GitHubOAuthClientSecret, githubOAuthClientSecret).Get()
		cmd.PersistentFlags().StringVarP(&secrets.GitHubOAuthClientSecret,
			"github_oauth_client_secret", "", secrets.GitHubOAuthClientSecret, githubOAuthClientSecret)
	}

	if flags&GithubOAuthClientID != 0 {
		secrets.GitHubOAuthClientID = env.RegisterStringVar("GITHUB_OAUTH_CLIENT_ID", secrets.GitHubOAuthClientID, githubOAuthClientID).Get()

		cmd.PersistentFlags().StringVarP(&secrets.GitHubOAuthClientID,
			"github_oauth_client_id", "", secrets.GitHubOAuthClientID, githubOAuthClientID)
	}

	loggingOptions := log.DefaultOptions()
	introspectionOptions := ctrlz.DefaultOptions()

	var reg *config.Registry

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
		dummy := dummyIoWriter{}
		grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

		cmd.SilenceUsage = true

		var err error
		if cfgRepo == "" {
			reg, err = config.LoadRegistryFromDirectory(cfgPath)
		} else {
			gc := gh.NewThrottledClient(context.Background(), secrets.GitHubToken)
			reg, err = config.LoadRegistryFromRepo(gc, gh.NewRepoDesc(cfgRepo), cfgPath)
		}

		if err != nil {
			return fmt.Errorf("unable to load configuration: %v", err)
		}

		return cb(reg, &secrets)
	}

	loggingOptions.AttachCobraFlags(cmd)

	if flags&ControlZ != 0 {
		introspectionOptions.AttachCobraFlags(cmd)
	}

	return cmd, reg
}

type dummyIoWriter struct{}

func (dummyIoWriter) Write([]byte) (int, error) { return 0, nil }
