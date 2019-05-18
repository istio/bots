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
	"net/http"
	"regexp"
	"strings"

	webhook "github.com/go-playground/webhooks/github"
	"github.com/google/go-github/v25/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("nagger", "The GitHub test nagger", 0)

const nagSignature = "\n\n_Courtesy of your friendly test nag_."

type PullRequestEventAction string

// Generates nagging messages in PRs
type Nagger struct {
	ctx               context.Context
	ghs               *gh.GitHubState
	ght               *util.GitHubThrottle
	orgs              []config.Org
	store             storage.Store
	nags              []config.Nag
	multiLineRegexes  map[string]*regexp.Regexp
	singleLineRegexes map[string]*regexp.Regexp
	repos             map[string][]config.Nag // index is org/repo, value is org-level nags
}

func NewNagger(ctx context.Context, ght *util.GitHubThrottle, store storage.Store, ghs *gh.GitHubState, orgs []config.Org, nags []config.Nag) (*Nagger, error) {
	tn := &Nagger{
		ctx:               ctx,
		ghs:               ghs,
		ght:               ght,
		orgs:              orgs,
		store:             store,
		nags:              nags,
		multiLineRegexes:  make(map[string]*regexp.Regexp),
		singleLineRegexes: make(map[string]*regexp.Regexp),
		repos:             make(map[string][]config.Nag),
	}

	for _, nag := range nags {
		if err := tn.processNagRegexes(nag); err != nil {
			return nil, err
		}
	}

	for _, org := range orgs {
		for _, nag := range org.Nags {
			if err := tn.processNagRegexes(nag); err != nil {
				return nil, err
			}
		}
	}

	for _, org := range orgs {
		for _, repo := range org.Repos {
			tn.repos[org.Name+"/"+repo.Name] = org.Nags
		}
	}

	return tn, nil
}

// Precompile all the regexes
func (tn *Nagger) processNagRegexes(nag config.Nag) error {
	for _, expr := range nag.MatchTitle {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		tn.singleLineRegexes[expr] = r
	}

	for _, expr := range nag.MatchBody {
		r, err := regexp.Compile("(?mi)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		tn.multiLineRegexes[expr] = r
	}

	for _, expr := range nag.MatchFiles {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		tn.singleLineRegexes[expr] = r
	}

	for _, expr := range nag.AbsentFiles {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %s: %v", expr, err)
		}
		tn.singleLineRegexes[expr] = r
	}

	return nil
}

func (tn *Nagger) Events() []webhook.Event {
	return []webhook.Event{
		webhook.PullRequestEvent,
	}
}

// asynchronously process a PR arriving from GitHub
func (tn *Nagger) Handle(_ http.ResponseWriter, githubObject interface{}) {
	prp, ok := githubObject.(webhook.PullRequestPayload)
	if !ok {
		// not what we're looking for
		return
	}

	// is the event one we care about?
	if prp.Action != "synchronize" &&
		prp.Action != "opened" &&
		prp.Action != "reopened" &&
		prp.Action != "edited" {
		return
	}

	a := tn.ghs.NewAccumulator()
	pr, err := a.PullRequestFromHook(&prp)
	if err != nil {
		scope.Errorf("Unable to process PR %d from repo %s: %v", prp.Number, prp.Repository.FullName, err)
		return
	}

	// see if the PR is in a repo we're monitoring
	nags, ok := tn.repos[prp.Repository.FullName]
	if !ok {
		scope.Infof("Ignoring PR %d from repo %s since it's not in a monitored repo", prp.Number, prp.Repository.FullName)
		return
	}

	scope.Infof("Processing PR %d from repo %s", prp.Number, prp.Repository.FullName)

	split := strings.Split(prp.Repository.FullName, "/")
	prb := pullRequedtBundle{pr, prp.Repository.FullName, split[0], split[1]}
	go tn.processPR(prb, nags)
}

type pullRequedtBundle struct {
	*storage.PullRequest
	fullRepoName string
	orgName      string
	repoName     string
}

