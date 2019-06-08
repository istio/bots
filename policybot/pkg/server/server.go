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

package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard/templates"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/fw"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/bots/policybot/plugins/handlers/github"
	"istio.io/bots/policybot/plugins/handlers/syncer"
	"istio.io/bots/policybot/plugins/handlers/zenhub"
	"istio.io/bots/policybot/plugins/topics/maintainers"
	"istio.io/bots/policybot/plugins/webhooks/cfgmonitor"
	"istio.io/bots/policybot/plugins/webhooks/labeler"
	"istio.io/bots/policybot/plugins/webhooks/nagger"
	"istio.io/bots/policybot/plugins/webhooks/refresher"
	"istio.io/pkg/log"
)

// Server represents a running bot instance.
type Server struct {
	listener   net.Listener
	shutdown   sync.WaitGroup
	httpServer http.Server
	allTopics  []fw.Topic

	clientID     string
	clientSecret string
	secretState  string
}

// Runs the server.
//
// If config comes from a container-based file, this will try to run the server, but if
// problems occur (probably due to bad config), then the function returns with an error.
//
// If config comes from a repo-based file, this will also try to run the server, but if an error
// occurs, it will refetch the config every minute and try again. And so in that case, this
// function never returns.
func RunServer(base *config.Args) error {
	for {
		// copy the baseline config
		cfg := *base

		// load the config file
		if err := fetchConfig(&cfg); err != nil {
			if cfg.StartupOptions.ConfigRepo != "" {
				log.Errorf("Unable to load configuration file, waiting for 1 minute and then will try again: %v", err)
				time.Sleep(time.Minute)
				continue
			} else {
				return fmt.Errorf("unable to load configuration file: %v", err)
			}
		}

		if err := runWithConfig(&cfg); err != nil {
			if cfg.StartupOptions.ConfigRepo != "" {
				log.Errorf("Unable to initialize server likely due to bad config, waiting for 1 minute and then will try again: %v", err)
				time.Sleep(time.Minute)
			} else {
				return fmt.Errorf("unable to initialize server: %v", err)
			}
		} else {
			log.Infof("Configuration change detected, attempting to reload configuration")
		}
	}
}

func runWithConfig(a *config.Args) error {
	log.Infof("Starting with:\n%s", a)

	creds, err := base64.StdEncoding.DecodeString(a.StartupOptions.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	ght := util.NewGitHubThrottle(context.Background(), a.StartupOptions.GitHubToken)
	_ = util.NewMailer(a.StartupOptions.SendGridAPIKey, a.EmailFrom, a.EmailOriginAddress)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	ghs := gh.NewGitHubState(store, a.CacheTTL)

	nag, err := nagger.NewNagger(context.Background(), ght, ghs, a.Orgs, a.Nags)
	if err != nil {
		return fmt.Errorf("unable to create nagger: %v", err)
	}

	labeler, err := labeler.NewLabeler(context.Background(), ght, ghs, a.Orgs, a.AutoLabels)
	if err != nil {
		return fmt.Errorf("unable to create labeler: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.StartupOptions.Port))
	if err != nil {
		return fmt.Errorf("unable to listen to port: %v", err)
	}

	router := mux.NewRouter()

	// secret state for OAuth exchanges
	secretState := make([]byte, 32)
	if _, err := rand.Read(secretState); err != nil {
		return fmt.Errorf("unable to generate secret state: %v", err)
	}

	s := &Server{
		listener: listener,
		httpServer: http.Server{
			Addr:           listener.Addr().(*net.TCPAddr).String(),
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        router,
		},
		clientID:     a.StartupOptions.GitHubOAuthClientID,
		clientSecret: a.StartupOptions.GitHubOAuthClientSecret,
		secretState:  base64.StdEncoding.EncodeToString(secretState),
	}

	monitor, err := cfgmonitor.NewMonitor(context.Background(), ght, a.StartupOptions.ConfigRepo, a.StartupOptions.ConfigFile, s.Close)
	if err != nil {
		return fmt.Errorf("unable to create config monitor: %v", err)
	}

	// core UI templates
	baseLayout := template.Must(template.New("base").Parse(templates.BaseTemplate)).Funcs(template.FuncMap{
		"getTopics": s.getTopics,
		"normalize": normalize,
	})
	_ = template.Must(baseLayout.Parse(templates.HeaderTemplate))
	_ = template.Must(baseLayout.Parse(templates.SidebarTemplate))
	mainLayout := template.Must(template.Must(baseLayout.Clone()).Parse(templates.MainTemplate))

	// github webhook handlers (keep refresher first in the list such that other plugins see an up-to-date view in storage)
	webhooks := []fw.Webhook{
		refresher.NewRefresher(store, a.Orgs),
		nag,
		labeler,
		monitor,
	}

	ghHandler, err := github.NewHandler(a.StartupOptions.GitHubWebhookSecret, webhooks...)
	if err != nil {
		return fmt.Errorf("unable to create GitHub webhook: %v", err)
	}

	// event handlers
	router.Handle("/githubwebhook", ghHandler).Methods("POST")
	router.Handle("/zenhubwebhook", zenhub.NewHandler()).Methods("POST")
	router.Handle("/sync", syncer.NewHandler(context.Background(), ght, ghs, store, a.Orgs)).Methods("GET")
	router.HandleFunc("/login", s.handleLogin)
	router.HandleFunc("/githuboauthcallback", s.handleOAuthCallback)

	// statically served directories
	registerStaticDir(router, "dashboard/generated/css", "/css/")
	registerStaticDir(router, "dashboard/generated/icons", "/icons/")
	registerStaticDir(router, "dashboard/generated/js", "/js/")
	registerStaticDir(router, "dashboard/static/img", "/img/")
	registerStaticDir(router, "dashboard/static/favicons", "/favicons/")

	// statically serve files
	registerStaticFile(router, "dashboard/static/favicon.ico", "/favicon.ico")
	registerStaticFile(router, "dashboard/static/browserconfig.xml", "/browserconfig.xml")
	registerStaticFile(router, "dashboard/static/manifest.json", "/manifest.json")

	// UI topics
	s.registerTopic(router, mainLayout, maintainers.NewMaintainerQueries(store, ghs))

	// home page
	router.
		Path("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fw.RenderHTML(w, template.Must(template.Must(mainLayout.Clone()).Parse(templates.HomeTemplate)), nil)
		})

	// not found fallback
	router.
		PathPrefix("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fw.RenderHTML(w, template.Must(template.Must(mainLayout.Clone()).Parse(templates.PageNotFoundTemplate)), nil)
		})

	log.Infof("Listening on port %d", a.StartupOptions.Port)
	err = s.httpServer.Serve(s.listener)
	if err != http.ErrServerClosed {
		return fmt.Errorf("listening on port %d failed: %v", a.StartupOptions.Port, err)
	}

	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Warnf("Error shutting down: %v", err)
		}
		s.shutdown.Wait()
	}
}

