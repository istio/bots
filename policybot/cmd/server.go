// Copyrigc 2019 Istio Authors
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
	"google.golang.org/grpc/grpclog"

	"istio.io/bots/policybot/dashboard"
	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/cfgmonitor"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/labeler"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/nagger"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/refresher"
	"istio.io/bots/policybot/handlers/githubwebhook/filters/unstaler"
	"istio.io/bots/policybot/handlers/zenhubwebhook"
	"istio.io/bots/policybot/pkg/blobstorage/gcs"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/storage/spanner"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/ctrlz"
	"istio.io/pkg/env"
	"istio.io/pkg/log"
)

const (
	githubWebhookSecret     = "Secret for the GitHub webhook"
	githubToken             = "Token to access the GitHub API"
	gcpCreds                = "Base64-encoded credentials to access GCP"
	configRepo              = "GitHub org/repo/branch where to fetch policybot config"
	configFile              = "Path to a configuration file"
	sendgridAPIKey          = "API Key for sendgrid.com"
	zenhubToken             = "Token to access the ZenHub API"
	port                    = "TCP port to listen to for incoming traffic"
	githubOAuthClientSecret = "Client secret for GitHub OAuth2 flow"
	githubOAuthClientID     = "Client ID for GitHub OAuth2 flow"
	httpsOnly               = "Send https redirect if x-forwarded-header is not set"
)

func serverCmd() *cobra.Command {
	ca := config.DefaultArgs()

	ca.StartupOptions.GitHubWebhookSecret = env.RegisterStringVar("GITHUB_WEBHOOK_SECRET", ca.StartupOptions.GitHubWebhookSecret, githubWebhookSecret).Get()
	ca.StartupOptions.GitHubToken = env.RegisterStringVar("GITHUB_TOKEN", ca.StartupOptions.GitHubToken, githubToken).Get()
	ca.StartupOptions.ZenHubToken = env.RegisterStringVar("ZENHUB_TOKEN", ca.StartupOptions.ZenHubToken, zenhubToken).Get()
	ca.StartupOptions.GCPCredentials = env.RegisterStringVar("GCP_CREDS", ca.StartupOptions.GCPCredentials, gcpCreds).Get()
	ca.StartupOptions.ConfigRepo = env.RegisterStringVar("CONFIG_REPO", ca.StartupOptions.ConfigRepo, configRepo).Get()
	ca.StartupOptions.ConfigFile = env.RegisterStringVar("CONFIG_FILE", ca.StartupOptions.ConfigFile, configFile).Get()
	ca.StartupOptions.SendGridAPIKey = env.RegisterStringVar("SENDGRID_APIKEY", ca.StartupOptions.SendGridAPIKey, sendgridAPIKey).Get()
	ca.StartupOptions.Port = env.RegisterIntVar("PORT", ca.StartupOptions.Port, port).Get()
	ca.StartupOptions.GitHubOAuthClientSecret =
		env.RegisterStringVar("GITHUB_OAUTH_CLIENT_SECRET", ca.StartupOptions.GitHubOAuthClientSecret, githubOAuthClientSecret).Get()
	ca.StartupOptions.GitHubOAuthClientID =
		env.RegisterStringVar("GITHUB_OAUTH_CLIENT_ID", ca.StartupOptions.GitHubOAuthClientID, githubOAuthClientID).Get()
	env.RegisterBoolVar("HTTPS_ONLY", ca.StartupOptions.HTTPSOnly, httpsOnly).Get()

	loggingOptions := log.DefaultOptions()
	introspectionOptions := ctrlz.DefaultOptions()

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Starts the policybot server",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.Configure(loggingOptions); err != nil {
				log.Errorf("Unable to configure logging: %v", err)
			}

			// neutralize gRPC logging since it spews out useless junk
			var dummy = dummyIoWriter{}
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

			if cs, err := ctrlz.Run(introspectionOptions, nil); err == nil {
				defer cs.Close()
			} else {
				log.Errorf("Unable to initialize ControlZ: %v", err)
			}

			cmd.SilenceUsage = true
			return runServer(ca)
		},
	}

	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigRepo,
		"config_repo", "", ca.StartupOptions.ConfigRepo, configRepo)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ConfigFile,
		"config_file", "", ca.StartupOptions.ConfigFile, configFile)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubWebhookSecret,
		"github_webhook_secret", "", ca.StartupOptions.GitHubWebhookSecret, githubWebhookSecret)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubToken,
		"github_token", "", ca.StartupOptions.GitHubToken, githubToken)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GCPCredentials,
		"gcp_creds", "", ca.StartupOptions.GCPCredentials, gcpCreds)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.SendGridAPIKey,
		"sendgrid_apikey", "", ca.StartupOptions.SendGridAPIKey, sendgridAPIKey)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.ZenHubToken,
		"zenhub_token", "", ca.StartupOptions.ZenHubToken, zenhubToken)
	serverCmd.PersistentFlags().IntVarP(&ca.StartupOptions.Port,
		"port", "", ca.StartupOptions.Port, port)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubOAuthClientSecret,
		"github_oauth_client_secret", "", ca.StartupOptions.GitHubOAuthClientSecret, githubOAuthClientSecret)
	serverCmd.PersistentFlags().StringVarP(&ca.StartupOptions.GitHubOAuthClientID,
		"github_oauth_client_id", "", ca.StartupOptions.GitHubOAuthClientID, githubOAuthClientID)
	serverCmd.PersistentFlags().BoolVarP(&ca.StartupOptions.HTTPSOnly,
		"https_only", "", ca.StartupOptions.HTTPSOnly, httpsOnly)

	loggingOptions.AttachCobraFlags(serverCmd)
	introspectionOptions.AttachCobraFlags(serverCmd)

	return serverCmd
}

