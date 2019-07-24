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

package unstaler

import (
	"context"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/pkg/log"
)

// Removes the staleness label and comments from issues and prs when activity is detected
type Unstaler struct {
	gc    *gh.ThrottledClient
	repos map[string]bool
}

const stalenessLabel = "stale"
const stalenessSignature = "\n\n_Courtesy of your friendly freshness tracker_."

var scope = log.RegisterScope("unstale", "Removes the stale label when activity is detected on a PR or issue", 0)

func NewUnstaler(gc *gh.ThrottledClient, orgs []config.Org) (filters.Filter, error) {
	u := &Unstaler{
		gc:    gc,
		repos: make(map[string]bool),
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			u.repos[org.Name+"/"+repo.Name] = true
		}
	}

	return u, nil
}

// process an event arriving from GitHub
func (u *Unstaler) Handle(context context.Context, event interface{}) {
	action := ""
	orgLogin := ""
	repoName := ""
	number := 0

	switch p := event.(type) {
	case *github.IssuesEvent:
		scope.Infof("Received IssuesEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetIssue().GetNumber()

	case *github.IssueCommentEvent:
		scope.Infof("Received IssueCommentEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetIssue().GetNumber()

	case *github.PullRequestEvent:
		scope.Infof("Received PullRequestEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetPullRequest().GetNumber()

	case *github.PullRequestReviewEvent:
		scope.Infof("Received PullRequestReviewEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetPullRequest().GetNumber()

	case *github.PullRequestReviewCommentEvent:
		scope.Infof("Received PullRequestReviewCommentEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetPullRequest().GetNumber()

	default:
		// not what we're looking for
		scope.Debugf("Unknown event received: %T %+v", p, p)
		return
	}

	// see if the event is in a repo we're monitoring
	if !u.repos[repoName+"/"+orgLogin] {
		scope.Infof("Ignoring event for issue/PR %d from repo %s/%s since it's not in a monitored repo", number, orgLogin, repoName)
		return
	}

	scope.Infof("Processing event for issue/PR %d from repo %s/%s, %s", number, repoName, orgLogin, action)

	if _, err := u.gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
		return client.Issues.RemoveLabelForIssue(context, orgLogin, repoName, number, stalenessLabel)
	}); err != nil {
		scope.Errorf("Unable to remove staleness label on issue/PR %d in repo %s/%s: %v", number, orgLogin, repoName, err)
		return
	}

	if err := gh.RemoveBotComment(context, u.gc, orgLogin, repoName, number, stalenessSignature); err != nil {
		scope.Error(err.Error())
	}
}
