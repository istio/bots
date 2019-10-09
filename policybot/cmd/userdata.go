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

	"istio.io/bots/policybot/pkg/storage/spanner"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/userdata"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

func userdataCmd() *cobra.Command {
	ca := config.DefaultArgs()

	ca.StartupOptions.GCPCredentials = env.RegisterStringVar("GCP_CREDS", ca.StartupOptions.GCPCredentials, gcpCreds).Get()
	ca.StartupOptions.ConfigFile = env.RegisterStringVar("CONFIG_FILE", ca.StartupOptions.ConfigFile, configFile).Get()
	ca.StartupOptions.ConfigRepo = env.RegisterStringVar("CONFIG_REPO", ca.StartupOptions.ConfigRepo, configRepo).Get()

	loggingOptions := log.DefaultOptions()

	userdataCmd := &cobra.Command{
		Use:   "userdata <file>",
		Short: "Loads user data into the bot's store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.Configure(loggingOptions); err != nil {
				log.Errorf("Unable to configure logging: %v", err)
			}

			// neutralize gRPC logging since it spews out useless junk
			var dummy = dummyIoWriter{}
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

			cmd.SilenceUsage = true
			return runUserData(ca, args[0])
		},
	}

	userdataCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigFile,
		"config_file", "", ca.StartupOptions.ConfigFile, configFile)
	userdataCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigRepo, "config_repo", "", ca.StartupOptions.ConfigRepo, configRepo)

	loggingOptions.AttachCobraFlags(userdataCmd)

	return userdataCmd
}

func runUserData(a *config.Args, file string) error {
	ud, err := userdata.Load(file)
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

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	return ud.Store(store)
}
