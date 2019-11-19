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
	"fmt"

	"istio.io/bots/policybot/pkg/zh"

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
	zc         *zh.ThrottledClient
	repos      map[string]bool
	lifecycler *lifecyclemgr.LifecycleMgr
	cache      *cache.Cache
}

var scope = log.RegisterScope("lifecycler", "Handles lifecycle events for PRs or issues", 0)

func New(gc *gh.ThrottledClient, zc *zh.ThrottledClient, orgs []config.Org, lifecycler *lifecyclemgr.LifecycleMgr, cache *cache.Cache) githubwebhook.Filter {
	u := &Lifecycler{
		gc:         gc,
		zc:         zc,
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
	repo := ""
	var repoNumber int64
	number := 0

	var issue *storage.Issue
	var pr *storage.PullRequest
	var sender *github.User

	switch p := event.(type) {
	case *github.IssuesEvent:
		scope.Infof("Received IssuesEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		repo = p.GetRepo().GetFullName()
		repoNumber = p.GetRepo().GetID()
		number = p.GetIssue().GetNumber()
		issue = gh.ConvertIssue(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetIssue())

	case *github.IssueCommentEvent:
		scope.Infof("Received IssueCommentEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		repo = p.GetRepo().GetFullName()
		repoNumber = p.GetRepo().GetID()
		number = p.GetIssue().GetNumber()
		issue = gh.ConvertIssue(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetIssue())

	case *github.PullRequestEvent:
		scope.Infof("Received PullRequestEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		sender = p.GetSender()
		action = p.GetAction()
		repo = p.GetRepo().GetFullName()
		repoNumber = p.GetRepo().GetID()
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
		repo = p.GetRepo().GetFullName()
		repoNumber = p.GetRepo().GetID()
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
		repo = p.GetRepo().GetFullName()
		repoNumber = p.GetRepo().GetID()
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
	if !l.repos[repo] {
		scope.Infof("Ignoring event for issue/PR %d from repo %s since it's not in a monitored repo", number, repo)
		return
	}

	if pr != nil {
		// we turn PR objects into Issue objects, since for the sake of lifecycle management they're the same
		issue = &storage.Issue{
			OrgLogin:    pr.OrgLogin,
			RepoName:    pr.RepoName,
			IssueNumber: pr.PullRequestNumber,
			Title:       pr.Title,
			Body:        pr.Body,
			Labels:      pr.Labels,
			CreatedAt:   pr.CreatedAt,
			UpdatedAt:   pr.UpdatedAt,
			ClosedAt:    pr.ClosedAt,
			State:       pr.State,
			Author:      pr.Author,
			Assignees:   pr.Assignees,
		}
	}

	if sender != nil {
		user := sender.GetLogin()

		member, err := l.cache.ReadMember(context, issue.OrgLogin, user)
		if err != nil {
			scope.Errorf("Unable to read member information about %s from org %s: %v", user, issue.OrgLogin, err)
			return
		}

		if member == nil {
			// if event is not from a member, it won't affect the lifecycle so return promptly
			scope.Infof("Ignoring event for issue/PR %d from repo %s since it wasn't caused by an org member", number, repo)
			return
		}
	}

	scope.Infof("Processing event for issue/PR %d from repo %s, %s, labels %v", number, repo, action, issue.Labels)

	if err := l.fetchPipeline(context, issue.OrgLogin, issue.RepoName, int(repoNumber), number); err != nil {
		scope.Errorf("%v", err)
	}

	if err := l.lifecycler.ManageIssue(context, issue); err != nil {
		scope.Errorf("%v", err)
	}
}

func (l *Lifecycler) fetchPipeline(context context.Context, orgLogin string, repoName string, repoNumber int, issueNumber int) error {
	// do we have pipeline info in our store?
	pipeline, err := l.cache.ReadIssuePipeline(context, orgLogin, repoName, issueNumber)
	if err != nil {
		return fmt.Errorf("could not get issue pipeline data for issue/PR %d in repo %s/%s: %v", issueNumber, orgLogin, repoName, err)
	}

	if pipeline != nil && pipeline.Pipeline != "" && pipeline.Pipeline != "New Issue" {
		// already have issue data
		return nil
	}

	// we don't have any useful local info, query ZenHub directly
	issueData, err := l.zc.ThrottledCall(func(client *zh.Client) (interface{}, error) {
		return client.GetIssueData(repoNumber, issueNumber)
	})

	if err != nil {
		if err == zh.ErrNotFound {
			// not found, so nothing to do...
			return nil
		}

		return fmt.Errorf("unable to get issue data from ZenHub for issue/PR %d in repo %s/%s: %v", issueNumber, orgLogin, repoName, err)
	}

	// now store the pipeline info in our store
	pipelines := []*storage.IssuePipeline{
		{
			OrgLogin:    orgLogin,
			RepoName:    repoName,
			IssueNumber: int64(issueNumber),
			Pipeline:    issueData.(*zh.IssueData).Pipeline.Name,
		},
	}

	err = l.cache.WriteIssuePipelines(context, pipelines)
	if err != nil {
		return fmt.Errorf("unable to update issue pipeline for issue/PR %d in repo %s/%s: %v", issueNumber, orgLogin, repoName, err)
	}

	return nil
}
