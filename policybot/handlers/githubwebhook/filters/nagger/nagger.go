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

package nagger

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

// Generates nagging messages in PRs based on regex matches on the title, body, and affected files
type Nagger struct {
	cache             *cache.Cache
	gc                *gh.ThrottledClient
	orgs              []config.Org
	nags              []config.Nag
	multiLineRegexes  map[string]*regexp.Regexp
	singleLineRegexes map[string]*regexp.Regexp
	repos             map[string][]config.Nag // index is org/repo, value is org-level nags
}

const nagSignature = "\n\n_Courtesy of your friendly test nag_."

var scope = log.RegisterScope("nagger", "The GitHub test nagger", 0)

func NewNagger(gc *gh.ThrottledClient, cache *cache.Cache, orgs []config.Org, nags []config.Nag) (filters.Filter, error) {
	n := &Nagger{
		cache:             cache,
		gc:                gc,
		orgs:              orgs,
		nags:              nags,
		multiLineRegexes:  make(map[string]*regexp.Regexp),
		singleLineRegexes: make(map[string]*regexp.Regexp),
		repos:             make(map[string][]config.Nag),
	}

	for _, nag := range nags {
		if err := n.processNagRegexes(nag); err != nil {
			return nil, err
		}
	}

	for _, org := range orgs {
		for _, nag := range org.Nags {
			if err := n.processNagRegexes(nag); err != nil {
				return nil, err
			}
		}
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			n.repos[org.Name+"/"+repo.Name] = org.Nags
		}
	}

	return n, nil
}

