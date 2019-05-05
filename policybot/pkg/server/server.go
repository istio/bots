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
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc/grpclog"

	"istio.io/istio/pkg/ctrlz"
	"istio.io/istio/pkg/log"

	"istio.io/bots/policybot/pkg/storage"
)

// Runs the server
func Run(a *Args) error {
	if err := log.Configure(a.LoggingOptions); err != nil {
		log.Errorf("Unable to configure logging: %v", err)
	}

	// neutralize gRPC logging since it spews out useless junk
	var dummy = dummyIoWriter{}
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(dummy, dummy, dummy))

	if cs, err := ctrlz.Run(a.IntrospectionOptions, nil); err == nil {
		defer cs.Close()
	} else {
		log.Errorf("Unable to initialize ControlZ: %v", err)
	}

	creds, err := base64.StdEncoding.DecodeString(a.GCPCredentials)
	if err != nil {
		log.Errorf("Unable to decode GCP credentials: %v", err)
		return err
	}

	store, err := storage.NewSpannerStore(context.Background(), a.SpannerDatabase, creds)
	if err != nil {
		log.Errorf("Unable to create storage layer: %v", err)
		return err
	}
	defer store.Close()

	serverMux := http.NewServeMux()

	deltaCollector, err := newDeltaCollector(a.GitHubSecret, store)
	if err != nil {
		log.Errorf("Unable to create GitHub delta collector: %v", err)
	} else {
		register(serverMux, "/githubwebhook", deltaCollector.handle)
	}

	reconciliator := newReconciliator(context.Background(), a.GitHubAccessToken, a.Orgs, store)
	register(serverMux, "/reconcile", reconciliator.handle)

	analyzer := newAnalyzer(store)
	register(serverMux, "/repos", analyzer.getRepos)

	log.Infof("Listening on port %d", a.Port)
	err = http.ListenAndServe(":"+strconv.Itoa(a.Port), serverMux)
	log.Errorf("Port listening failed: %v", err)

	return err
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

type dummyIoWriter struct{}

func (dummyIoWriter) Write([]byte) (int, error) { return 0, nil }
