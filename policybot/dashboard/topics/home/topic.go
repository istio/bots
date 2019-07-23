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

	"istio.io/bots/policybot/dashboard/types"
)

// Home provides the landing page for the UI dashboard
type Home struct {
	page    *template.Template
	entries []Entry
}

// A single entry to display on the dashboard landing page
type Entry struct {
	Title       string
	Description string
	URL         string
}

// New creates a new Home instance.
func New(entries []Entry) *Home {
	return &Home{
		page:    template.Must(template.New("home").Parse(string(MustAsset("page.html")))),
		entries: entries,
	}
}

// Renders the HTML for this topic.
func (h *Home) Render(req *http.Request) (types.RenderInfo, error) {
	var sb strings.Builder
	if err := h.page.Execute(&sb, h.entries); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}
