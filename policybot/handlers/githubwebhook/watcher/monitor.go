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

package watcher

import (
	"context"
	"strings"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/istio/pkg/log"
)

// RepoWatcher waits for changes to files in GitHub.
type RepoWatcher struct {
	repo   gh.RepoDesc
	path   string
	notify func()
}

var scope = log.RegisterScope("watcher", "Listens for changes in GitHub files")

func NewRepoWatcher(repo gh.RepoDesc, path string, notify func()) githubwebhook.Filter {
	return &RepoWatcher{
		repo:   repo,
		path:   path,
		notify: notify,
	}
}

// monitor for changes to policybot's configuration
func (m *RepoWatcher) Handle(context context.Context, event interface{}) {
	pp, ok := event.(*github.PushEvent)
	if !ok {
		// not what we're looking for
		return
	}

	scope.Infof("Received push event in repo %s", pp.GetRepo().GetFullName())

	if pp.GetRepo().GetOwner().GetLogin() != m.repo.OrgLogin || pp.GetRepo().GetName() != m.repo.RepoName {
		// not the org/repo we care about
		scope.Info("Not the desired repo, ignoring")
		return
	}

	// TODO: ensure the right branch (m.repo.Branch) is being affected, not sure how to get the branch info sadly

	for _, commit := range pp.Commits {
		for _, s := range commit.Modified {
			if strings.HasPrefix(s, m.path) {
				scope.Infof("Detected modification to file %s in repo %s", s, m.repo)
				m.notify()
				return
			}
		}

		for _, s := range commit.Added {
			if strings.HasPrefix(s, m.path) {
				scope.Infof("Detected addition of file %s in repo %s", s, m.repo)
				m.notify()
				return
			}
		}

		for _, s := range commit.Removed {
			if strings.HasPrefix(s, m.path) {
				scope.Infof("Detected removal of file %s in repo %s", s, m.repo)
				m.notify()
				return
			}
		}
	}

	scope.Infof("No changes detected to the monitored path '%s'", m.path)
}
