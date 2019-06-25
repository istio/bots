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
	RenderHTML(w http.ResponseWriter, htmlFragment string)
	RenderHTMLError(w http.ResponseWriter, err error)
	RenderJSON(w http.ResponseWriter, statusCode int, data interface{})
}

type renderContext struct {
	topic            Topic
	primaryTemplates *template.Template
	errorTemplates   *template.Template
}

type templateInfo struct {
	Title       string
	Description string
	Content     string
}

var scope = log.RegisterScope("dashboard", "The UI dashboard.", 0)

func newRenderContext(topic Topic, primaryTemplates *template.Template, errorTemplates *template.Template) RenderContext {
	return renderContext{
		topic:            topic,
		primaryTemplates: primaryTemplates,
		errorTemplates:   errorTemplates,
	}
}

func (rc renderContext) RenderHTML(w http.ResponseWriter, htmlFragment string) {
	b := &bytes.Buffer{}

	info := templateInfo{
		Title:       rc.topic.Title(),
		Description: rc.topic.Description(),
		Content:     htmlFragment,
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
