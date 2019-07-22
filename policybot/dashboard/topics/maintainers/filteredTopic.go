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

package maintainers

import (
	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard"
)

type filteredTopic struct {
	title       string
	description string
	suffix      string
}

var _ dashboard.Topic = filteredTopic{}

func (ft filteredTopic) Title() string {
	return ft.title
}

func (ft filteredTopic) Description() string {
	return ft.description
}

func (ft filteredTopic) URLSuffix() string {
	return ft.suffix
}

func (ft filteredTopic) Subtopics() []dashboard.Topic {
	return nil
}

func (ft filteredTopic) Configure(htmlRouter *mux.Router, apiRouter *mux.Router, context dashboard.RenderContext, opt *dashboard.Options) {
}
