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

package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

// RenderContext exposes methods to let topics produce output
type RenderContext interface {
	// Render an HTML page.
	//
	// The override title is optional and can be used to replace the canonical title (which is supplied by
	// the topic)
	//
	// The content fragment represents the main chunk of topic-HTML to insert into the main page template.
	//
	// The optional control fragment represents a chunk of HTML to insert into the "control section" of the
	// main page template and is where page-level commands & controls can be inserted.
	RenderHTML(w http.ResponseWriter, overrideTitle string, contentFragment string, controlFragment string)

	// Generate an HTML error page, displaying the given error
	RenderHTMLError(w http.ResponseWriter, err error)

	// Generate a chunk of JSON.
	RenderJSON(w http.ResponseWriter, statusCode int, data interface{})
}

type renderContext struct {
	topic            *RegisteredTopic
	primaryTemplates *template.Template
	errorTemplates   *template.Template
}

type templateInfo struct {
	Title       string
	Description string
	URL         string
	Content     string
	Control     string
}

var scope = log.RegisterScope("dashboard", "The UI dashboard.", 0)

func newRenderContext(topic *RegisteredTopic, primaryTemplates *template.Template, errorTemplates *template.Template) RenderContext {
	return renderContext{
		topic:            topic,
		primaryTemplates: primaryTemplates,
		errorTemplates:   errorTemplates,
	}
}

func (rc renderContext) RenderHTML(w http.ResponseWriter, overrideTitle string, contentFragment string, controlFragment string) {
	b := &bytes.Buffer{}

	title := overrideTitle
	if overrideTitle == "" {
		title = rc.topic.Title
	}

	info := templateInfo{
		Title:       title,
		Description: rc.topic.Description,
		Content:     contentFragment,
		Control:     controlFragment,
		URL:         rc.topic.URL,
	}

	if err := rc.primaryTemplates.Execute(b, info); err != nil {
		rc.RenderHTMLError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = b.WriteTo(w)
}

// RenderHTMLError outputs an error message
func (rc renderContext) RenderHTMLError(w http.ResponseWriter, err error) {
	b := &bytes.Buffer{}

	info := templateInfo{
		Title:       "ERROR",
		Description: "ERROR",
		Content:     fmt.Sprintf("%v", err),
	}

	if err2 := rc.errorTemplates.Execute(b, info); err2 != nil {
		util.RenderError(w, err)
		return
	}

	statusCode := http.StatusInternalServerError
	if httpErr, ok := err.(util.HTTPError); ok {
		statusCode = httpErr.StatusCode
	}

	w.WriteHeader(statusCode)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = b.WriteTo(w)

	scope.Errorf("Returning error to client: %v", info.Content)
}

func (rc renderContext) RenderJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		util.RenderError(w, util.HTTPErrorf(http.StatusInternalServerError, "%v", err))
	}
}
