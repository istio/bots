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

package milestonemgr

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/pkg/log"
)

// MilestoneMgr creates milestones in GitHub repos
type MilestoneMgr struct {
	gc   *gh.ThrottledClient
	args *config.Args
}

var scope = log.RegisterScope("milestonemgr", "The GitHub milestone manager", 0)

func New(gc *gh.ThrottledClient, args *config.Args) *MilestoneMgr {
	return &MilestoneMgr{
		gc:   gc,
		args: args,
	}
}

func (mm *MilestoneMgr) MakeConfiguredMilestones(context context.Context) error {
	for _, org := range mm.args.Orgs {
		for _, repo := range org.Repos {
			// global
			for _, milestone := range mm.args.MilestonesToCreate {
				err := mm.makeMilestone(context, org.Name, repo.Name, milestone)
				if err != nil {
					return fmt.Errorf("unable to create milestone %s in repo %s/%s: %v", milestone.Name, org.Name, repo.Name, err)
				}
			}

			// org-level
			for _, milestone := range org.MilestonesToCreate {
				err := mm.makeMilestone(context, org.Name, repo.Name, milestone)
				if err != nil {
					return fmt.Errorf("unable to create milestone %s in repo %s/%s: %v", milestone.Name, org.Name, repo.Name, err)
				}
			}

			// repo-level
			for _, milestone := range repo.MilestonesToCreate {
				err := mm.makeMilestone(context, org.Name, repo.Name, milestone)
				if err != nil {
					return fmt.Errorf("unable to create milestone %s in repo %s/%s: %v", milestone.Name, org.Name, repo.Name, err)
				}
			}
		}
	}

	return nil
}

var open = "open"
var closed = "closed"

func (mm *MilestoneMgr) makeMilestone(context context.Context, orgLogin string, repoName string, milestone config.Milestone) error {
	ms := &github.Milestone{
		State:       &open,
		Title:       &milestone.Name,
		Description: &milestone.Description,
	}

	if milestone.Closed {
		ms.State = &closed
	}

	var zeroTime time.Time
	if milestone.DueDate != zeroTime {
		ms.DueOn = &milestone.DueDate
	}

	_, _, err := mm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.CreateMilestone(context, orgLogin, repoName, ms)
	})

	if err == nil {
		scope.Infof("Created milestone %s in repo %s/%s", milestone.Name, orgLogin, repoName)
		return nil
	}

	num, err := findMilestone(context, mm.gc, orgLogin, repoName, milestone.Name)
	if num < 0 {
		if err == nil {
			return fmt.Errorf("unable to create or edit milestone %s in repo %s/%s", milestone.Name, orgLogin, repoName)
		}
		return err
	}

	_, _, err = mm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.EditMilestone(context, orgLogin, repoName, num, ms)
	})

	if err == nil {
		scope.Infof("Updated milestone %s in repo %s/%s", milestone.Name, orgLogin, repoName)
		return nil
	}

	return err
}

// findMilestone looks for a milestone with the given name in a repo
func findMilestone(context context.Context, gc *gh.ThrottledClient, orgLogin string, repoName string, name string) (int, error) {
	opt := &github.MilestoneListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		milestones, resp, err := gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListMilestones(context, orgLogin, repoName, opt)
		})

		if err != nil {
			return -1, fmt.Errorf("unable to list milestones for repo %s/%s: %v", orgLogin, repoName, err)
		}

		for _, milestone := range milestones.([]*github.Milestone) {
			title := milestone.GetTitle()
			if title == name {
				return milestone.GetNumber(), nil
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return -1, nil
}
