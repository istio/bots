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
	gc   *gh.ThrottledClient
	args *config.Args
}

var scope = log.RegisterScope("labelmgr", "The GitHub label manager", 0)

func New(gc *gh.ThrottledClient, args *config.Args) *LabelMgr {
	return &LabelMgr{
		gc:   gc,
		args: args,
	}
}

func (lm *LabelMgr) MakeConfiguredLabels(context context.Context) error {
	for _, org := range lm.args.Orgs {
		for _, repo := range org.Repos {
			// global
			for _, label := range lm.args.LabelsToCreate {
				err := lm.makeLabel(context, org.Name, repo.Name, label)
				if err != nil {
					return fmt.Errorf("unable to create label %s in repo %s/%s: %v", label.Name, org.Name, repo.Name, err)
				}
			}

			// org-level
			for _, label := range org.LabelsToCreate {
				err := lm.makeLabel(context, org.Name, repo.Name, label)
				if err != nil {
					return fmt.Errorf("unable to create label %s in repo %s/%s: %v", label.Name, org.Name, repo.Name, err)
				}
			}

			// repo-level
			for _, label := range repo.LabelsToCreate {
				err := lm.makeLabel(context, org.Name, repo.Name, label)
				if err != nil {
					return fmt.Errorf("unable to create label %s in repo %s/%s: %v", label.Name, org.Name, repo.Name, err)
				}
			}
		}
	}

	return nil
}

func (lm *LabelMgr) makeLabel(context context.Context, orgLogin string, repoName string, label config.Label) error {
	_, _, err := lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.CreateLabel(context, orgLogin, repoName, &github.Label{
			Name:        &label.Name,
			Color:       &label.Color,
			Description: &label.Description,
		})
	})

	if err == nil {
		scope.Infof("Created label %s in repo %s/%s", label.Name, orgLogin, repoName)
		return nil
	}

	_, _, err = lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.EditLabel(context, orgLogin, repoName, label.Name, &github.Label{
			Name:        &label.Name,
			Color:       &label.Color,
			Description: &label.Description,
		})
	})

	if err == nil {
		scope.Infof("Updated label %s in repo %s/%s", label.Name, orgLogin, repoName)
		return nil
	}

	return err
}
