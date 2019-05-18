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

package gh

import (
	"fmt"
	"reflect"
	"strings"

	hook "github.com/go-playground/webhooks/github"
	api "github.com/google/go-github/v25/github"

	"istio.io/bots/policybot/pkg/storage"
)

// Accumulates objects in anticipation of bulk commits
type Accumulator struct {
	ghs           *GitHubState
	objects       map[string]interface{}
	labels        []*storage.Label
	users         []*storage.User
	orgs          []*storage.Org
	repos         []*storage.Repo
	issues        []*storage.Issue
	issueComments []*storage.IssueComment
	pullRequests  []*storage.PullRequest
}

func (ghs *GitHubState) NewAccumulator() *Accumulator {
	return &Accumulator{
		ghs:           ghs,
		objects:       make(map[string]interface{}),
		labels:        make([]*storage.Label, 0),
		users:         make([]*storage.User, 0),
		orgs:          make([]*storage.Org, 0),
		repos:         make([]*storage.Repo, 0),
		issues:        make([]*storage.Issue, 0),
		issueComments: make([]*storage.IssueComment, 0),
		pullRequests:  make([]*storage.PullRequest, 0),
	}
}

func (a *Accumulator) Absorb(other *Accumulator) {
	a.labels = append(a.labels, other.labels...)
	a.users = append(a.users, other.users...)
	a.orgs = append(a.orgs, other.orgs...)
	a.repos = append(a.repos, other.repos...)
	a.issues = append(a.issues, other.issues...)
	a.issueComments = append(a.issueComments, other.issueComments...)
	a.pullRequests = append(a.pullRequests, other.pullRequests...)

	for k, v := range other.objects {
		a.objects[k] = v
	}
}

func (a *Accumulator) accumulate(id string, object interface{}) interface{} {
	if existing, ok := a.ghs.cache.Get(id); ok {
		if reflect.DeepEqual(existing, object) {
			return existing
		}
	}

	a.objects[id] = object
	return object
}

func (a *Accumulator) addLabel(label *storage.Label) *storage.Label {
	o := a.accumulate(label.LabelID, label).(*storage.Label)
	if o != label {
		a.labels = append(a.labels, label)
	}
	return o
}

func (a *Accumulator) addUser(user *storage.User) *storage.User {
	o := a.accumulate(user.UserID, user).(*storage.User)
	if o == user {
		a.users = append(a.users, user)
	}
	return o
}

func (a *Accumulator) addOrg(org *storage.Org) *storage.Org {
	o := a.accumulate(org.OrgID, org).(*storage.Org)
	if o == org {
		a.orgs = append(a.orgs, org)
	}
	return o
}

func (a *Accumulator) addRepo(repo *storage.Repo) *storage.Repo {
	o := a.accumulate(repo.RepoID, repo).(*storage.Repo)
	if o == repo {
		a.repos = append(a.repos, repo)
	}
	return o
}

func (a *Accumulator) addIssue(issue *storage.Issue) *storage.Issue {
	o := a.accumulate(issue.IssueID, issue).(*storage.Issue)
	if o == issue {
		a.issues = append(a.issues, issue)
	}
	return o
}

func (a *Accumulator) addIssueComment(issueComment *storage.IssueComment) *storage.IssueComment {
	o := a.accumulate(issueComment.IssueCommentID, issueComment).(*storage.IssueComment)
	if o == issueComment {
		a.issueComments = append(a.issueComments, issueComment)
	}
	return o
}

func (a *Accumulator) addPullRequest(pr *storage.PullRequest) *storage.PullRequest {
	o := a.accumulate(pr.PullRequestID, pr).(*storage.PullRequest)
	if o == pr {
		a.pullRequests = append(a.pullRequests, pr)
	}
	return o
}

func (a *Accumulator) clean() {
	a.labels = a.labels[:0]
	a.users = a.users[:0]
	a.orgs = a.orgs[:0]
	a.repos = a.repos[:0]
	a.issues = a.issues[:0]
	a.issueComments = a.issueComments[:0]
	a.pullRequests = a.pullRequests[:0]

	for k := range a.objects {
		delete(a.objects, k)
	}
}

