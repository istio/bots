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
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"istio.io/bots/policybot/dashboard/topics/workinggroups"

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard/templates/layout"
	"istio.io/bots/policybot/dashboard/templates/widgets"
	"istio.io/bots/policybot/dashboard/topics/commithub"
	"istio.io/bots/policybot/dashboard/topics/coverage"
	"istio.io/bots/policybot/dashboard/topics/features"
	"istio.io/bots/policybot/dashboard/topics/flakes"
	"istio.io/bots/policybot/dashboard/topics/home"
	"istio.io/bots/policybot/dashboard/topics/issues"
	"istio.io/bots/policybot/dashboard/topics/maintainers"
	"istio.io/bots/policybot/dashboard/topics/members"
	"istio.io/bots/policybot/dashboard/topics/perf"
	"istio.io/bots/policybot/dashboard/topics/pullrequests"
	"istio.io/bots/policybot/dashboard/types"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

// Dashboard captures all the user-interface state necessary to expose the full UI to clients.
type Dashboard struct {
	primaryTemplates *template.Template
	errorTemplates   *template.Template
	entries          []*sidebarEntry
	router           *mux.Router
	options          Options
	currentEntry     *sidebarEntry
	oauthHandler     *oauthHandler
	entryMap         map[*mux.Route]*sidebarEntry
}

type templateInfo struct {
	Title         string
	Description   string
	Content       string
	Control       string
	Entries       []*sidebarEntry
	SelectedEntry *sidebarEntry
}

var scope = log.RegisterScope("dashboard", "The UI layer", 0)

func New(router *mux.Router, store storage.Store, cache *cache.Cache, a *config.Args) *Dashboard {
	d := &Dashboard{
		primaryTemplates: template.Must(template.New("base").Parse(layout.BaseTemplate)),
		errorTemplates:   template.Must(template.New("base").Parse(layout.BaseTemplate)),
		router:           router,
		options:          Options{"istio"}, // TODO: get rid of Istio default
		oauthHandler:     newOAuthHandler(a.StartupOptions.GitHubOAuthClientID, a.StartupOptions.GitHubOAuthClientSecret),
		entryMap:         make(map[*mux.Route]*sidebarEntry),
	}

	// primary templates
	d.primaryTemplates.Funcs(template.FuncMap{
		"dict":   dict,
		"printf": printf,
	})
	_ = template.Must(d.primaryTemplates.Parse(layout.PrimaryTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.HeaderTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.SidebarTemplate))
	_ = template.Must(d.primaryTemplates.Parse(widgets.SidebarLevelTemplate))

	// 'page not found' template
	nf := notFound{
		template.Must(template.New("base").Parse(layout.BaseTemplate)),
	}
	_ = template.Must(nf.templates.Parse(layout.NotFoundTemplate))
	_ = template.Must(nf.templates.Parse(widgets.HeaderTemplate))

	// error template
	_ = template.Must(d.errorTemplates.Parse(layout.ErrorTemplate))
	_ = template.Must(d.errorTemplates.Parse(widgets.HeaderTemplate))

	// statically served files
	d.registerStaticFile("dashboard/static/favicon.ico", "/favicon.ico")
	d.registerStaticFile("dashboard/static/browserconfig.xml", "/browserconfig.xml")
	d.registerStaticFile("dashboard/static/manifest.json", "/manifest.json")
	d.registerStaticFile("dashboard/static/js/fitty.min.js", "/js/fitty.min.js")

	// statically served directories
	d.registerStaticDir("generated/css", "/css/")
	d.registerStaticDir("generated/icons", "/icons/")
	d.registerStaticDir("generated/js", "/js/")
	d.registerStaticDir("dashboard/static/img", "/img/")
	d.registerStaticDir("dashboard/static/favicons", "/favicons/")

	// topics
	maintainers := maintainers.New(store, cache, a.CacheTTL, time.Duration(a.MaintainerActivityWindow), a.DefaultOrg)
	members := members.New(store, cache, a.CacheTTL, time.Duration(a.MemberActivityWindow), a.DefaultOrg, a.Orgs)
	issues := issues.New(store, cache)
	pullRequests := pullrequests.New(store, cache)
	perf := perf.New(store, cache)
	commitHub := commithub.New(store, cache)
	flakes := flakes.New(store, cache)
	coverage := coverage.New(store, cache)
	features := features.New(store, cache)
	workingGroups := workinggroups.New(store, cache)

	// all the sidebar entries and their associated UI pages
	d.addEntry("Maintainers", "Lists the folks that maintain the project.").
		addEntry("Recently Active", "Maintainers that have recently contributed to the project.").
		addPageWithQuery("/maintainers", "filter", "active", maintainers.RenderList).
		endEntry().
		addEntry("Recently Inactive", "Maintainers that have not recently contributed to the project.").
		addPageWithQuery("/maintainers", "filter", "inactive", maintainers.RenderList).
		endEntry().
		addEntry("Emeritus", "Maintainers that are no longer  involved with the project.").
		addPageWithQuery("/maintainers", "filter", "emeritus", maintainers.RenderList).
		endEntry().
		addPage("/maintainers", maintainers.RenderList).
		addPage("/maintainers/{login}", maintainers.RenderSingle).
		endEntry()

	d.addEntry("Members", "Lists the folks that help develop and manage the project.").
		addEntry("Recently Active", "Members that have recently contributed to the project.").
		addPageWithQuery("/members", "filter", "active", members.RenderList).
		endEntry().
		addEntry("Recently Inactive", "Members that have not recently contributed to the project.").
		addPageWithQuery("/members", "filter", "inactive", members.RenderList).
		endEntry().
		addPage("/members", members.RenderList).
		addPage("/members/{login}", members.RenderSingle).
		endEntry()

	d.addEntry("Working Groups", "Shows information about the project's working groups.").
		addPage("/workinggroups", workingGroups.Render).
		endEntry()

	d.addEntry("Issues", "Information on new and old issues.").
		addPage("/issues", issues.Render).
		endEntry()

	d.addEntry("Pull Requests", "Information on new and old pull requests.").
		addPage("/prs", pullRequests.Render).
		endEntry()

	d.addEntry("Performance", "Learn about the project's performance testing.").
		addPage("/perf", perf.Render).
		endEntry()

	d.addEntry("Commit Hub", "Interact with pull requests and commits.").
		addPage("/commits", commitHub.Render).
		endEntry()

	d.addEntry("Code Coverage", "Understand the project's code coverage.").
		addPage("/coverage", coverage.Render).
		endEntry()

	d.addEntry("Test Flakes", "Discover the wonderful world of test flakes.").
		addPage("/flakes", flakes.Render).
		endEntry()

	d.addEntry("Features and Test Plans", "Get information on product features and associated test plans.").
		addPage("/features", features.Render).
		endEntry()

	// home page
	var homeEntries []home.Entry
	for _, sbe := range d.entries {
		homeEntries = append(homeEntries, home.Entry{
			Title:       sbe.Title,
			Description: sbe.Description,
			URL:         sbe.URL,
		})
	}
	home := home.New(homeEntries)
	d.registerUIPage("/", home.Render)

	// 'page not found' error page
	router.NotFoundHandler = nf

	// oauth support
	router.HandleFunc("/login", d.oauthLogin)
	router.HandleFunc("/githuboauthcallback", d.oauthCallback)

	// API endpoints
	d.registerAPI("/api/maintainers/", maintainers.GetList)
	d.registerAPI("/api/members/", members.GetList)

	return d
}

