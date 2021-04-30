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
	"regexp"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// Attaches labels to issues and PRs based on regex matches on the title, body, etc.
type Labeler struct {
	cache             *cache.Cache
	gc                *gh.ThrottledClient
	singleLineRegexes map[string]*regexp.Regexp
	multiLineRegexes  map[string]*regexp.Regexp
	reg               *config.Registry
}

var scope = log.RegisterScope("labeler", "Issue and PR auto-labeler", 0)

func NewLabeler(gc *gh.ThrottledClient, cache *cache.Cache, reg *config.Registry) (githubwebhook.Filter, error) {
	l := &Labeler{
		cache:             cache,
		gc:                gc,
		singleLineRegexes: make(map[string]*regexp.Regexp),
		multiLineRegexes:  make(map[string]*regexp.Regexp),
		reg:               reg,
	}

	for _, r := range reg.Records(recordType, "*") {
		al := r.(*autoLabelRecord)
		if err := l.processAutoLabelRegexes(al); err != nil {
			return nil, err
		}
	}

	return l, nil
}

// Precompile all the regexes
func (l *Labeler) processAutoLabelRegexes(al *autoLabelRecord) error {
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

	for _, expr := range al.PresentLabels {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		l.singleLineRegexes[expr] = r
	}

	return nil
}

// process an event arriving from GitHub
func (l *Labeler) Handle(context context.Context, event interface{}) {
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
	autoLabels := l.reg.Records(recordType, repo)
	if len(autoLabels) == 0 {
		scope.Infof("Ignoring event for issue/PR %d from repo %s since there are no matching auto labels", number, repo)
		return
	}

	scope.Infof("Processing event for issue/PR %d from repo %s, %s", number, repo, action)

	if issue != nil {
		l.processIssue(context, issue, autoLabels)
	} else {
		l.processPullRequest(context, pr, autoLabels)
	}
}

func (l *Labeler) processIssue(context context.Context, issue *storage.Issue, als []config.Record) {
	// get all the issue's labels
	var labels []*storage.Label
	for _, labelName := range issue.Labels {
		label, err := l.cache.ReadLabel(context, issue.OrgLogin, issue.RepoName, labelName)
		if err != nil {
			scope.Errorf("Unable to get labels for issue/pr %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
			return
		} else if label != nil {
			labels = append(labels, label)
		}
	}

	// find any matching global auto labels
	var toApply []string
	var toRemove []string

	// find any matching auto labels
	for _, r := range als {
		al := r.(*autoLabelRecord)

		if l.matchAutoLabel(al, issue.Title, issue.Body, labels) {
			toApply = append(toApply, al.LabelsToApply...)
			toRemove = append(toRemove, al.LabelsToRemove...)
		}
	}

	if len(toApply) > 0 {
		if _, _, err := l.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.AddLabelsToIssue(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), toApply)
		}); err != nil {
			scope.Errorf("Unable to set labels on issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
			return
		}
	}

	scope.Infof("Applied %d label(s) to issue/PR %d from repo %s/%s", len(toApply), issue.IssueNumber, issue.OrgLogin, issue.RepoName)

	if len(toRemove) > 0 {
		for _, label := range toRemove {
			if _, err := l.gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
				return client.Issues.RemoveLabelForIssue(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), label)
			}); err != nil {
				scope.Errorf("Unable to remove labels on issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
				return
			}
		}
	}

	scope.Infof("Removed %d label(s) from issue/PR %d from repo %s/%s", len(toRemove), issue.IssueNumber, issue.OrgLogin, issue.RepoName)
}

func (l *Labeler) processPullRequest(context context.Context, pr *storage.PullRequest, als []config.Record) {
	// get all the pr's labels
	var labels []*storage.Label
	for _, labelName := range pr.Labels {
		label, err := l.cache.ReadLabel(context, pr.OrgLogin, pr.RepoName, labelName)
		if err != nil {
			scope.Errorf("Unable to get labels for pr %d in repo %s/%s: %v", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
			return
		} else if label != nil {
			labels = append(labels, label)
		}
	}

	// find any matching auto labels
	var toApply []string
	for _, r := range als {
		al := r.(*autoLabelRecord)

		if l.matchAutoLabel(al, pr.Title, pr.Body, labels) {
			toApply = append(toApply, al.LabelsToApply...)
		}
	}

	if len(toApply) > 0 {
		if _, _, err := l.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.AddLabelsToIssue(context, pr.OrgLogin, pr.RepoName, int(pr.PullRequestNumber), toApply)
		}); err != nil {
			scope.Errorf("Unable to set labels on PR %d in repo %s/%s: %v", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
			return
		}
	}

	scope.Infof("Applied %d label(s) to PR %d from repo %s/%s", len(toApply), pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
}

func (l *Labeler) matchAutoLabel(al *autoLabelRecord, title string, body string, labels []*storage.Label) bool {
	// if both the title and body don't match, we're done
	if !l.titleMatch(al, title) && !l.bodyMatch(al, body) {
		return false
	}

	// if any of the 'must be absent' labels match, we bail
	for _, expr := range al.AbsentLabels {
		r := l.singleLineRegexes[expr]
		for _, label := range labels {
			if r.MatchString(label.LabelName) {
				return false
			}
		}
	}

	// if any of the 'must be present' labels don't match, we bail
	for _, expr := range al.PresentLabels {
		r := l.singleLineRegexes[expr]
		found := false
		for _, label := range labels {
			if r.MatchString(label.LabelName) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	// we found a suitable candidate
	return true
}

func (l *Labeler) titleMatch(al *autoLabelRecord, title string) bool {
	for _, expr := range al.MatchTitle {
		r := l.singleLineRegexes[expr]
		if r.MatchString(title) {
			return true
		}
	}

	return false
}

func (l *Labeler) bodyMatch(al *autoLabelRecord, body string) bool {
	for _, expr := range al.MatchBody {
		r := l.multiLineRegexes[expr]
		if r.MatchString(body) {
			return true
		}
	}

	return false
}
