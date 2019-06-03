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

package labeler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"istio.io/pkg/log"

	webhook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/util"
)

// Generates nagging messages in PRs based on regex matches on the title, body, and affected files
type Labeler struct {
	ctx               context.Context
	ghs               *gh.GitHubState
	ght               *util.GitHubThrottle
	orgs              []config.Org
	autoLabels        []config.AutoLabel
	singleLineRegexes map[string]*regexp.Regexp
	multiLineRegexes  map[string]*regexp.Regexp
	repos             map[string][]config.AutoLabel // index is org/repo, value is org-level auto-labels
}

var scope = log.RegisterScope("labeler", "Issue and PR auto-labeler", 0)

func NewLabeler(ctx context.Context, ght *util.GitHubThrottle, ghs *gh.GitHubState, orgs []config.Org, autoLabels []config.AutoLabel) (*Labeler, error) {
	l := &Labeler{
		ctx:               ctx,
		ghs:               ghs,
		ght:               ght,
		orgs:              orgs,
		autoLabels:        autoLabels,
		singleLineRegexes: make(map[string]*regexp.Regexp),
		multiLineRegexes:  make(map[string]*regexp.Regexp),
		repos:             make(map[string][]config.AutoLabel),
	}

	for _, al := range autoLabels {
		if err := l.processAutoLabelRegexes(al); err != nil {
			return nil, err
		}
	}

	for _, org := range orgs {
		for _, al := range org.AutoLabels {
			if err := l.processAutoLabelRegexes(al); err != nil {
				return nil, err
			}
		}
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			l.repos[org.Name+"/"+repo.Name] = org.AutoLabels
		}
	}

	return l, nil
}

// Precompile all the regexes
func (l *Labeler) processAutoLabelRegexes(al config.AutoLabel) error {
	for _, expr := range al.MatchTitle {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		l.singleLineRegexes[expr] = r
	}

	for _, expr := range al.MatchBody {
		r, err := regexp.Compile("(?mi)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		l.multiLineRegexes[expr] = r
	}

	for _, expr := range al.AbsentLabels {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		l.singleLineRegexes[expr] = r
	}

	return nil
}

func (l *Labeler) Events() []webhook.Event {
	return []webhook.Event{
		webhook.IssuesEvent,
		webhook.PullRequestEvent,
	}
}

// process an event arriving from GitHub
func (l *Labeler) Handle(_ http.ResponseWriter, githubObject interface{}) {
	action := ""
	repo := ""
	number := 0
	var issue *storage.Issue

	ip, ok := githubObject.(webhook.IssuesPayload)
	if ok {
		action = ip.Action
		repo = ip.Repository.FullName
		number = int(ip.Issue.Number)
		issue = gh.IssueFromHook(&ip)
	}

	prp, ok := githubObject.(webhook.PullRequestPayload)
	if ok {
		action = prp.Action
		repo = prp.Repository.FullName
		number = int(prp.PullRequest.Number)
		_, issue = gh.PullRequestFromHook(&prp)
	}

	if action != "opened" && action != "review_requested" {
		// not what we care about
		return
	}

	// see if the event is in a repo we're monitoring
	autoLabels, ok := l.repos[repo]
	if !ok {
		scope.Infof("Ignoring event %d from repo %s since it's not in a monitored repo", number, repo)
		return
	}

	scope.Infof("Processing event %d from repo %s", number, repo)

	split := strings.Split(repo, "/")
	l.processIssue(issue, repo, split[0], split[1], autoLabels)
}

func (l *Labeler) processIssue(issue *storage.Issue, fullRepoName, orgName, repoName string, orgALs []config.AutoLabel) {
	// get all the issue's labels
	var labels []*storage.Label
	for _, labelID := range issue.LabelIDs {
		label, err := l.ghs.ReadLabel(issue.OrgID, issue.RepoID, labelID)
		if err != nil {
			scope.Errorf("Unable to get labels for event %d in repo %s: %v", issue.Number, fullRepoName, err)
			return
		}
		labels = append(labels, label)
	}

	// find any matching global auto labels
	var toApply []string
	for _, al := range l.autoLabels {
		if l.matchAutoLabel(al, issue.Title, issue.Body, labels) {
			toApply = append(toApply, al.Labels...)
		}
	}

	// find any matching org-level auto labels
	for _, al := range orgALs {
		if l.matchAutoLabel(al, issue.Title, issue.Body, labels) {
			toApply = append(toApply, al.Labels...)
		}
	}

	if len(toApply) > 0 {
		if _, _, err := l.ght.Get().Issues.AddLabelsToIssue(l.ctx, orgName, repoName, int(issue.Number), toApply); err != nil {
			scope.Errorf("Unable to set labels on event %d in repo %s: %v", issue.Number, fullRepoName, err)
			return
		}
	}

	scope.Infof("Applied %d label(s) to event %d from repo %s", len(toApply), issue.Number, fullRepoName)
}

func (l *Labeler) matchAutoLabel(al config.AutoLabel, title string, body string, labels []*storage.Label) bool {
	// if the title and body don't match, we're done
	if !l.titleMatch(al, title) && !l.bodyMatch(al, body) {
		return false
	}

	// if any labels match, we're done
	for _, label := range labels {
		if l.labelMatch(al, label.Name) {
			return false
		}
	}

	return true
}

func (l *Labeler) titleMatch(al config.AutoLabel, title string) bool {
	for _, expr := range al.MatchTitle {
		r := l.singleLineRegexes[expr]
		if r.MatchString(title) {
			return true
		}
	}

	return false
}

func (l *Labeler) bodyMatch(al config.AutoLabel, body string) bool {
	for _, expr := range al.MatchBody {
		r := l.multiLineRegexes[expr]
		if r.MatchString(body) {
			return true
		}
	}

	return false
}

func (l *Labeler) labelMatch(al config.AutoLabel, label string) bool {
	for _, expr := range al.AbsentLabels {
		r := l.singleLineRegexes[expr]
		if r.MatchString(label) {
			return true
		}
	}

	return false
}
