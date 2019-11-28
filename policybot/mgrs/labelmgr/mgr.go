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

package labelmgr

import (
	"context"
	"fmt"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/pkg/log"
)

// LabelMgr creates labels in GitHub repos
type LabelMgr struct {
	gc  *gh.ThrottledClient
	reg *config.Registry
}

var scope = log.RegisterScope("labelmgr", "The GitHub label manager", 0)

func New(gc *gh.ThrottledClient, reg *config.Registry) *LabelMgr {
	return &LabelMgr{
		gc:  gc,
		reg: reg,
	}
}

func (lm *LabelMgr) MakeConfiguredLabels(context context.Context, dryRun bool) error {
	for _, repo := range lm.reg.Repos() {
		for _, r := range lm.reg.Records(recordType, repo.OrgAndRepo) {
			label := r.(*labelRecord)

			if dryRun {
				scope.Infof("Would have created or updated label %s in repo %s", label.Name, repo)
				continue
			}

			if err := lm.makeLabel(context, repo, label); err != nil {
				return fmt.Errorf("unable to create label %s in repo %s: %v", label.Name, repo, err)
			}
		}
	}

	return nil
}

func (lm *LabelMgr) makeLabel(context context.Context, repo gh.RepoDesc, label *labelRecord) error {
	_, _, err := lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.CreateLabel(context, repo.OrgLogin, repo.RepoName, &github.Label{
			Name:        &label.Name,
			Color:       &label.Color,
			Description: &label.Description,
		})
	})

	if err == nil {
		scope.Infof("Created label %s in repo %s", label.Name, repo)
		return nil
	}

	_, _, err = lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.EditLabel(context, repo.OrgLogin, repo.RepoName, label.Name, &github.Label{
			Name:        &label.Name,
			Color:       &label.Color,
			Description: &label.Description,
		})
	})

	if err == nil {
		scope.Infof("Updated label %s in repo %s", label.Name, repo)
		return nil
	}

	return err
}