// Precompile all the regexes
func (n *Nagger) processNagRegexes(nag config.Nag) error {
	for _, expr := range nag.MatchTitle {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		n.singleLineRegexes[expr] = r
	}

	for _, expr := range nag.MatchBody {
		r, err := regexp.Compile("(?mi)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		n.multiLineRegexes[expr] = r
	}

	for _, expr := range nag.MatchFiles {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		n.singleLineRegexes[expr] = r
	}

	for _, expr := range nag.AbsentFiles {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		n.singleLineRegexes[expr] = r
	}

	return nil
}

// process an event arriving from GitHub
func (n *Nagger) Handle(context context.Context, event interface{}) {
	prp, ok := event.(*github.PullRequestEvent)
	if !ok {
		// not what we're looking for
		return
	}

	// see if the PR is in a repo we're monitoring
	nags, ok := n.repos[prp.GetRepo().GetFullName()]
	if !ok {
		scope.Infof("Ignoring PR %d from repo %s since it's not in a monitored repo", prp.Number, prp.GetRepo().GetFullName())
		return
	}

	// NOTE: this assumes the PR state has already been stored by the refresher filter
	pr, err := n.cache.ReadPullRequest(context, prp.GetRepo().GetOwner().GetLogin(), prp.GetRepo().GetName(), prp.GetPullRequest().GetNumber())
	if err != nil {
		scope.Errorf("Unable to retrieve data from storage for PR %d from repo %s: %v", prp.Number, prp.GetRepo().GetFullName(), err)
		return
	}

	scope.Infof("Processing PR %d from repo %s", prp.Number, prp.GetRepo().GetFullName())

	n.processPR(context, pr, nags)
}

// process a PR
func (n *Nagger) processPR(context context.Context, pr *storage.PullRequest, orgNags []config.Nag) {
	body := pr.Body
	title := pr.Title

	contentMatches := make([]config.Nag, 0)
	for _, nag := range n.nags {
		if n.titleMatch(nag, title) || n.bodyMatch(nag, body) {
			contentMatches = append(contentMatches, nag)
		}
	}

	for _, nag := range orgNags {
		if n.titleMatch(nag, title) || n.bodyMatch(nag, body) {
			contentMatches = append(contentMatches, nag)
		}
	}

	if len(contentMatches) == 0 {
		scope.Infof("Nothing to nag about for PR %d from repo %s/%s since its title and body don't match any nags",
			pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
		n.removeNagComment(context, pr)
		return
	}

	fileMatches := make([]config.Nag, 0)
	for _, nag := range contentMatches {
		if n.fileMatch(nag.MatchFiles, pr.Files) {
			fileMatches = append(fileMatches, nag)
		}
	}

	if len(fileMatches) == 0 {
		scope.Infof("Nothing to nag about for PR %d from repo %s/%s since its affected files don't match any nags",
			pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
		n.removeNagComment(context, pr)
		return
	}

	// at this point, fileMatches contains any nags whose MatchFile and (MatchTitle|MatchBody) regexes matched

	// now see if the required files are present in order to avoid the nag comment
	for _, nag := range fileMatches {
		if !n.fileMatch(nag.AbsentFiles, pr.Files) {
			scope.Infof("Nagging PR %d from repo %s/%s (nag: %s)", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, nag.Name)
			n.postNagComment(context, pr, nag)

			// only post a single nag comment per PR even if it's got multiple hits
			return
		}
	}

	scope.Infof("Nothing to nag about for PR %d from repo %s/%s since it contains required files", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName)
	n.removeNagComment(context, pr)
}

func (n *Nagger) removeNagComment(context context.Context, pr *storage.PullRequest) {
	existing, id := n.getNagComment(context, pr)
	if existing != "" {
		if _, err := n.gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
			return client.Issues.DeleteComment(context, pr.OrgLogin, pr.RepoName, id)
		}); err != nil {
			scope.Errorf("Unable to delete nag comment in PR %d from repo %s/%s: %v", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
		}
	}
}

func (n *Nagger) getNagComment(context context.Context, pr *storage.PullRequest) (string, int64) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := n.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListComments(context, pr.OrgLogin, pr.RepoName, int(pr.PullRequestNumber), opt)
		})

		if err != nil {
			scope.Errorf("Unable to list comments for pull request %d in repo %s/%s: %v\n", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
			return "", -1
		}

		for _, comment := range comments.([]*github.IssueComment) {
			body := comment.GetBody()
			if strings.Contains(body, nagSignature) {
				return body, comment.GetID()
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return "", -1
}

func (n *Nagger) postNagComment(context context.Context, pr *storage.PullRequest, nag config.Nag) {
	msg := nag.Message + nagSignature
	pc := &github.IssueComment{
		Body: &msg,
	}

	existing, id := n.getNagComment(context, pr)
	if existing == msg {
		// nag comment is already present
		return
	} else if existing != "" {
		// try to delete the previous nag
		if _, err := n.gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
			return client.Issues.DeleteComment(context, pr.OrgLogin, pr.RepoName, id)
		}); err != nil {
			scope.Errorf("Unable to delete nag comment in PR %d from repo %s/%s: %v", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
		}
	}

	_, _, err := n.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.CreateComment(context, pr.OrgLogin, pr.RepoName, int(pr.PullRequestNumber), pc)
	})

	if err != nil {
		scope.Errorf("Unable to attach nagging comment to PR %d from repo %s/%s: %v", pr.PullRequestNumber, pr.OrgLogin, pr.RepoName, err)
	}
}

func (n *Nagger) titleMatch(nag config.Nag, title string) bool {
	for _, expr := range nag.MatchTitle {
		r := n.singleLineRegexes[expr]
		if r.MatchString(title) {
			return true
		}
	}

	return false
}

func (n *Nagger) bodyMatch(nag config.Nag, body string) bool {
	for _, expr := range nag.MatchBody {
		r := n.multiLineRegexes[expr]
		if r.MatchString(body) {
			return true
		}
	}

	return false
}

func (n *Nagger) fileMatch(expressions []string, files []string) bool {
	for _, expr := range expressions {
		r := n.singleLineRegexes[expr]
		for _, f := range files {
			if r.MatchString(f) {
				return true
			}
		}
	}

	return false
}
