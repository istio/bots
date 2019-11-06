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

package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"istio.io/bots/policybot/dashboard"
	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/boilerplatecleaner"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/cfgmonitor"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/labeler"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/lifecyclerfilter"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/nagger"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/refresher"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/testresultfilter"
	"istio.io/bots/policybot/handlers/zenhubwebhook"
	"istio.io/bots/policybot/mgrs/lifecyclemgr"
	"istio.io/bots/policybot/pkg/blobstorage/gcs"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

const (
	port                   = "TCP port to listen to for incoming traffic"
	httpsOnly              = "Send https redirect if x-forwarded-header is not set"
	enableTestResultFilter = "Enable the test result Github webhook filter"
)

func serverCmd() *cobra.Command {
	serverCmd, ca := config.Run("server", "Starts the policybot server", 0,
		config.GithubOAuthClientID|
			config.SendgridAPIKey|
			config.GithubOAuthClientSecret|
			config.GitHubWebhookSecret|
			config.ConfigFile|
			config.ConfigRepo|
			config.ZenhubToken|
			config.GitHubToken|
			config.GCPCreds|
			config.ControlZ, runServer)

	serverCmd.PersistentFlags().IntVarP(&ca.ServerPort,
		"port", "", ca.ServerPort, port)
	serverCmd.PersistentFlags().BoolVarP(&ca.HTTPSOnly,
		"https_only", "", ca.HTTPSOnly, httpsOnly)
	serverCmd.PersistentFlags().BoolVarP(&ca.EnableTestResultFilter,
		"enable_test_result_filter", "", ca.EnableTestResultFilter,
		enableTestResultFilter)

	return serverCmd
}

// Server represents a running bot instance.
type server struct {
	listener net.Listener
}

// Runs the server.
//
// If config comes from a container-based file, this will try to run the server, but if
// problems occur (probably due to bad config), then the function returns with an error.
//
// If config comes from a repo-based file, this will also try to run the server, but if an error
// occurs, it will refetch the config every minute and try again. And so in that case, this
// function never returns.
func runServer(base *config.Args, _ []string) error {
	for {
		// copy the baseline config
		c := *base
		cfg := &c

		// load the config file
		if err := cfg.Fetch(); err != nil {
			if cfg.ConfigRepo != "" {
				log.Errorf("Unable to load configuration file, waiting for 1 minute and then will try again: %v", err)
				time.Sleep(time.Minute)
				continue
			} else {
				return fmt.Errorf("unable to load configuration file: %v", err)
			}
		}

		if err := runWithConfig(cfg); err != nil {
			if cfg.ConfigRepo != "" {
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
	log.Debugf("Starting with:\n%s", a)

	creds, err := base64.StdEncoding.DecodeString(a.Secrets.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	gc := gh.NewThrottledClient(context.Background(), a.Secrets.GitHubToken)
	_ = util.NewMailer(a.Secrets.SendGridAPIKey, a.EmailFrom, a.EmailOriginAddress)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	bs, err := gcs.NewStore(context.Background(), creds)
	if err != nil {
		return fmt.Errorf("unable to create blob storage layer: %v", err)
	}
	defer bs.Close()

	cache := cache.New(store, time.Duration(a.CacheTTL))

	nag, err := nagger.NewNagger(gc, cache, a.Orgs, a.Nags)
	if err != nil {
		return fmt.Errorf("unable to create nagger: %v", err)
	}

	labeler, err := labeler.NewLabeler(gc, cache, a.Orgs, a.AutoLabels)
	if err != nil {
		return fmt.Errorf("unable to create labeler: %v", err)
	}

	cleaner, err := boilerplatecleaner.New(gc, a.Orgs, a.BoilerplatesToClean)
	if err != nil {
		return fmt.Errorf("unable to create boilerplate cleaner: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.ServerPort))
	if err != nil {
		return fmt.Errorf("unable to listen to port: %v", err)
	}

	router := mux.NewRouter()

	httpServer := http.Server{
		Addr:           listener.Addr().(*net.TCPAddr).String(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        router,
	}

	s := &server{
		listener: listener,
	}

	monitor, err := cfgmonitor.NewMonitor(a.ConfigRepo, a.ConfigFile, s.Close)
	if err != nil {
		return fmt.Errorf("unable to create config monitor: %v", err)
	}

	lf := lifecyclemgr.New(gc, store, a)

	// github webhook filters (keep refresher first in the list such that other filter see an up-to-date view in storage)
	filters := []filters.Filter{
		refresher.NewRefresher(cache, store, gc, a.Orgs),
		nag,
		lifecyclerfilter.NewLifecyclerFilter(gc, a.Orgs, lf),
		labeler,
		cleaner,
		monitor,
	}

	if a.EnableTestResultFilter {
		testFilter := testresultfilter.NewTestResultFilter(cache, a.Orgs, gc, bs, store)
		filters = append(filters, testFilter)
	}

	if a.HTTPSOnly {
		// we only want https
		router.Headers("X-Forwarded-Proto", "HTTP").HandlerFunc(handleHTTP)
	}

	// top-level handlers
	router.Handle("/githubwebhook", githubwebhook.NewHandler(a.Secrets.GitHubWebhookSecret, filters...)).Methods("POST")
	router.Handle("/zenhubwebhook", zenhubwebhook.NewHandler(store, cache, lf)).Methods("POST")

	// prep the UI
	_ = dashboard.New(router, store, cache, a)

	log.Infof("Listening on port %d", a.ServerPort)
	err = httpServer.Serve(s.listener)
	if err != http.ErrServerClosed {
		return fmt.Errorf("listening on port %d failed: %v", a.ServerPort, err)
	}

	return nil
}

func (s *server) Close() {
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Warnf("Error shutting down: %v", err)
		}
	}
}

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("https://%s%s", r.Host, r.URL), http.StatusPermanentRedirect)
}
