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
	"bytes"
	"fmt"

	"istio.io/pkg/ctrlz"
	"istio.io/pkg/log"
)

type Repo struct {
	Name string
}

type Org struct {
	Name  string
	Repos []Repo
}

type Args struct {
	Port                 int
	GitHubSecret         string
	GitHubAccessToken    string
	GCPCredentials       string
	SpannerDatabase      string
	Orgs                 []Org
	LoggingOptions       *log.Options
	IntrospectionOptions *ctrlz.Options
}

func DefaultArgs() *Args {
	return &Args{
		Port:                 8080,
		LoggingOptions:       log.DefaultOptions(),
		IntrospectionOptions: ctrlz.DefaultOptions(),
	}
}

// String produces a stringified version of the arguments for debugging.
func (a *Args) String() string {
	buf := &bytes.Buffer{}

	// don't output secrets in the logs...
	// _, _ = fmt.Fprintf(buf, "GitHubSecret: %s\n", a.GitHubSecret)
	// _, _ = fmt.Fprintf(buf, "GitHubAccessToken: %s\n", a.GitHubAccessToken)
	// _, _ = fmt.Fprintf(buf, "GCPCredentials: %s\n", a.GCPCredentials)

	_, _ = fmt.Fprintf(buf, "Port: %d\n", a.Port)
	_, _ = fmt.Fprintf(buf, "SpannerDatabase: %s\n", a.SpannerDatabase)
	_, _ = fmt.Fprintf(buf, "Orgs: %+v\n", a.Orgs)
	_, _ = fmt.Fprintf(buf, "LoggingOptions: %#v\n", a.LoggingOptions)
	_, _ = fmt.Fprintf(buf, "IntrospectionOptions: %#v\n", a.IntrospectionOptions)

	return buf.String()
}
