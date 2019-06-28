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

	webhook "github.com/go-playground/webhooks/github"
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
	ght               *gh.ThrottledClient
	orgs              []config.Org
	nags              []config.Nag
	multiLineRegexes  map[string]*regexp.Regexp
	singleLineRegexes map[string]*regexp.Regexp
	repos             map[string][]config.Nag // index is org/repo, value is org-level nags
}

const nagSignature = "\n\n_Courtesy of your friendly test nag_."

var scope = log.RegisterScope("nagger", "The GitHub test nagger", 0)

func NewNagger(ght *gh.ThrottledClient, cache *cache.Cache, orgs []config.Org, nags []config.Nag) (filters.Filter, error) {
	n := &Nagger{
		cache:             cache,
		ght:               ght,
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

func (n *Nagger) Events() []webhook.Event {
	return []webhook.Event{
		webhook.PullRequestEvent,
	}
}

// process an event arriving from GitHub
func (n *Nagger) Handle(context context.Context, githubObject interface{}) {
	prp, ok := githubObject.(webhook.PullRequestPayload)
	if !ok {
		// not what we're looking for
		return
	}

	// see if the PR is in a repo we're monitoring
	nags, ok := n.repos[prp.Repository.FullName]
	if !ok {
		scope.Infof("Ignoring PR %d from repo %s since it's not in a monitored repo", prp.Number, prp.Repository.FullName)
		return
	}

	// NOTE: this assumes the PR state has already been stored by the refresher plugin
	pr, err := n.cache.ReadPullRequest(context, prp.Repository.Owner.NodeID, prp.Repository.NodeID, prp.PullRequest.NodeID)
	if err != nil {
		scope.Errorf("Unable to retrieve data from storage for PR %d from repo %s: %v", prp.Number, prp.Repository.FullName, err)
		return
	}

	scope.Infof("Processing PR %d from repo %s", prp.Number, prp.Repository.FullName)

	split := strings.Split(prp.Repository.FullName, "/")
	prb := pullRequestBundle{pr, prp.Repository.FullName, split[0], split[1]}
	n.processPR(context, prb, nags)
}

type pullRequestBundle struct {
	*storage.PullRequest
	fullRepoName string
	orgName      string
	repoName     string
}

// process a PR
func (n *Nagger) processPR(context context.Context, prb pullRequestBundle, orgNags []config.Nag) {
	body := prb.Body
	title := prb.Title

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
		scope.Infof("Nothing to nag about for PR %d from repo %s since its title and body don't match any nags", prb.Number, prb.fullRepoName)
		n.removeNagComment(context, prb)
		return
	}

	fileMatches := make([]config.Nag, 0)
	for _, nag := range contentMatches {
		if n.fileMatch(nag.MatchFiles, prb.PullRequest.Files) {
			fileMatches = append(fileMatches, nag)
		}
	}

	if len(fileMatches) == 0 {
		scope.Infof("Nothing to nag about for PR %d from repo %s since its affected files don't match any nags", prb.Number, prb.fullRepoName)
		n.removeNagComment(context, prb)
		return
	}

	// at this point, fileMatches contains any nags whose MatchFile and (MatchTitle|MatchBody) regexes matched

	// now see if the required files are present in order to avoid the nag comment
	for _, nag := range fileMatches {
		if !n.fileMatch(nag.AbsentFiles, prb.PullRequest.Files) {
			scope.Infof("Nagging PR %d from repo %s (nag: %s)", prb.Number, prb.fullRepoName, nag.Name)
			n.postNagComment(context, prb, nag)

			// only post a single nag comment per PR even if it's got multiple hits
			return
		}
	}

	scope.Infof("Nothing to nag about for PR %d from repo %s since it contains required files", prb.Number, prb.fullRepoName)
	n.removeNagComment(context, prb)
}

func (n *Nagger) removeNagComment(context context.Context, prb pullRequestBundle) {
	existing, id := n.getNagComment(context, prb)
	if existing != "" {
		if _, err := n.ght.Get(context).Issues.DeleteComment(context, prb.orgName, prb.repoName, id); err != nil {
			scope.Errorf("Unable to delete nag comment in PR %d from repo %s: %v", prb.Number, prb.fullRepoName, err)
		}
	}
}

func (n *Nagger) getNagComment(context context.Context, prb pullRequestBundle) (string, int64) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := n.ght.Get(context).Issues.ListComments(context, prb.orgName, prb.repoName, int(prb.Number), opt)
		if err != nil {
			scope.Errorf("Unable to list comments for pull request %d in repo %s: %v\n", prb.Number, prb.fullRepoName, err)
			return "", -1
		}

		for _, comment := range comments {
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

func (n *Nagger) postNagComment(context context.Context, prb pullRequestBundle, nag config.Nag) {
	msg := nag.Message + nagSignature
	pc := &github.IssueComment{
		Body: &msg,
	}

	existing, id := n.getNagComment(context, prb)
	if existing == msg {
		// nag comment is already present
		return
	} else if existing != "" {
		// try to delete the previous nag
		if _, err := n.ght.Get(context).Issues.DeleteComment(context, prb.orgName, prb.repoName, id); err != nil {
			scope.Errorf("Unable to delete nag comment in PR %d from repo %s: %v", prb.Number, prb.fullRepoName, err)
		}
	}

	_, _, err := n.ght.Get(context).Issues.CreateComment(context, prb.orgName, prb.repoName, int(prb.Number), pc)
	if err != nil {
		scope.Errorf("Unable to attach nagging comment to PR %d from repo %s: %v", prb.Number, prb.fullRepoName, err)
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
