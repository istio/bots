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

package cleaner

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"istio.io/bots/policybot/handlers/githubwebhook"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/log"
)

// Removes redundant boilerplate content from PRs and issues
type Cleaner struct {
	gc               *gh.ThrottledClient
	multiLineRegexes map[string]*regexp.Regexp
	reg              *config.Registry
}

var scope = log.RegisterScope("cleaner", "Issue and PR boilerplate cleaner", 0)

func New(gc *gh.ThrottledClient, reg *config.Registry) (githubwebhook.Filter, error) {
	l := &Cleaner{
		gc:               gc,
		multiLineRegexes: make(map[string]*regexp.Regexp),
		reg:              reg,
	}

	for _, r := range reg.Records(recordType, "*") {
		b := r.(*boilerplateRecord)
		if err := l.processBoilerplateRegexes(b); err != nil {
			return nil, err
		}
	}

	return l, nil
}

// Precompile all the regexes
func (l *Cleaner) processBoilerplateRegexes(b *boilerplateRecord) error {
	r, err := regexp.Compile("(?mis)" + b.Regex)
	if err != nil {
		return fmt.Errorf("invalid regular expression %s: %v", b.Regex, err)
	}
	l.multiLineRegexes[b.Regex] = r

	return nil
}

// process an event arriving from GitHub
func (l *Cleaner) Handle(context context.Context, event interface{}) {
	action := ""
	repo := ""
	number := 0
	var issue *storage.Issue
	var pr *storage.PullRequest

	switch p := event.(type) {
	case *github.IssuesEvent:
		scope.Infof("Received IssuesEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		action = p.GetAction()
		repo = p.GetRepo().GetFullName()
		number = p.GetIssue().GetNumber()
		issue = gh.ConvertIssue(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetIssue())

	case *github.PullRequestEvent:
		scope.Infof("Received PullRequestEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		action = p.GetAction()
		repo = p.GetRepo().GetFullName()
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

	if action != "opened" {
		// not what we care about
		scope.Infof("Ignoring event for issue/PR %d from repo %s since it doesn't have a supported action: %s", number, repo, action)
		return
	}

	// see if the event is in a repo we're monitoring
	boilerplates := l.reg.Records(recordType, repo)
	if len(boilerplates) == 0 {
		scope.Infof("Ignoring event for issue/PR %d from repo %s since there are no matching boilerplates", number, repo)
		return
	}

	scope.Infof("Processing event for issue/PR %d from repo %s, %s", number, repo, action)

	if issue != nil {
		l.processIssue(context, issue, boilerplates)
	} else {
		l.processPullRequest(context, pr, boilerplates)
	}
}

func (l *Cleaner) processIssue(context context.Context, issue *storage.Issue, boilerplates []config.Record) {
	original := strings.ReplaceAll(issue.Body, "\r\n", "\n")

	if !strings.HasSuffix(original, "\n") {
		// makes the regex matches more reliable
		original += "\n"
	}

	body := original

	for _, rec := range boilerplates {
		b := rec.(*boilerplateRecord)
		r := l.multiLineRegexes[b.Regex]

		oldBody := body
		body = r.ReplaceAllString(body, b.Replacement)
		if oldBody != body {
			scope.Infof("Removed the `%s` boilerplate from issue %d in repo %s/%s", b.Name, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
		}
	}

	if body != original {
		body = strings.TrimRight(body, "\n")

		ir := &github.IssueRequest{Body: &body}
		if _, _, err := l.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.Edit(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), ir)
		}); err != nil {
			scope.Errorf("Unable to remove boilerplate from issue %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
			return
		}
	} else {
		scope.Infof("No boilerplate to remove from issue %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	}
}

func (l *Cleaner) processPullRequest(context context.Context, pr *storage.PullRequest, boilerplates []config.Record) {
	original := strings.ReplaceAll(pr.Body, "\r\n", "\n")

	if !strings.HasSuffix(original, "\n") {
		// makes the regex matches more reliable
		original += "\n"
	}

	body := original

	for _, rec := range boilerplates {
		b := rec.(*boilerplateRecord)
		r := l.multiLineRegexes[b.Regex]

		oldBody := body
		body = r.ReplaceAllString(body, b.Replacement)
		if oldBody != body {
			scope.Infof("Removed the `%s` boilerplate from PR %d in repo %s/%s", b.Name, pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
		}
	}

	if body != original {
		body = strings.TrimRight(body, "\n")

		ir := &github.IssueRequest{Body: &body}
		if _, _, err := l.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.Edit(context, pr.OrgLogin, pr.RepoName, int(pr.PullRequestNumber), ir)
		}); err != nil {
			scope.Errorf("Unable to remove boilerplate from PR %d in repo %s/%s: %v", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
			return
		}
	} else {
		scope.Infof("No boilerplate to remove from PR %d in repo %s/%s", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
	}
}