func (a *Accumulator) Commit() error {
	var err error
	if err = a.commitUsers(); err == nil {
		if err = a.commitOrgs(); err == nil {
			if err = a.commitRepos(); err == nil {
				if err = a.commitLabels(); err == nil {
					if err = a.commitIssues(); err == nil {
						if err = a.commitIssueComments(); err == nil {
							a.commitPullRequests()
						}
					}
				}
			}
		}
	}

	a.clean()
	return err
}

func (a *Accumulator) commitLabels() error {
	if len(a.labels) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteLabels(a.labels); err != nil {
		return err
	}

	for _, l := range a.labels {
		a.ghs.cache.Set(l.LabelID, l)
	}

	return nil
}

func (a *Accumulator) commitUsers() error {
	if len(a.users) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteUsers(a.users); err != nil {
		return err
	}

	for _, l := range a.users {
		a.ghs.cache.Set(l.UserID, l)
	}

	return nil
}

func (a *Accumulator) commitOrgs() error {
	if len(a.orgs) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteOrgs(a.orgs); err != nil {
		return err
	}

	for _, l := range a.orgs {
		a.ghs.cache.Set(l.OrgID, l)
	}

	return nil
}

func (a *Accumulator) commitRepos() error {
	if len(a.repos) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteRepos(a.repos); err != nil {
		return err
	}

	for _, l := range a.repos {
		a.ghs.cache.Set(l.RepoID, l)
	}

	return nil
}

func (a *Accumulator) commitIssues() error {
	if len(a.issues) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteIssues(a.issues); err != nil {
		return err
	}

	for _, l := range a.issues {
		a.ghs.cache.Set(l.IssueID, l)
	}

	return nil
}

func (a *Accumulator) commitIssueComments() error {
	if len(a.issueComments) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteIssueComments(a.issueComments); err != nil {
		return err
	}

	for _, l := range a.issueComments {
		a.ghs.cache.Set(l.IssueCommentID, l)
	}

	return nil
}

func (a *Accumulator) commitPullRequests() {
	if len(a.pullRequests) == 0 {
		return
	}

	for _, l := range a.pullRequests {
		a.ghs.cache.Set(l.PullRequestID, l)
	}
}

func (a *Accumulator) IssueFromAPI(org string, repo string, issue *api.Issue) *storage.Issue {
	if result := a.objects[issue.GetNodeID()]; result != nil {
		return result.(*storage.Issue)
	}

	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.GetNodeID()
		_ = a.LabelFromAPI(org, repo, &label)
	}

	assignees := make([]string, len(issue.Assignees))
	for i, user := range issue.Assignees {
		assignees[i] = user.GetNodeID()
		_ = a.UserFromAPI(user)
	}

	_ = a.UserFromAPI(issue.User)

	return a.addIssue(&storage.Issue{
		OrgID:         org,
		RepoID:        repo,
		IssueID:       issue.GetNodeID(),
		Number:        issue.GetNumber(),
		Title:         issue.GetTitle(),
		Body:          issue.GetBody(),
		LabelIDs:      labels,
		CreatedAt:     issue.GetCreatedAt(),
		UpdatedAt:     issue.GetUpdatedAt(),
		ClosedAt:      issue.GetClosedAt(),
		State:         issue.GetState(),
		AuthorID:      issue.GetUser().GetNodeID(),
		AssigneeIDs:   assignees,
		IsPullRequest: issue.IsPullRequest(),
	})
}

func (a *Accumulator) IssueCommentFromAPI(org string, repo string, issue string, issueComment *api.IssueComment) *storage.IssueComment {
	if result := a.objects[issueComment.GetNodeID()]; result != nil {
		return result.(*storage.IssueComment)
	}

	_ = a.UserFromAPI(issueComment.User)

	return a.addIssueComment(&storage.IssueComment{
		OrgID:          org,
		RepoID:         repo,
		IssueID:        issue,
		IssueCommentID: issueComment.GetNodeID(),
		Body:           issueComment.GetBody(),
		CreatedAt:      issueComment.GetCreatedAt(),
		UpdatedAt:      issueComment.GetUpdatedAt(),
		AuthorID:       issueComment.GetUser().GetNodeID(),
	})
}

func (a *Accumulator) UserFromAPI(u *api.User) *storage.User {
	if result := a.objects[u.GetNodeID()]; result != nil {
		return result.(*storage.User)
	}

	return a.addUser(&storage.User{
		UserID:  u.GetNodeID(),
		Login:   u.GetLogin(),
		Name:    u.GetName(),
		Company: u.GetCompany(),
	})
}

