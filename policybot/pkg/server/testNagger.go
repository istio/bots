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

package server

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	webhook "github.com/go-playground/webhooks/github"
	"github.com/google/go-github/v25/github"
	"golang.org/x/oauth2"

	"istio.io/pkg/log"
)

const nagSignature = "\n\n_Courtesy of your friendly test nag_."

type PullRequestEventAction string

const (
	// PullRequestActionOpened means the PR was created
	PullRequestActionOpened PullRequestEventAction = "opened"
	// PullRequestActionEdited means the PR body changed.
	PullRequestActionEdited PullRequestEventAction = "edited"
	// PullRequestActionReopened means the PR was reopened.
	PullRequestActionReopened PullRequestEventAction = "reopened"
	// PullRequestActionSynchronize means the git state changed.
	PullRequestActionSynchronize PullRequestEventAction = "synchronize"
)

type testNagger struct {
	ctx               context.Context
	client            *github.Client
	orgs              []Org
	nags              []Nag
	multiLineRegexes  map[string]*regexp.Regexp
	singleLineRegexes map[string]*regexp.Regexp
}

func newTestNagger(ctx context.Context, githubAccessToken string, orgs []Org, nags []Nag) (*testNagger, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAccessToken},
	)
	httpClient := oauth2.NewClient(ctx, src)
	client := github.NewClient(httpClient)

	tn := &testNagger{
		client:            client,
		ctx:               ctx,
		orgs:              orgs,
		nags:              nags,
		multiLineRegexes:  make(map[string]*regexp.Regexp),
		singleLineRegexes: make(map[string]*regexp.Regexp),
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

	return tn, nil
}

// Precompile all the regexes
func (tn *testNagger) processNagRegexes(nag Nag) error {
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
			return err
		}
		tn.singleLineRegexes[expr] = r
	}

	for _, expr := range nag.AbsentFiles {
		r, err := regexp.Compile("(?i)" + expr)
		if err != nil {
			return err
		}
		tn.singleLineRegexes[expr] = r
	}

	return nil
}

func (tn *testNagger) handleNewPR(prp *webhook.PullRequestPayload) {
	// if the event one we care about?
	action := PullRequestEventAction(prp.Action)
	if action != PullRequestActionSynchronize &&
		action != PullRequestActionOpened &&
		action != PullRequestActionReopened &&
		action != PullRequestActionEdited {
		return
	}

	// see if the PR is in a repo we're monitoring
	for _, org := range tn.orgs {
		if org.Name == prp.Repository.Owner.Login {
			for _, repo := range org.Repos {
				if repo.Name == prp.Repository.Name {
					log.Infof("Processing PR %d from repo %s", prp.Number, prp.Repository.FullName)
					go tn.processPR(prp, org, repo)
					return
				}
			}
		}
	}

	log.Infof("Ignoring PR %d from repo %s since it's not in a monitored repo", prp.Number, prp.Repository.FullName)
}

