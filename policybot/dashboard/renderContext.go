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
)

// RenderContext exposes methods to let topics produce output
type RenderContext interface {
	RenderHTML(w http.ResponseWriter, htmlFragment string)
	RenderJSON(w http.ResponseWriter, statusCode int, data interface{})
}

type renderContext struct {
	topic        Topic
	baseTemplate *template.Template
}

type templateInfo struct {
	Title       string
	Description string
	Content     string
}

func newRenderContext(topic Topic, baseTemplate *template.Template) RenderContext {
	return renderContext{
		topic:        topic,
		baseTemplate: baseTemplate,
	}
}

func (rc renderContext) RenderHTML(w http.ResponseWriter, htmlFragment string) {
	b := &bytes.Buffer{}

	info := templateInfo{
		Title:       rc.topic.Title(),
		Description: rc.topic.Description(),
		Content:     htmlFragment,
	}

	if err := rc.baseTemplate.Execute(b, info); err != nil {
		RenderError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = b.WriteTo(w)
}

func (rc renderContext) RenderJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		RenderError(w, http.StatusInternalServerError, err)
	}
}

// RenderError outputs an error message
func RenderError(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = fmt.Fprintf(w, "%v", err)
}
