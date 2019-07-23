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
	"net/http"
	"text/template"

	"istio.io/bots/policybot/pkg/util"
)

type notFound struct {
	templates *template.Template
}

func (nf notFound) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	info := templateInfo{
		Title:       "Page Not Found",
		Description: "Page Not Found",
	}

	var b bytes.Buffer
	if err := nf.templates.Execute(&b, info); err != nil {
		util.RenderError(w, util.HTTPErrorf(http.StatusNotFound, "Page Not Found"))
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = b.WriteTo(w)
}