func (a *Accumulator) OrgFromAPI(o *api.Organization) *storage.Org {
	if result := a.objects[o.GetNodeID()]; result != nil {
		return result.(*storage.Org)
	}

	return a.addOrg(&storage.Org{
		OrgID: o.GetNodeID(),
		Login: o.GetLogin(),
	})
}

func (a *Accumulator) RepoFromAPI(r *api.Repository) *storage.Repo {
	if result := a.objects[r.GetNodeID()]; result != nil {
		return result.(*storage.Repo)
	}

	_ = a.OrgFromAPI(r.Organization)

	return a.addRepo(&storage.Repo{
		OrgID:       r.Organization.GetNodeID(),
		RepoID:      r.GetNodeID(),
		Name:        r.GetName(),
		Description: r.GetDescription(),
	})
}

func (a *Accumulator) LabelFromAPI(org string, repo string, l *api.Label) *storage.Label {
	if result := a.objects[l.GetNodeID()]; result != nil {
		return result.(*storage.Label)
	}

	return a.addLabel(&storage.Label{
		OrgID:       org,
		RepoID:      repo,
		Name:        l.GetName(),
		Description: l.GetDescription(),
	})
}

func (a *Accumulator) PullRequestFromAPI(org string, repo string, pr *api.PullRequest) *storage.PullRequest {
	if result := a.objects[pr.GetNodeID()]; result != nil {
		return result.(*storage.PullRequest)
	}

	labels := make([]string, len(pr.Labels))
	for i, label := range pr.Labels {
		labels[i] = label.GetNodeID()
		_ = a.LabelFromAPI(org, repo, label)
	}

	assignees := make([]string, len(pr.Assignees))
	for i, user := range pr.Assignees {
		assignees[i] = user.GetNodeID()
		_ = a.UserFromAPI(user)
	}

	_ = a.UserFromAPI(pr.User)

	return a.addPullRequest(&storage.PullRequest{
		OrgID:         org,
		RepoID:        repo,
		PullRequestID: pr.GetNodeID(),
		Number:        pr.GetNumber(),
		Title:         pr.GetTitle(),
		Body:          pr.GetBody(),
		LabelIDs:      labels,
		CreatedAt:     pr.GetCreatedAt(),
		UpdatedAt:     pr.GetUpdatedAt(),
		ClosedAt:      pr.GetClosedAt(),
		State:         pr.GetState(),
		AuthorID:      pr.GetUser().GetNodeID(),
		AssigneeIDs:   assignees,
	})
}

func (a *Accumulator) IssueFromHook(ip *hook.IssuesPayload) (*storage.Issue, error) {
	org, err := a.ghs.store.ReadOrgByLogin(strings.Split(ip.Repository.FullName, "/")[0])
	if org == nil {
		// don't know this org
		return nil, err
	}

	labels := make([]api.Label, len(ip.Issue.Labels))
	for i, label := range ip.Issue.Labels {
		labels[i] = api.Label{
			NodeID:      &label.NodeID,
			Name:        &label.Name,
			Description: &label.Description,
		}
	}

	assignees := make([]*api.User, len(ip.Issue.Assignees))
	for i, user := range ip.Issue.Assignees {
		assignees[i] = &api.User{
			NodeID:  &user.NodeID,
			Login:   &user.Login,
			Name:    nil, // TODO: unavailable
			Company: nil, // TODO: unavailable
		}
	}

	number := int(ip.Issue.Number)

	return a.IssueFromAPI(org.OrgID, ip.Repository.NodeID, &api.Issue{
		NodeID:    &ip.Issue.NodeID,
		Number:    &number,
		Title:     &ip.Issue.Title,
		Body:      &ip.Issue.Body,
		Labels:    labels,
		CreatedAt: &ip.Issue.CreatedAt,
		UpdatedAt: &ip.Issue.UpdatedAt,
		ClosedAt:  ip.Issue.ClosedAt,
		State:     &ip.Issue.State,
		User: &api.User{
			NodeID:  &ip.Issue.User.NodeID,
			Login:   &ip.Issue.User.Login,
			Name:    nil, // unavailable
			Company: nil, // unavailable
		},
		Assignees: assignees,
	}), nil
}