func (d *Dashboard) addEntry(title string, description string) *sidebarEntry {
	newEntry := &sidebarEntry{
		Title:       title,
		Description: description,
		Dashboard:   d,
	}

	d.currentEntry = newEntry
	d.entries = append(d.entries, newEntry)

	return newEntry
}

func (d *Dashboard) registerUIPage(path string, render types.RenderFunc) *mux.Route {
	return d.router.
		StrictSlash(true).
		Methods("GET").
		Path(path).
		HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			renderInfo, err := render(req)
			if err != nil {
				d.renderError(w, err)
				return
			}

			entry := d.entryMap[mux.CurrentRoute(req)]

			title := renderInfo.Title
			if title == "" && entry != nil {
				title = entry.Title
			}

			desc := ""
			if entry != nil {
				desc = entry.Description
			}

			info := templateInfo{
				Title:         title,
				Description:   desc,
				Content:       renderInfo.Content,
				Control:       renderInfo.Control,
				Entries:       d.entries,
				SelectedEntry: entry,
			}

			var b bytes.Buffer
			if err := d.primaryTemplates.Execute(&b, info); err != nil {
				d.renderError(w, err)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = b.WriteTo(w)
		})
}

func (d *Dashboard) registerAPI(path string, handler http.HandlerFunc) {
	d.router.
		StrictSlash(true).
		Path(path).
		HandlerFunc(handler)
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

func (d *Dashboard) oauthLogin(w http.ResponseWriter, req *http.Request) {
	err := d.oauthHandler.ServeLogin(w, req)
	if err != nil {
		d.renderError(w, err)
	}
}

func (d *Dashboard) oauthCallback(w http.ResponseWriter, req *http.Request) {
	err := d.oauthHandler.ServeCallback(w, req)
	if err != nil {
		d.renderError(w, err)
	}
}

// renderError generates an error page
func (d *Dashboard) renderError(w http.ResponseWriter, err error) {
	info := templateInfo{
		Title:       "ERROR",
		Description: "ERROR",
		Content:     fmt.Sprintf("%v", err),
	}

	var b bytes.Buffer
	if tplErr := d.errorTemplates.Execute(&b, info); tplErr != nil {
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
