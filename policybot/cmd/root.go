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
	"flag"

	"github.com/spf13/cobra"

	"istio.io/pkg/version"
)

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "policybot",
		Short: "A GitHub bot that implement Istio policies around issues, pull requests, and test flakes.",
		Args:  cobra.ExactArgs(0),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(syncerCmd())
	rootCmd.AddCommand(flakeChaserCmd())
	rootCmd.AddCommand(version.CobraCommand())

	return rootCmd
}