func (a *Accumulator) IssueCommentFromHook(icp *hook.IssueCommentPayload) (*storage.IssueComment, error) {
	org, err := a.ghs.store.ReadOrgByLogin(strings.Split(icp.Repository.FullName, "/")[0])
	if org == nil {
		// don't know this org
		return nil, err
	}

	labels := make([]api.Label, len(icp.Issue.Labels))
	for i, label := range icp.Issue.Labels {
		labels[i] = api.Label{
			NodeID:      &label.NodeID,
			Name:        &label.Name,
			Description: &label.Description,
		}
	}

	assignees := make([]*api.User, len(icp.Issue.Assignees))
	for i, user := range icp.Issue.Assignees {
		assignees[i] = &api.User{
			NodeID:  &user.NodeID,
			Login:   &user.Login,
			Name:    nil, // TODO: unavailable
			Company: nil, // TODO: unavailable
		}
	}

	number := int(icp.Issue.Number)

	_ = a.IssueFromAPI(org.OrgID, icp.Repository.NodeID, &api.Issue{
		NodeID:    &icp.Issue.NodeID,
		Number:    &number,
		Title:     &icp.Issue.Title,
		Body:      &icp.Issue.Body,
		Labels:    labels,
		CreatedAt: &icp.Issue.CreatedAt,
		UpdatedAt: &icp.Issue.UpdatedAt,
		ClosedAt:  icp.Issue.ClosedAt,
		State:     &icp.Issue.State,
		User: &api.User{
			NodeID:  &icp.Issue.User.NodeID,
			Login:   &icp.Issue.User.Login,
			Name:    nil, // unavailable
			Company: nil, // unavailable
		},
		Assignees: assignees,
	})

	return a.IssueCommentFromAPI(org.OrgID, icp.Repository.NodeID, icp.Issue.NodeID, &api.IssueComment{
		NodeID: &icp.Comment.NodeID,
		Body:   &icp.Comment.Body,
		User: &api.User{
			NodeID:  &icp.Comment.User.NodeID,
			Login:   &icp.Comment.User.Login,
			Name:    nil, // unavailable
			Company: nil, // unavailable
		},
		CreatedAt: &icp.Comment.CreatedAt,
		UpdatedAt: &icp.Comment.UpdatedAt,
	}), nil
}

func (a *Accumulator) PullRequestFromHook(prp *hook.PullRequestPayload) (*storage.PullRequest, error) {
	split := strings.Split(prp.Repository.FullName, "/")
	orgName := split[0]
	repoName := split[1]

	org, err := a.ghs.store.ReadOrgByLogin(orgName)
	if org == nil {
		// don't know this org
		return nil, fmt.Errorf("unknown organization %s: %v", orgName, err)
	}

	repo, err := a.ghs.store.ReadRepoByName(org.OrgID, repoName)
	if repo == nil {
		// don't know this repo
		return nil, fmt.Errorf("unknown repository %s: %v", prp.Repository.FullName, err)
	}

	labels := make([]*api.Label, len(prp.PullRequest.Labels))
	for i, label := range prp.PullRequest.Labels {
		labels[i] = &api.Label{
			NodeID:      &label.NodeID,
			Name:        &label.Name,
			Description: &label.Description,
		}
	}

	assignees := make([]*api.User, len(prp.PullRequest.Assignees))
	for i, user := range prp.PullRequest.Assignees {
		assignees[i] = &api.User{
			NodeID:  &user.NodeID,
			Login:   &user.Login,
			Name:    nil, // TODO: unavailable
			Company: nil, // TODO: unavailable
		}
	}

	number := int(prp.Number)

	return a.PullRequestFromAPI(org.OrgID, repo.RepoID, &api.PullRequest{
		NodeID:    &prp.PullRequest.NodeID,
		Number:    &number,
		Title:     &prp.PullRequest.Title,
		Body:      &prp.PullRequest.Body,
		Labels:    labels,
		CreatedAt: &prp.PullRequest.CreatedAt,
		UpdatedAt: &prp.PullRequest.UpdatedAt,
		ClosedAt:  prp.PullRequest.ClosedAt,
		State:     &prp.PullRequest.State,
		User: &api.User{
			NodeID:  &prp.PullRequest.User.NodeID,
			Login:   &prp.PullRequest.User.Login,
			Name:    nil, // unavailable
			Company: nil, // unavailable
		},
		Assignees: assignees,
	}), nil
}
