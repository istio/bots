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
	"istio.io/bots/policybot/handlers/githubwebhook/cleaner"
	"istio.io/bots/policybot/handlers/githubwebhook/labeler"
	"istio.io/bots/policybot/handlers/githubwebhook/lifecycler"
	"istio.io/bots/policybot/handlers/githubwebhook/nagger"
	"istio.io/bots/policybot/handlers/githubwebhook/refresher"
	"istio.io/bots/policybot/handlers/githubwebhook/watcher"
	"istio.io/bots/policybot/handlers/githubwebhook/welcomer"
	"istio.io/bots/policybot/mgrs/lifecyclemgr"
	"istio.io/bots/policybot/pkg/blobstorage/gcs"
	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

const (
	httpsOnly = "Send https redirect if x-forwarded-header is not set"
)

func serverCmd() *cobra.Command {
	httpsOnlyVar := false

	serverCmd, _ := cmdutil.Run("server", "Starts the policybot server", 0,
		cmdutil.GithubOAuthClientID|
			cmdutil.SendgridAPIKey|
			cmdutil.GithubOAuthClientSecret|
			cmdutil.GitHubWebhookSecret|
			cmdutil.ConfigPath|
			cmdutil.ConfigRepo|
			cmdutil.GitHubToken|
			cmdutil.GCPCreds|
			cmdutil.ControlZ, func(reg *config.Registry, secrets *cmdutil.Secrets) error {
			return runServer(reg, secrets, httpsOnlyVar)
		})

	serverCmd.PersistentFlags().BoolVarP(&httpsOnlyVar, "https_only", "", httpsOnlyVar, httpsOnly)

	return serverCmd
}

// Server represents a running bot instance.
type server struct {
	listener net.Listener
}

// Runs the server.
//
// If config comes from a container-based directory, this will try to run the server, but if
// problems occur (probably due to bad config), then the function returns with an error.
//
// If config comes from a repo-based directory, this will also try to run the server, but if an error
// occurs, it will refetch the config every minute and try again. And so in that case, this
// function never returns.
func runServer(reg *config.Registry, secrets *cmdutil.Secrets, httpsOnly bool) error {
	for {
		if err := runWithConfig(reg, secrets, httpsOnly); err != nil {
			if reg.OriginRepo() != (gh.RepoDesc{}) {
				log.Errorf("Unable to initialize server likely due to bad config, waiting for 1 minute and then will try again: %v", err)
				time.Sleep(time.Minute)
			} else {
				return fmt.Errorf("unable to initialize server: %v", err)
			}
		} else {
			log.Infof("Configuration change detected, attempting to reload configuration")

			var err error
			var newReg *config.Registry
			if reg.OriginRepo() == (gh.RepoDesc{}) {
				newReg, err = config.LoadRegistryFromDirectory(reg.OriginPath())
			} else {
				gc := gh.NewThrottledClient(context.Background(), secrets.GitHubToken)
				newReg, err = config.LoadRegistryFromRepo(gc, reg.OriginRepo(), reg.OriginPath())
			}

			if err != nil {
				log.Errorf("Unable to load new config, keeping existing config: %v", err)
			} else {
				reg = newReg
			}
		}
	}
}

func runWithConfig(reg *config.Registry, secrets *cmdutil.Secrets, httpsOnly bool) error {
	log.Debugf("Starting up")

	creds, err := base64.StdEncoding.DecodeString(secrets.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	core := reg.Core()

	store, err := spanner.NewStore(context.Background(), core.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	bs, err := gcs.NewStore(context.Background(), creds)
	if err != nil {
		return fmt.Errorf("unable to create blob storage layer: %v", err)
	}
	defer bs.Close()

	c := cache.New(store, time.Duration(core.CacheTTL))
	gc := gh.NewThrottledClient(context.Background(), secrets.GitHubToken)
	_ = util.NewMailer(secrets.SendGridAPIKey, core.EmailFrom, core.EmailOriginAddress)
	lf := lifecyclemgr.New(gc, store, c, reg)

	nag, err := nagger.NewNagger(gc, c, reg)
	if err != nil {
		return fmt.Errorf("unable to create nagger: %v", err)
	}

	labeler, err := labeler.NewLabeler(gc, c, reg)
	if err != nil {
		return fmt.Errorf("unable to create labeler: %v", err)
	}

	cleaner, err := cleaner.New(gc, reg)
	if err != nil {
		return fmt.Errorf("unable to create boilerplate cleaner: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", core.ServerPort))
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

	// github webhook filters (keep refresher first in the list such that other filter see an up-to-date view in storage)
	filters := []githubwebhook.Filter{
		refresher.NewRefresher(c, store, bs, gc, reg),
		nag,
		lifecycler.New(gc, reg, lf, c),
		labeler,
		cleaner,
		welcomer.NewWelcomer(gc, store, c, reg),
		watcher.NewRepoWatcher(reg.OriginRepo(), reg.OriginPath(), s.Close),
	}

	if httpsOnly {
		// we only want https
		router.Headers("X-Forwarded-Proto", "HTTP").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fmt.Sprintf("https://%s%s", r.Host, r.URL), http.StatusPermanentRedirect)
		})
	}

	// top-level handlers
	router.Handle("/githubwebhook", githubwebhook.NewHandler(secrets.GitHubWebhookSecret, filters...)).Methods("POST")

	// prep the UI
	_ = dashboard.New(router, store, c, reg, secrets)

	log.Infof("Listening on port %d", core.ServerPort)

	err = httpServer.Serve(s.listener)
	if err != http.ErrServerClosed {
		return fmt.Errorf("listening on port %d failed: %v", core.ServerPort, err)
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
