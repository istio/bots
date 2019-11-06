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

	"istio.io/bots/policybot/mgrs/lifecyclemgr"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/spanner"
)

func lifecycleMgrCmd() *cobra.Command {
	cmd, _ := config.Run("lifecyclemgr", "Runs the issue and pull request lifecycle manager", 0,
		config.ConfigFile|config.ConfigRepo|config.GitHubToken|config.GCPCreds, runLifecycleMgr)

	return cmd
}

func runLifecycleMgr(a *config.Args, _ []string) error {
	creds, err := base64.StdEncoding.DecodeString(a.Secrets.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	gc := gh.NewThrottledClient(context.Background(), a.Secrets.GitHubToken)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	mgr := lifecyclemgr.New(gc, store, a)
	return mgr.ManageAll(context.Background())
}
