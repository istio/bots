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
	"net/http"
	"strconv"

	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
)

//check if website is down hourly, send email if detected error
func HourlyReport(reg *config.Registry, secrets *cmdutil.Secrets) error {
	message := ""
	sendMessage := false
	//check if website istio.io is down
	resp, err := http.Get("https://istio.io")
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		sendMessage = true
		message += "istio.io is down:" + strconv.Itoa(resp.StatusCode) + resp.Status
	}
	defer resp.Body.Close()
	//check if website preliminary.istio.io is down
	resp, err = http.Get("https://preliminary.istio.io/")
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		sendMessage = true
		message += "preliminary.istio.io is down:" + strconv.Itoa(resp.StatusCode) + resp.Status
	}
	defer resp.Body.Close()

	if sendMessage {
		err = Send("Website is down", "", message, reg, secrets)
		return err
	}
	return nil
}
