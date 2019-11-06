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

	"istio.io/bots/policybot/mgrs/userdatamgr"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/storage/spanner"
)

func userdataMgrCmd() *cobra.Command {
	cmd, _ := config.Run("userdatamgr <file>", "Runs the user data manager, which loads user data into the bot's store", 1,
		config.ConfigFile|config.ConfigRepo|config.GCPCreds, runUserdataMgr)

	return cmd
}

func runUserdataMgr(a *config.Args, args []string) error {
	file := args[0]

	creds, err := base64.StdEncoding.DecodeString(a.Secrets.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	ud, err := userdatamgr.Load(file)
	if err != nil {
		return err
	}

	return ud.Store(store)
}