func (tn *testNagger) processPR(prp *webhook.PullRequestPayload, org Org, repo Repo) {
	pr, _, err := tn.client.PullRequests.Get(tn.ctx, prp.Repository.Owner.Login, prp.Repository.Name, int(prp.Number))
	if err != nil {
		log.Errorf("Unable to get information on PR %d in repo %s: %v", prp.Number, prp.Repository.FullName, err)
		return
	}

	body := pr.GetBody()
	title := pr.GetTitle()

	contentMatches := make([]Nag, 0)
	for _, nag := range tn.nags {
		if tn.titleMatch(nag, title) || tn.bodyMatch(nag, body) {
			contentMatches = append(contentMatches, nag)
		}
	}

	for _, nag := range org.Nags {
		if tn.titleMatch(nag, title) || tn.bodyMatch(nag, body) {
			contentMatches = append(contentMatches, nag)
		}
	}

	if len(contentMatches) == 0 {
		log.Infof("Ignoring PR %d from repo %s since its title and body don't match any nags", prp.Number, prp.Repository.FullName)
		tn.removeNagComment(org.Name, repo.Name, int(prp.Number))
		return
	}

	opt := &github.ListOptions{
		PerPage: 100,
	}

	var allFiles []string
	for {
		files, resp, err := tn.client.PullRequests.ListFiles(tn.ctx, prp.Repository.Owner.Login, prp.Repository.Name, int(prp.Number), opt)
		if err != nil {
			scope.Errorf("Unable to list all files for pull request %d in repo %s: %v\n", prp.Number, prp.Repository.FullName, err)
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

	fileMatches := make([]Nag, 0)
	for _, nag := range contentMatches {
		if tn.fileMatch(nag.MatchFiles, allFiles) {
			fileMatches = append(fileMatches, nag)
		}
	}

	if len(fileMatches) == 0 {
		log.Infof("Ignoring PR %d from repo %s since its affected files don't match any nags", prp.Number, prp.Repository.FullName)
		tn.removeNagComment(org.Name, repo.Name, int(prp.Number))
		return
	}

	// at this point, fileMatches contains any nags whose MatchFile and (MatchTitle|MatchBody) regexes matched

	// now see if the required files are present in order to avoid the nag comment
	for _, nag := range fileMatches {
		if !tn.fileMatch(nag.AbsentFiles, allFiles) {
			tn.postNagComment(prp.Repository.Owner.Login, prp.Repository.Name, int(prp.Number), nag)

			// only post a single nag comment per PR even if it's got multiple hits
			return
		}
	}

	log.Infof("Approved PR %d from repo %s since it contains required files", prp.Number, prp.Repository.FullName)
	tn.removeNagComment(org.Name, repo.Name, int(prp.Number))
}

func (tn *testNagger) removeNagComment(org string, repo string, num int) {
	existing, id := tn.getNagComment(org, repo, num)
	if existing != "" {
		if _, err := tn.client.Issues.DeleteComment(tn.ctx, org, repo, id); err != nil {
			log.Errorf("Unable to delete nag comment in PR %d from repo %s/%s: %v", num, org, repo, err)
		}
	}
}

func (tn *testNagger) getNagComment(org string, repo string, num int) (string, int64) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := tn.client.Issues.ListComments(tn.ctx, org, repo, num, opt)
		if err != nil {
			scope.Errorf("Unable to list comments for pull request %d in repo %s/%s: %v\n", num, org, repo, err)
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

func (tn *testNagger) postNagComment(org string, repo string, num int, nag Nag) {
	msg := nag.Message + nagSignature
	pc := &github.IssueComment{
		Body: &msg,
	}

	existing, id := tn.getNagComment(org, repo, num)
	if existing == msg {
		// nag comment is already present
		return
	} else if existing != "" {
		// try to delete the previous nag
		if _, err := tn.client.Issues.DeleteComment(tn.ctx, org, repo, id); err != nil {
			log.Errorf("Unable to delete nag comment in PR %d from repo %s/%s: %v", num, org, repo, err)
		}
	}

	_, _, err := tn.client.Issues.CreateComment(tn.ctx, org, repo, num, pc)
	if err != nil {
		log.Errorf("Unable to attach nagging comment to PR %d from repo %s/%s: %v", num, org, repo, err)
	} else {
		log.Infof("Attached nagging comment to PR %d from repo %s/%s (nag: %s)", num, org, repo, nag.Name)
	}
}

func (tn *testNagger) titleMatch(nag Nag, title string) bool {
	for _, expr := range nag.MatchTitle {
		r := tn.singleLineRegexes[expr]
		if r.MatchString(title) {
			return true
		}
	}

	return false
}

func (tn *testNagger) bodyMatch(nag Nag, body string) bool {
	for _, expr := range nag.MatchBody {
		r := tn.multiLineRegexes[expr]
		if r.MatchString(body) {
			return true
		}
	}

	return false
}

func (tn *testNagger) fileMatch(expressions []string, files []string) bool {
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