func registerStaticDir(router *mux.Router, fsPath string, sitePath string) {
	router.
		PathPrefix(sitePath).
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(sitePath, http.FileServer(http.Dir(fsPath))).ServeHTTP(w, r)
		})
}

func registerStaticFile(router *mux.Router, fsPath string, sitePath string) {
	router.
		Path(sitePath).
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, fsPath)
		})
}

func (s *Server) registerTopic(router *mux.Router, layout *template.Template, t fw.Topic) {
	htmlRouter := router.NewRoute().PathPrefix("/" + t.Prefix()).Subrouter()
	jsonRouter := router.NewRoute().PathPrefix("/" + t.Prefix() + "api").Subrouter()

	tmpl := template.Must(template.Must(layout.Clone()).Parse("{{ define \"title\" }}" + t.Title() + "{{ end }}"))
	t.Activate(fw.NewContext(htmlRouter, jsonRouter, tmpl))

	s.allTopics = append(s.allTopics, t)
}

type topic struct {
	Name string
	URL  string
}

func (s *Server) getTopics() []topic {
	var result []topic
	for _, t := range s.allTopics {
		result = append(result, topic{
			Name: t.Title(),
			URL:  "/" + t.Prefix(),
		})
	}
	return result
}

func normalize(input string) string {
	return strings.Replace(input, "/", "-", -1)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	url := "https://github.com/login/oauth/authorize?client_id=" + s.clientID + "&scope=user,repo&state=" + s.secretState
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	httpClient := http.Client{}

	if err := r.ParseForm(); err != nil {
		fw.RenderError(w, http.StatusBadRequest, fmt.Errorf("unable to parse query: %v", err))
		return
	}

	if r.FormValue("state") != s.secretState {
		fw.RenderError(w, http.StatusBadRequest, fmt.Errorf("unable to verify request state"))
		return
	}

	url := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s", s.clientID, s.clientSecret, r.FormValue("code"))
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fw.RenderError(w, http.StatusInternalServerError, fmt.Errorf("unable to create request: %v", err))
		return
	}
	// ask for the response in JSON
	req.Header.Set("accept", "application/json")

	// send out the request to GitHub for the access token
	res, err := httpClient.Do(req)
	if err != nil {
		fw.RenderError(w, http.StatusInternalServerError, fmt.Errorf("unable to contact GitHub: %v", err))
		return
	}
	defer res.Body.Close()

	var t OAuthAccessResponse
	if err := json.NewDecoder(res.Body).Decode(&t); err != nil {
		fw.RenderError(w, http.StatusBadRequest, fmt.Errorf("unable to parse response from GitHub: %v", err))
		return
	}

	// finally, have GitHub redirect the user to the home page, passing the access token to the page
	w.Header().Set("Location", "/?access_token="+t.AccessToken)
	w.WriteHeader(http.StatusFound)
}

type OAuthAccessResponse struct {
	AccessToken string `json:"access_token"`
}
