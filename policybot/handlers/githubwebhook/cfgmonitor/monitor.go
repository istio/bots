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

package cfgmonitor

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/pkg/log"
)

// Monitors for changes in the bot's config file.
type Monitor struct {
	org    string
	repo   string
	branch string
	file   string
	notify func()
}

var scope = log.RegisterScope("monitor", "Listens for changes in policybot config", 0)

func NewMonitor(repo string, file string, notify func()) (githubwebhook.Filter, error) {
	if repo == "" {
		// disable everything if we don't have a repo
		return &Monitor{}, nil
	}

	splits := strings.Split(repo, "/")
	if len(splits) != 3 {
		return nil, fmt.Errorf("invalid value for configuration repo, needs to be org/repo/branch, is `%s`", repo)
	}

	ct := &Monitor{
		org:    splits[0],
		repo:   splits[1],
		branch: splits[2],
		file:   file,
		notify: notify,
	}
	return ct, nil
}

// monitor for changes to policybot's config file
func (m *Monitor) Handle(context context.Context, event interface{}) {
	pp, ok := event.(github.PushEvent)
	if !ok {
		// not what we're looking for
		return
	}

	if pp.GetRepo().GetOwner().GetLogin() != m.org || pp.GetRepo().GetName() != m.repo {
		// not the org/repo we care about
		return
	}

	// TODO: ensure the right branch (m.branch) is being affected, not sure how to get the branch info sadly

	for _, commit := range pp.Commits {
		for _, s := range commit.Modified {
			if s == m.file {
				scope.Infof("Detected modification to config file %s in repo %s", m.file, m.repo)
				m.notify()
				return
			}
		}

		for _, s := range commit.Added {
			if s == m.file {
				scope.Infof("Detected addition of config file %s in repo %s", m.file, m.repo)
				m.notify()
				return
			}
		}

		for _, s := range commit.Removed {
			if s == m.file {
				scope.Infof("Detected removal of config file %s in repo %s", m.file, m.repo)
				m.notify()
				return
			}
		}
	}
}
