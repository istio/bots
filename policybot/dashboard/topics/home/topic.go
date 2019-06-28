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

//go:generate ../../../scripts/gen_topic.sh

package home

import (
	"net/http"
	"strings"
	"text/template"

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard"
)

type topic struct {
	topics []dashboard.RegisteredTopic
	home   *template.Template
}

func NewTopic(topics []dashboard.RegisteredTopic) dashboard.Topic {
	return &topic{
		topics: topics,
		home:   template.Must(template.New("home").Parse(string(MustAsset("page.html")))),
	}
}

func (t *topic) Title() string {
	return ""
}

func (t *topic) Description() string {
	return "Istio engineering dashboard"
}

func (t *topic) Name() string {
	return ""
}

func (t *topic) Configure(htmlRouter *mux.Router, apiRouter *mux.Router, context dashboard.RenderContext, opt *dashboard.Options) {
	htmlRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sb := &strings.Builder{}
			if err := t.home.Execute(sb, t.topics); err != nil {
				context.RenderHTMLError(w, err)
			} else {
				context.RenderHTML(w, sb.String())
			}
		})
}