// synchronously process a PR
func (tn *Nagger) processPR(prb pullRequedtBundle, orgNags []config.Nag) {
	body := prb.Body
	title := prb.Title

	contentMatches := make([]config.Nag, 0)
	for _, nag := range tn.nags {
		if tn.titleMatch(nag, title) || tn.bodyMatch(nag, body) {
			contentMatches = append(contentMatches, nag)
		}
	}

	for _, nag := range orgNags {
		if tn.titleMatch(nag, title) || tn.bodyMatch(nag, body) {
			contentMatches = append(contentMatches, nag)
		}
	}

	if len(contentMatches) == 0 {
		scope.Infof("Approving PR %d from repo %s since its title and body don't match any nags", prb.Number, prb.fullRepoName)
		tn.removeNagComment(prb)
		return
	}

	opt := &github.ListOptions{
		PerPage: 100,
	}

	var allFiles []string
	for {
		files, resp, err := tn.ght.Get().PullRequests.ListFiles(tn.ctx, prb.orgName, prb.repoName, prb.Number, opt)
		if err != nil {
			scope.Errorf("Unable to list all files for pull request %d in repo %s: %v\n", prb.Number, prb.fullRepoName, err)
			return
		}

		for _, f := range files {
			allFiles = append(allFiles, f.GetFilename())
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	fileMatches := make([]config.Nag, 0)
	for _, nag := range contentMatches {
		if tn.fileMatch(nag.MatchFiles, allFiles) {
			fileMatches = append(fileMatches, nag)
		}
	}

	if len(fileMatches) == 0 {
		scope.Infof("Approving PR %d from repo %s since its affected files don't match any nags", prb.Number, prb.fullRepoName)
		tn.removeNagComment(prb)
		return
	}

	// at this point, fileMatches contains any nags whose MatchFile and (MatchTitle|MatchBody) regexes matched

	// now see if the required files are present in order to avoid the nag comment
	for _, nag := range fileMatches {
		if !tn.fileMatch(nag.AbsentFiles, allFiles) {
			scope.Infof("Nagging PR %d from repo %s (nag: %s)", prb.Number, prb.fullRepoName, nag.Name)
			tn.postNagComment(prb, nag)

			// only post a single nag comment per PR even if it's got multiple hits
			return
		}
	}

	scope.Infof("Approving PR %d from repo %s since it contains required files", prb.Number, prb.fullRepoName)
	tn.removeNagComment(prb)
}

func (tn *Nagger) removeNagComment(prb pullRequedtBundle) {
	existing, id := tn.getNagComment(prb)
	if existing != "" {
		if _, err := tn.ght.Get().Issues.DeleteComment(tn.ctx, prb.orgName, prb.repoName, id); err != nil {
			scope.Errorf("Unable to delete nag comment in PR %d from repo %s: %v", prb.Number, prb.fullRepoName, err)
		} else {
			_ = tn.store.RecordTestNagRemoved(prb.RepoID)
		}
	}
}

func (tn *Nagger) getNagComment(prb pullRequedtBundle) (string, int64) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := tn.ght.Get().Issues.ListComments(tn.ctx, prb.orgName, prb.repoName, prb.Number, opt)
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

func (tn *Nagger) postNagComment(prb pullRequedtBundle, nag config.Nag) {
	msg := nag.Message + nagSignature
	pc := &github.IssueComment{
		Body: &msg,
	}

	existing, id := tn.getNagComment(prb)
	if existing == msg {
		// nag comment is already present
		return
	} else if existing != "" {
		// try to delete the previous nag
		if _, err := tn.ght.Get().Issues.DeleteComment(tn.ctx, prb.orgName, prb.repoName, id); err != nil {
			scope.Errorf("Unable to delete nag comment in PR %d from repo %s: %v", prb.Number, prb.fullRepoName, err)
		} else {
			_ = tn.store.RecordTestNagRemoved(prb.RepoID)
		}
	}

	_, _, err := tn.ght.Get().Issues.CreateComment(tn.ctx, prb.orgName, prb.repoName, prb.Number, pc)
	if err != nil {
		scope.Errorf("Unable to attach nagging comment to PR %d from repo %s: %v", prb.Number, prb.fullRepoName, err)
	} else {
		err = tn.store.RecordTestNagAdded(prb.RepoID)
		if err != nil {
			scope.Errorf("Unable to record test nag addition: %v", err)
		}
	}
}

func (tn *Nagger) titleMatch(nag config.Nag, title string) bool {
	for _, expr := range nag.MatchTitle {
		r := tn.singleLineRegexes[expr]
		if r.MatchString(title) {
			return true
		}
	}

	return false
}

func (tn *Nagger) bodyMatch(nag config.Nag, body string) bool {
	for _, expr := range nag.MatchBody {
		r := tn.multiLineRegexes[expr]
		if r.MatchString(body) {
			return true
		}
	}

	return false
}

func (tn *Nagger) fileMatch(expressions []string, files []string) bool {
	for _, expr := range expressions {
		r := tn.singleLineRegexes[expr]
		for _, f := range files {
			if r.MatchString(f) {
				return true
			}
		}
	}

	return false
}
