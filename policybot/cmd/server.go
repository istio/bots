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
	"github.com/spf13/viper"

	"istio.io/bots/policybot/pkg/server"
	"istio.io/common/pkg/log"
)

func serverCmd() *cobra.Command {
	ca := server.DefaultArgs()

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Starts IstioPolicyBot as a server",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.AutomaticEnv()

			ca.Port = viper.GetInt("port")
			ca.GitHubSecret = viper.GetString("github_secret")
			ca.GitHubAccessToken = viper.GetString("github_token")
			ca.GCPCredentials = viper.GetString("gcp_creds")
			ca.SpannerDatabase = viper.GetString("spanner_db")

			if err := viper.UnmarshalKey("orgs", &ca.Orgs); err != nil {
				return err
			}

			log.Infof("IstioPolicyBot started with:\n%s", ca)
			return server.Run(ca)
		},
	}

	var cfgFile string
	serverCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Path to a configuration file.")

	cobra.OnInitialize(func() {
		if len(cfgFile) > 0 {
			viper.SetConfigFile(cfgFile)
			if err := viper.ReadInConfig(); err != nil {
				log.Fatalf("Unable to read configuration file %s: %v", cfgFile, err)
			}
		}
	})

	serverCmd.PersistentFlags().IntP("port", "", ca.Port,
		"The IP port to listen to")

	serverCmd.PersistentFlags().StringP("github_secret", "", ca.GitHubSecret,
		"The GitHub secret used with the webhook")

	serverCmd.PersistentFlags().StringP("github_token", "", ca.GitHubAccessToken,
		"The GitHub access token used to call the GitHub API")

	serverCmd.PersistentFlags().StringP("gcp_creds", "", ca.GCPCredentials,
		"The GCP credentials JSON, enabling the bot to use GCP services")

	serverCmd.PersistentFlags().StringP("spanner_db", "", ca.SpannerDatabase,
		"Name of the Spanner database having been previously configured for this bot.")

	ca.LoggingOptions.AttachCobraFlags(serverCmd)
	ca.IntrospectionOptions.AttachCobraFlags(serverCmd)

	_ = viper.BindPFlags(serverCmd.PersistentFlags())

	return serverCmd
}
