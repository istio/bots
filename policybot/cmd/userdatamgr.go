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

	"github.com/spf13/cobra"

	"istio.io/bots/policybot/mgrs/userdatamgr"
	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/storage/spanner"
)

func userdataMgrCmd() *cobra.Command {
	cmd, _ := cmdutil.Run("userdatamgr", "Runs the user data manager, which loads user data into the bot's store", 0,
		cmdutil.ConfigPath|cmdutil.ConfigRepo, runUserdataMgr)

	return cmd
}

func runUserdataMgr(reg *config.Registry, secrets *cmdutil.Secrets) error {
	core := reg.Core()

	store, err := spanner.NewStore(context.Background(), core.SpannerDatabase)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	mgr := userdatamgr.New(store, reg)
	return mgr.Store(false)
}
