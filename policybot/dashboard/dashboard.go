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

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard/templates/layout"
	"istio.io/bots/policybot/dashboard/templates/widgets"
	"istio.io/bots/policybot/pkg/util"
)

// Dashboard captures all the user-interface state necessary to expose the full UI to clients.
type Dashboard struct {
	primaryTemplates  *template.Template
	notFoundTemplates *template.Template
	errorTemplates    *template.Template
	topics            []*RegisteredTopic
	htmlRouter        *mux.Router
	apiRouter         *mux.Router
	options           Options
}

// RegisteredTopic represents a top-level UI topic that's been registered for use.
type RegisteredTopic struct {
	Title       string
	Description string
	URL         string
	Subtopics   []*RegisteredTopic
}

func New(router *mux.Router, clientID string, clientSecret string) *Dashboard {
	d := &Dashboard{
		primaryTemplates:  template.Must(template.New("base").Parse(layout.BaseTemplate)),
		notFoundTemplates: template.Must(template.New("base").Parse(layout.BaseTemplate)),
		errorTemplates:    template.Must(template.New("base").Parse(layout.BaseTemplate)),
		htmlRouter:        router,
		apiRouter:         router.PathPrefix("/api").Subrouter(),
		options:           Options{"istio"}, // TODO: get rid of Istio default
	}

	// primary templates
	d.primaryTemplates.Funcs(template.FuncMap{
		"getTopics": d.Topics,
		"dict":      dict,
		"printf":    printf,
	})
	_ = template.Must(d.primaryTemplates.Parse(layout.PrimaryTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.HeaderTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.SidebarTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.SidebarLevelTemplate))

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
	d.topics = append(d.topics, d.registerTopic(t, d.htmlRouter, d.apiRouter, ""))
}

func (d *Dashboard) registerTopic(t Topic, htmlRouter *mux.Router, apiRouter *mux.Router, basePath string) *RegisteredTopic {
	htmlRouter = htmlRouter.PathPrefix("/" + t.Name()).Subrouter()
	apiRouter = apiRouter.PathPrefix("/" + t.Name()).Subrouter()

	rt := &RegisteredTopic{
		Title:       t.Title(),
		Description: t.Description(),
		URL:         basePath + "/" + t.Name(),
		Subtopics:   d.handleSubtopics(t.Subtopics(), htmlRouter, apiRouter, basePath+"/"+t.Name()),
	}

	t.Configure(htmlRouter, apiRouter, newRenderContext(rt, d.primaryTemplates, d.errorTemplates), &Options{"istio"}) // TODO: eliminate istio default

	return rt
}

func (d *Dashboard) handleSubtopics(subtopics []Topic, htmlRouter *mux.Router, apiRouter *mux.Router, basePath string) []*RegisteredTopic {
	var rt []*RegisteredTopic
	for _, s := range subtopics {
		rt = append(rt, d.registerTopic(s, htmlRouter, apiRouter, basePath))
	}
	return rt
}

func (d *Dashboard) RegisterPageNotFound() {
	d.htmlRouter.StrictSlash(true).
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

func (d *Dashboard) Topics() []*RegisteredTopic {
	return d.topics
}

func (d *Dashboard) registerStaticDir(fsPath string, sitePath string) {
	d.htmlRouter.
		PathPrefix(sitePath).
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(sitePath, http.FileServer(http.Dir(fsPath))).ServeHTTP(w, r)
		})
}

func (d *Dashboard) registerStaticFile(fsPath string, sitePath string) {
	d.htmlRouter.
		Path(sitePath).
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.RequestURI, ".json") {
				w.Header().Set("Content-Type", "application/json")
			}
			http.ServeFile(w, r, fsPath)
		})
}

func (rt *RegisteredTopic) IsAncestor(ti templateInfo) bool {
	return strings.HasPrefix(ti.URL, rt.URL)
}
