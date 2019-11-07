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

package lifecycler

import (
	"context"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/bots/policybot/mgrs/lifecyclemgr"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

type Lifecycler struct {
	gc         *gh.ThrottledClient
	repos      map[string]bool
	lifecycler *lifecyclemgr.LifecycleMgr
	cache      *cache.Cache
}

var scope = log.RegisterScope("lifecycler", "Handles lifecycle events for PRs or issues", 0)

func New(gc *gh.ThrottledClient, orgs []config.Org, lifecycler *lifecyclemgr.LifecycleMgr, cache *cache.Cache) githubwebhook.Filter {
	u := &Lifecycler{
		gc:         gc,
		repos:      make(map[string]bool),
		lifecycler: lifecycler,
		cache:      cache,
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			u.repos[org.Name+"/"+repo.Name] = true
		}
	}

	return u
}

// process an event arriving from GitHub
func (l *Lifecycler) Handle(context context.Context, event interface{}) {
	action := ""
	orgLogin := ""
	repoName := ""
	number := 0

	var issue *storage.Issue
	var pr *storage.PullRequest
	var sender *github.User

	switch p := event.(type) {
	case *github.IssuesEvent:
		scope.Infof("Received IssuesEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		repoName = p.GetRepo().GetFullName()
		number = p.GetIssue().GetNumber()
		issue = gh.ConvertIssue(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetIssue())

	case *github.IssueCommentEvent:
		scope.Infof("Received IssueCommentEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetIssue().GetNumber()
		issue = gh.ConvertIssue(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetIssue())

	case *github.PullRequestEvent:
		scope.Infof("Received PullRequestEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		repoName = p.GetRepo().GetFullName()
		number = p.GetPullRequest().GetNumber()
		pr = gh.ConvertPullRequest(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetPullRequest(),
			nil)

	case *github.PullRequestReviewEvent:
		scope.Infof("Received PullRequestReviewEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetPullRequest().GetNumber()
		pr = gh.ConvertPullRequest(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetPullRequest(),
			nil)

	case *github.PullRequestReviewCommentEvent:
		scope.Infof("Received PullRequestReviewCommentEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		orgLogin = p.GetRepo().GetOwner().GetLogin()
		repoName = p.GetRepo().GetName()
		number = p.GetPullRequest().GetNumber()
		pr = gh.ConvertPullRequest(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetPullRequest(),
			nil)

	default:
		// not what we're looking for
		scope.Debugf("Unknown event received: %T %+v", p, p)
		return
	}

	// see if the event is in a repo we're monitoring
	if !l.repos[orgLogin+"/"+repoName] {
		scope.Infof("Ignoring event for issue/PR %d from repo %s/%s since it's not in a monitored repo", number, orgLogin, repoName)
		return
	}

	if issue != nil {
		if sender != nil {
			user := sender.GetLogin()

			member, err := l.cache.ReadMember(context, issue.OrgLogin, user)
			if err != nil {
				scope.Errorf("Unable to read member information about %s from org %s: %v", user, issue.OrgLogin, err)
				return
			}

			if member == nil {
				// if event is not from a member, it won't affect the lifecycle so return promptly
				scope.Infof("Ignoring event for issue/PR %d from repo %s/%s since it wasn't caused by an org member", number, orgLogin, repoName)
				return
			}
		}

		scope.Infof("Processing event for issue %d from repo %s/%s, %s, labels %v", number, repoName, orgLogin, action, issue.Labels)

		if err := l.lifecycler.ManageIssue(context, issue); err != nil {
			scope.Errorf("%v", err)
		}
	} else if pr != nil {
		// TODO
		_ = pr
	}
}
