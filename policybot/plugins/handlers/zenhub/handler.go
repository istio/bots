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

package zenhub

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"istio.io/bots/policybot/pkg/zh"

	"istio.io/pkg/log"
)

var scope = log.RegisterScope("zenhub", "The ZenHub webhook handler", 0)

// Decodes and dispatches ZenHub webhook calls
type handler struct {
}

func NewHandler() http.Handler {
	return &handler{}
}

type typer struct {
	Type string `json:"type"`
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		scope.Errorf("Unable to read body from ZenHub event: %v", err)
		return
	}

	data := &typer{}
	if err = json.Unmarshal(body, data); err != nil {
		scope.Errorf("Unable to parse ZenHub event body: %v", err)
		return
	}

	switch data.Type {
	case "issue_transfer":
		scope.Infof("Received IssueTransferEvent from ZenHub")

		result := &zh.IssueTransferEvent{}
		if err = json.Unmarshal(body, result); err != nil {
			log.Errorf("Unable to decode ZenHub issue transfer event: %v", err)
			return
		}

		// TODO: do something with this event

	case "issue_reprioritized_event":
		scope.Infof("Received IssueReprioritizedEvent from ZenHub")

		result := &zh.IssueReprioritizedEvent{}
		if err = json.Unmarshal(body, result); err != nil {
			log.Errorf("Unable to decode ZenHub issue reprioritization event: %v", err)
			return
		}

		// TODO: do something with this event
	}
}
