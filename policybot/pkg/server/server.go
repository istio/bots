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
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/ghodss/yaml"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/bots/policybot/plugins/analyzer"
	"istio.io/bots/policybot/plugins/cfgmonitor"
	"istio.io/bots/policybot/plugins/labeler"
	"istio.io/bots/policybot/plugins/nagger"
	"istio.io/bots/policybot/plugins/refresher"
	"istio.io/bots/policybot/plugins/syncer"
	"istio.io/pkg/log"
)

// Runs the server.
//
// If config comes from a container-based file, this will try to run the server, but if
// problems occur (probably due to bad config), then the function returns with an error.
//
// If config comes from a repo-based file, this will also try to run the server, but if an error
// occurs, it will refetch the config every minute and try again. And so in that case, this
// function never returns.
func RunServer(a *config.Args) error {
	for {
		// copy the baseline config
		cfg := *a

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

		if err := serve(&cfg); err != nil {
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

// Runs the syncer.
//
// If config comes from a container-based file, this will try to run the server, but if
// problems occur (probably due to bad config), then the function returns with an error.
//
// If config comes from a repo-based file, this will also try to run the server, but if an error
// occurs, it will refetch the config every minute and try again. And so in that case, this
// function never returns.
func RunSyncer(a *config.Args) error {
	// load the config file
	if err := fetchConfig(a); err != nil {
		return fmt.Errorf("unable to load configuration file: %v", err)
	}

	creds, err := base64.StdEncoding.DecodeString(a.StartupOptions.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	ght := util.NewGitHubThrottle(context.Background(), a.StartupOptions.GitHubToken)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	ghs := gh.NewGitHubState(store, a.CacheTTL)

	prepStore(context.Background(), ght, ghs, a.Orgs)
	syncer.NewSyncer(context.Background(), ght, ghs, store, a.Orgs).Sync()
	return nil
}

func fetchConfig(a *config.Args) error {
	if a.StartupOptions.ConfigFile == "" {
		return errors.New("no configuration file supplied")
	}

	var b []byte
	var err error

	if a.StartupOptions.ConfigRepo == "" {
		if b, err = ioutil.ReadFile(a.StartupOptions.ConfigFile); err != nil {
			return fmt.Errorf("unable to read configuration file %s: %v", a.StartupOptions.ConfigFile, err)
		}

		if err = yaml.Unmarshal(b, &a); err != nil {
			return fmt.Errorf("unable to parse configuration file %s: %v", a.StartupOptions.ConfigFile, err)
		}
	} else {
		url := "https://raw.githubusercontent.com/" + a.StartupOptions.ConfigRepo + "/" + a.StartupOptions.ConfigFile
		r, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("unable to fetch configuration file from %s: %v", url, err)
		}
		if r.StatusCode >= 400 {
			return fmt.Errorf("unable to fetch configuration file from %s: status code %d", url, r.StatusCode)
		}
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			return fmt.Errorf("unable to read configuration file from %s: %v", url, err)
		}

		if err = yaml.Unmarshal(b, &a); err != nil {
			return fmt.Errorf("unable to parse configuration file from %s: %v", url, err)
		}
	}

	return nil
}

func serve(a *config.Args) error {
	log.Infof("Starting with:\n%s", a)

	creds, err := base64.StdEncoding.DecodeString(a.StartupOptions.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	serverMux := http.NewServeMux()
	ght := util.NewGitHubThrottle(context.Background(), a.StartupOptions.GitHubToken)
	_ = util.NewMailer(a.StartupOptions.SendGridAPIKey, a.EmailFrom, a.EmailOriginAddress)

	store, err := spanner.NewStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		return fmt.Errorf("unable to create storage layer: %v", err)
	}
	defer store.Close()

	ghs := gh.NewGitHubState(store, a.CacheTTL)

	prepStore(context.Background(), ght, ghs, a.Orgs)

	nag, err := nagger.NewNagger(context.Background(), ght, ghs, a.Orgs, a.Nags)
	if err != nil {
		return fmt.Errorf("unable to create nagger: %v", err)
	}

	labeler, err := labeler.NewLabeler(context.Background(), ght, ghs, a.Orgs, a.AutoLabels)
	if err != nil {
		return fmt.Errorf("unable to create labeler: %v", err)
	}

	refresh := refresher.NewRefresher(ghs, a.Orgs)

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(a.StartupOptions.Port),
		Handler: serverMux,
	}

	monitor, err := cfgmonitor.NewMonitor(context.Background(), ght, a.StartupOptions.ConfigRepo, a.StartupOptions.ConfigFile, func() {
		// stop the web server when we detect config changes, this causes a reload of everything
		_ = srv.Shutdown(context.Background())
	})

	if err != nil {
		return fmt.Errorf("unable to create config monitor: %v", err)
	}

	// NB: keep refresher first in the list such that other plugins see an up-to-date view in storage.
	hook, err := newHook(a.StartupOptions.GitHubSecret,
		refresh,
		nag,
		labeler,
		monitor)
	if err != nil {
		return fmt.Errorf("unable to create GitHub webhook: %v", err)
	}

	register(serverMux, "/githubwebhook", hook.handle)
	register(serverMux, "/sync", syncer.NewSyncer(context.Background(), ght, ghs, store, a.Orgs).Handle)
	register(serverMux, "/repos", analyzer.NewAnalyzer(store).Handle)

	log.Infof("Listening on port %d", a.StartupOptions.Port)
	err = srv.ListenAndServe()
	if err != http.ErrServerClosed {
		return fmt.Errorf("listening on port %d failed: %v", a.StartupOptions.Port, err)
	}

	return nil
}

func prepStore(ctx context.Context, ght *util.GitHubThrottle, ghs *gh.GitHubState, orgs []config.Org) {
	a := ghs.NewAccumulator()

	for _, orgConfig := range orgs {
		for _, repoConfig := range orgConfig.Repos {
			if repo, _, err := ght.Get().Repositories.Get(ctx, orgConfig.Name, repoConfig.Name); err != nil {
				log.Errorf("Unable to query information about repository %s/%s from GitHub: %v", orgConfig.Name, repoConfig.Name, err)
			} else {
				_ = a.RepoFromAPI(repo)
			}
		}
	}

	if err := a.Commit(); err != nil {
		log.Errorf("Unable to commit data to storage: %v", err)
	}
}

func register(mux *http.ServeMux, pattern string, handler func(w http.ResponseWriter, h *http.Request)) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		handler(w, r)

		log.Infof(
			"%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	})
}
