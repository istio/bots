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
	api "github.com/google/go-github/v25/github"

	"istio.io/bots/policybot/pkg/storage"
)

// These functions ingest GitHub API objects and insert their normalized equivalents
// into the accumulator so that they can eventually be committed to storage.

func (a *Accumulator) IssueFromAPI(org string, repo string, issue *api.Issue) *storage.Issue {
	if result := a.issues[issue.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
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
		OrgID:       org,
		RepoID:      repo,
		IssueID:     issue.GetNodeID(),
		Number:      int64(issue.GetNumber()),
		Title:       issue.GetTitle(),
		Body:        issue.GetBody(),
		LabelIDs:    labels,
		CreatedAt:   issue.GetCreatedAt(),
		UpdatedAt:   issue.GetUpdatedAt(),
		ClosedAt:    issue.GetClosedAt(),
		State:       issue.GetState(),
		AuthorID:    issue.GetUser().GetNodeID(),
		AssigneeIDs: assignees,
	})
}

func (a *Accumulator) IssueCommentFromAPI(org string, repo string, issue string, issueComment *api.IssueComment) *storage.IssueComment {
	if result := a.issueComments[issueComment.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
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
	if result := a.users[u.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
	}

	return a.addUser(&storage.User{
		UserID:  u.GetNodeID(),
		Login:   u.GetLogin(),
		Name:    u.GetName(),
		Company: u.GetCompany(),
	})
}

func (a *Accumulator) OrgFromAPI(o *api.Organization) *storage.Org {
	if result := a.orgs[o.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
	}

	return a.addOrg(&storage.Org{
		OrgID: o.GetNodeID(),
		Login: o.GetLogin(),
	})
}

func (a *Accumulator) RepoFromAPI(r *api.Repository) *storage.Repo {
	if result := a.repos[r.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
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
	if result := a.labels[l.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
	}

	return a.addLabel(&storage.Label{
		OrgID:       org,
		RepoID:      repo,
		Name:        l.GetName(),
		Description: l.GetDescription(),
	})
}

func (a *Accumulator) PullRequestFromAPI(org string, repo string, pr *api.PullRequest, files []string) *storage.PullRequest {
	if result := a.pullRequests[pr.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
	}

	labels := make([]api.Label, len(pr.Labels))
	for i, label := range pr.Labels {
		labels[i] = *label
	}

	_ = a.IssueFromAPI(org, repo, &api.Issue{
		Number:    pr.Number,
		State:     pr.State,
		Title:     pr.Title,
		Body:      pr.Body,
		User:      pr.User,
		Labels:    labels,
		Comments:  pr.Comments,
		ClosedAt:  pr.ClosedAt,
		CreatedAt: pr.CreatedAt,
		UpdatedAt: pr.UpdatedAt,
		Assignees: pr.Assignees,
		NodeID:    pr.NodeID,
	})

	reviewers := make([]string, len(pr.RequestedReviewers))
	for i, user := range pr.RequestedReviewers {
		reviewers[i] = user.GetNodeID()
		_ = a.UserFromAPI(user)
	}

	return a.addPullRequest(&storage.PullRequest{
		OrgID:                org,
		RepoID:               repo,
		IssueID:              pr.GetNodeID(),
		RequestedReviewerIDs: reviewers,
		UpdatedAt:            pr.GetUpdatedAt(),
		Files:                files,
	})
}

func (a *Accumulator) PullRequestReviewFromAPI(org string, repo string, issue string, prr *api.PullRequestReview) *storage.PullRequestReview {
	if result := a.pullRequestReviews[prr.GetNodeID()]; result != nil {
		// already in the accumulator
		return result
	}

	_ = a.UserFromAPI(prr.User)

	return a.addPullRequestReview(&storage.PullRequestReview{
		OrgID:               org,
		RepoID:              repo,
		IssueID:             issue,
		PullRequestReviewID: prr.GetNodeID(),
		Body:                prr.GetBody(),
		SubmittedAt:         prr.GetSubmittedAt(),
		AuthorID:            prr.GetUser().GetNodeID(),
		State:               prr.GetState(),
	})
}

func (a *Accumulator) MemberFromAPI(o *storage.Org, u *api.User) *storage.Member {
	user := a.UserFromAPI(u)

	member := &storage.Member{
		OrgID:  o.OrgID,
		UserID: user.UserID,
	}

	a.addMember(member)
	return member
}
