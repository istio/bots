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

	"istio.io/bots/policybot/notification"
	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
)

func notificationCmd() *cobra.Command {
	cmd, _ := cmdutil.Run("notification", "send notification when detect website error", 0,
		cmdutil.ConfigPath|cmdutil.ConfigRepo|cmdutil.SendgridAPIKey, runNotification)

	return cmd
}

// Runs the flake manager.
func runNotification(reg *config.Registry, secrets *cmdutil.Secrets) error {
	//notification.Test(reg, secrets)
	notification.HourlyReport(reg, secrets)
	return nil
	//notification.SendHelloEmail()
	//return notification.Send("docs group","hello", reg, secrets)
}
