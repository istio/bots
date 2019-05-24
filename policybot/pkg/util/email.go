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

package util

import (
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Mailer struct {
	client *sendgrid.Client
	from   *mail.Email
}

func NewMailer(sendgridAPIKey string, fromName string, fromEmailAddress string) *Mailer {
	return &Mailer{
		from:   mail.NewEmail(fromName, fromEmailAddress),
		client: sendgrid.NewSendClient(sendgridAPIKey),
	}
}

func (m *Mailer) Send(toName string, toEmailAddress string, subject string, htmlContent string) error {
	to := mail.NewEmail(toName, toEmailAddress)
	message := mail.NewSingleEmail(m.from, subject, to, "plain text content", htmlContent)
	r, err := m.client.Send(message)
	if err == nil {
		if r.StatusCode >= 400 {
			return fmt.Errorf("unable to send email to %s: status code %d", toEmailAddress, r.StatusCode)
		}
	}

	return err
}
