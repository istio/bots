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
package notification

import (
	"fmt"

	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/util"
)

//send notification to docs member. Email includes owner and event information in the title
func Send(title string, owner string, message string, reg *config.Registry, secrets *cmdutil.Secrets) error {
	toName := "istio member"
	toEmailAddress := "csm-docs@google.com"
	subject := owner + title

	core := reg.Core()
	mail := *util.NewMailer(secrets.SendGridAPIKey, core.EmailFrom, core.EmailOriginAddress)
	err := mail.Send(toName, toEmailAddress, subject, message)
	if err != nil {
		fmt.Printf("can't send email: %v", err)
		return err
	}
	return nil
}