type dummyIoWriter struct{}

func (dummyIoWriter) Write([]byte) (int, error) { return 0, nil }

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
func runServer(base *config.Args) error {
	for {
		// copy the baseline config
		c := *base
		cfg := &c

		// load the config file
		if err := cfg.Fetch(); err != nil {
			if cfg.StartupOptions.ConfigRepo != "" {
				log.Errorf("Unable to load configuration file, waiting for 1 minute and then will try again: %v", err)
				time.Sleep(time.Minute)
				continue
			} else {
				return fmt.Errorf("unable to load configuration file: %v", err)
			}
		}

		if err := runWithConfig(cfg); err != nil {
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
	log.Debugf("Starting with:\n%s", a)

	creds, err := base64.StdEncoding.DecodeString(a.StartupOptions.GCPCredentials)
	if err != nil {
		return fmt.Errorf("unable to decode GCP credentials: %v", err)
	}

	gc := gh.NewThrottledClient(context.Background(), a.StartupOptions.GitHubToken)
	_ = util.NewMailer(a.StartupOptions.SendGridAPIKey, a.EmailFrom, a.EmailOriginAddress)

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

	cache := cache.New(store, a.CacheTTL)

	nag, err := nagger.NewNagger(gc, cache, a.Orgs, a.Nags)
	if err != nil {
		return fmt.Errorf("unable to create nagger: %v", err)
	}

	labeler, err := labeler.NewLabeler(gc, cache, a.Orgs, a.AutoLabels)
	if err != nil {
		return fmt.Errorf("unable to create labeler: %v", err)
	}

	unstaler, err := unstaler.NewUnstaler(gc, a.Orgs)
	if err != nil {
		return fmt.Errorf("unable to create unstaler: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.StartupOptions.Port))
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

	monitor, err := cfgmonitor.NewMonitor(a.StartupOptions.ConfigRepo, a.StartupOptions.ConfigFile, s.Close)
	if err != nil {
		return fmt.Errorf("unable to create config monitor: %v", err)
	}

	// github webhook filters (keep refresher first in the list such that other filter see an up-to-date view in storage)
	filters := []filters.Filter{
		refresher.NewRefresher(cache, store, gc, a.Orgs),
		nag,
		unstaler,
		labeler,
		monitor,
		//		resultgatherer.NewResultGatherer(store, cache, a.Orgs, a.BucketName),
	}

	if a.StartupOptions.HTTPSOnly {
		// we only want https
		router.Headers("X-Forwarded-Proto", "HTTP").HandlerFunc(handleHTTP)
	}

	// top-level handlers
	router.Handle("/githubwebhook", githubwebhook.NewHandler(a.StartupOptions.GitHubWebhookSecret, filters...)).Methods("POST")
	router.Handle("/zenhubwebhook", zenhubwebhook.NewHandler(store, cache)).Methods("POST")

	// prep the UI
	_ = dashboard.New(router, store, cache, a)

	log.Infof("Listening on port %d", a.StartupOptions.Port)
	err = httpServer.Serve(s.listener)
	if err != http.ErrServerClosed {
		return fmt.Errorf("listening on port %d failed: %v", a.StartupOptions.Port, err)
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
