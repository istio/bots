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
	"strings"
	"text/template"

	"istio.io/bots/policybot/pkg/util"

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard/templates/layout"
	"istio.io/bots/policybot/dashboard/templates/widgets"
)

// Dashboard captures all the user-interface state necessary to expose the full UI to clients.
type Dashboard struct {
	primaryTemplates  *template.Template
	notFoundTemplates *template.Template
	errorTemplates    *template.Template
	topics            []RegisteredTopic
	router            *mux.Router
}

// RegisteredTopic represents a top-level UI topic that's been registered for use.
type RegisteredTopic struct {
	Title       string
	Description string
	URL         string
}

func New(router *mux.Router, clientID string, clientSecret string) *Dashboard {
	d := &Dashboard{
		primaryTemplates:  template.Must(template.New("base").Parse(layout.BaseTemplate)),
		notFoundTemplates: template.Must(template.New("base").Parse(layout.BaseTemplate)),
		errorTemplates:    template.Must(template.New("base").Parse(layout.BaseTemplate)),
		router:            router,
	}

	// primary templates
	d.primaryTemplates.Funcs(template.FuncMap{
		"getTopics": d.Topics,
	})
	_ = template.Must(d.primaryTemplates.Parse(layout.PrimaryTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.HeaderTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.SidebarTemplate))

	// 'not found' template
	_ = template.Must(d.notFoundTemplates.Parse(layout.NotFoundTemplate))
	_ = template.Must(d.notFoundTemplates.Parse(widgets.HeaderTemplate))

	// error template
	_ = template.Must(d.errorTemplates.Parse(layout.ErrorTemplate))
	_ = template.Must(d.errorTemplates.Parse(widgets.HeaderTemplate))

	// statically served directories
	d.registerStaticDir("generated/css", "/css/")
	d.registerStaticDir("generated/icons", "/icons/")
	d.registerStaticDir("generated/js", "/js/")
	d.registerStaticDir("dashboard/static/img", "/img/")
	d.registerStaticDir("dashboard/static/favicons", "/favicons/")

	// statically served files
	d.registerStaticFile("dashboard/static/favicon.ico", "/favicon.ico")
	d.registerStaticFile("dashboard/static/browserconfig.xml", "/browserconfig.xml")
	d.registerStaticFile("dashboard/static/manifest.json", "/manifest.json")

	// oauth support
	oauthLogin, oauthCallback := newOAuthHandlers(clientID, clientSecret, newRenderContext(nil, d.primaryTemplates, d.errorTemplates))
	router.Handle("/login", oauthLogin)
	router.Handle("/githuboauthcallback", oauthCallback)

	return d
}

func (d *Dashboard) RegisterTopic(t Topic) {
	htmlRouter := d.router.PathPrefix("/" + t.Name()).Subrouter()
	apiRouter := d.router.PathPrefix("/" + t.Name() + "api").Subrouter()

	t.Configure(htmlRouter, apiRouter, newRenderContext(t, d.primaryTemplates, d.errorTemplates), &Options{"istio"}) // TODO: eliminate istio default

	d.topics = append(d.topics,
		RegisteredTopic{
			Title:       t.Title(),
			Description: t.Description(),
			URL:         "/" + t.Name(),
		})
}

func (d *Dashboard) RegisterPageNotFound() {
	d.router.StrictSlash(true).
		PathPrefix("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			b := &bytes.Buffer{}

			info := templateInfo{
				Title:       "Page Not Found",
				Description: "Page Not Found",
			}

			if err := d.notFoundTemplates.Execute(b, info); err != nil {
				util.RenderError(w, util.HTTPErrorf(http.StatusInternalServerError, "%v", err))
				return
			}

			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = b.WriteTo(w)
		})
}

func (d *Dashboard) Topics() []RegisteredTopic {
	return d.topics
}

func (d *Dashboard) registerStaticDir(fsPath string, sitePath string) {
	d.router.
		PathPrefix(sitePath).
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(sitePath, http.FileServer(http.Dir(fsPath))).ServeHTTP(w, r)
		})
}

func (d *Dashboard) registerStaticFile(fsPath string, sitePath string) {
	d.router.
		Path(sitePath).
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.RequestURI, ".json") {
				w.Header().Set("Content-Type", "application/json")
			}
			http.ServeFile(w, r, fsPath)
		})
}
