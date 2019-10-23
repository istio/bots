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

package boilerplatecleaner

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook/filters"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// Removes redundant boilerplate content from PRs and issues
type Cleaner struct {
	cache            *cache.Cache
	gc               *gh.ThrottledClient
	orgs             []config.Org
	boilerplates     []config.Boilerplate
	multiLineRegexes map[string]*regexp.Regexp
	repos            map[string][]config.Boilerplate // index is org/repo, value is org-level boilerplate
}

var scope = log.RegisterScope("cleaner", "Issue and PR boilerplate cleaner", 0)

func NewCleaner(gc *gh.ThrottledClient, cache *cache.Cache, orgs []config.Org, boilerplates []config.Boilerplate) (filters.Filter, error) {
	l := &Cleaner{
		cache:            cache,
		gc:               gc,
		orgs:             orgs,
		boilerplates:     boilerplates,
		multiLineRegexes: make(map[string]*regexp.Regexp),
		repos:            make(map[string][]config.Boilerplate),
	}

	for _, al := range boilerplates {
		if err := l.processBoilerplateRegexes(al); err != nil {
			return nil, err
		}
	}

	for _, org := range orgs {
		for _, al := range org.BoilerplatesToClean {
			if err := l.processBoilerplateRegexes(al); err != nil {
				return nil, err
			}
		}
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			l.repos[org.Name+"/"+repo.Name] = org.BoilerplatesToClean
		}
	}

	return l, nil
}

// Precompile all the regexes
func (l *Cleaner) processBoilerplateRegexes(b config.Boilerplate) error {
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
	boilerplates, ok := l.repos[repo]
	if !ok {
		scope.Infof("Ignoring event for issue/PR %d from repo %s since it's not in a monitored repo", number, repo)
		return
	}

	scope.Infof("Processing event for issue/PR %d from repo %s, %s", number, repo, action)

	if issue != nil {
		l.processIssue(context, issue, boilerplates)
	} else {
		l.processPullRequest(context, pr, boilerplates)
	}
}

func (l *Cleaner) processIssue(context context.Context, issue *storage.Issue, orgBoilerplates []config.Boilerplate) {
	original := strings.ReplaceAll(issue.Body, "\r\n", "\n")

	if !strings.HasSuffix(original, "\n") {
		// makes the regex matches more reliable
		original += "\n"
	}

	body := original

	for _, b := range orgBoilerplates {
		r := l.multiLineRegexes[b.Regex]
		body = r.ReplaceAllString(body, b.Replacement)
	}

	for _, b := range l.boilerplates {
		r := l.multiLineRegexes[b.Regex]
		body = r.ReplaceAllString(body, b.Replacement)
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
		scope.Infof("Removed boilerplate from issue %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	} else {
		scope.Infof("No boilerplate to remove from issue %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	}
}

func (l *Cleaner) processPullRequest(context context.Context, pr *storage.PullRequest, orgBoilerplates []config.Boilerplate) {
	original := strings.ReplaceAll(pr.Body, "\r\n", "\n")

	if !strings.HasSuffix(original, "\n") {
		// makes the regex matches more reliable
		original += "\n"
	}

	body := original

	for _, b := range orgBoilerplates {
		r := l.multiLineRegexes[b.Regex]
		body = r.ReplaceAllString(body, b.Replacement)
	}

	for _, b := range l.boilerplates {
		r := l.multiLineRegexes[b.Regex]
		body = r.ReplaceAllString(body, b.Replacement)
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
		scope.Infof("Removed boilerplate from PR %d in repo %s/%s", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
	} else {
		scope.Infof("No boilerplate to remove from PR %d in repo %s/%s", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
	}
}
