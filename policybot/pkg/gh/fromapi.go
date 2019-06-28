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
	api "github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/storage"
)

func IssueFromAPI(orgID string, repoID string, issue *api.Issue) *storage.Issue {
	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.GetNodeID()
	}

	assignees := make([]string, len(issue.Assignees))
	for i, user := range issue.Assignees {
		assignees[i] = user.GetNodeID()
	}

	return &storage.Issue{
		OrgID:       orgID,
		RepoID:      repoID,
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
	}
}

func IssueCommentFromAPI(orgID string, repoID string, issueID string, issueComment *api.IssueComment) *storage.IssueComment {
	return &storage.IssueComment{
		OrgID:          orgID,
		RepoID:         repoID,
		IssueID:        issueID,
		IssueCommentID: issueComment.GetNodeID(),
		Body:           issueComment.GetBody(),
		CreatedAt:      issueComment.GetCreatedAt(),
		UpdatedAt:      issueComment.GetUpdatedAt(),
		AuthorID:       issueComment.GetUser().GetNodeID(),
	}
}

func RepoCommentFromAPI(orgID string, repoID string, comment *api.RepositoryComment) *storage.RepoComment {
	return &storage.RepoComment{
		OrgID:     orgID,
		RepoID:    repoID,
		CommentID: comment.GetNodeID(),
		Body:      comment.GetBody(),
		CreatedAt: comment.GetCreatedAt(),
		UpdatedAt: comment.GetUpdatedAt(),
		AuthorID:  comment.GetUser().GetNodeID(),
	}
}

func UserFromAPI(u *api.User) *storage.User {
	return &storage.User{
		UserID:    u.GetNodeID(),
		Login:     u.GetLogin(),
		Name:      u.GetName(),
		Company:   u.GetCompany(),
		AvatarURL: u.GetAvatarURL(),
	}
}

func OrgFromAPI(o *api.Organization) *storage.Org {
	return &storage.Org{
		OrgID: o.GetNodeID(),
		Login: o.GetLogin(),
	}
}

func RepoFromAPI(r *api.Repository) *storage.Repo {
	return &storage.Repo{
		OrgID:       r.Organization.GetNodeID(),
		RepoID:      r.GetNodeID(),
		Name:        r.GetName(),
		Description: r.GetDescription(),
		RepoNumber:  r.GetID(),
	}
}

func LabelFromAPI(orgID string, repoID string, l *api.Label) *storage.Label {
	return &storage.Label{
		OrgID:       orgID,
		RepoID:      repoID,
		Name:        l.GetName(),
		Description: l.GetDescription(),
	}
}

func PullRequestFromAPI(orgID string, repoID string, pr *api.PullRequest, files []string) *storage.PullRequest {
	labels := make([]string, len(pr.Labels))
	for i, label := range pr.Labels {
		labels[i] = label.GetNodeID()
	}

	assignees := make([]string, len(pr.Assignees))
	for i, user := range pr.Assignees {
		assignees[i] = user.GetNodeID()
	}

	reviewers := make([]string, len(pr.RequestedReviewers))
	for i, user := range pr.RequestedReviewers {
		reviewers[i] = user.GetNodeID()
	}

	return &storage.PullRequest{
		OrgID:                orgID,
		RepoID:               repoID,
		PullRequestID:        pr.GetNodeID(),
		UpdatedAt:            pr.GetUpdatedAt(),
		CreatedAt:            pr.GetCreatedAt(),
		ClosedAt:             pr.GetClosedAt(),
		MergedAt:             pr.GetMergedAt(),
		Files:                files,
		LabelIDs:             labels,
		AssigneeIDs:          assignees,
		RequestedReviewerIDs: reviewers,
		State:                pr.GetState(),
		Title:                pr.GetTitle(),
		Body:                 pr.GetBody(),
		AuthorID:             pr.GetUser().GetNodeID(),
	}
}

func PullRequestCommentFromAPI(orgID string, repoID string, prID string, issueComment *api.IssueComment) *storage.PullRequestComment {
	return &storage.PullRequestComment{
		OrgID:                orgID,
		RepoID:               repoID,
		PullRequestID:        prID,
		PullRequestCommentID: issueComment.GetNodeID(),
		Body:                 issueComment.GetBody(),
		CreatedAt:            issueComment.GetCreatedAt(),
		UpdatedAt:            issueComment.GetUpdatedAt(),
		AuthorID:             issueComment.GetUser().GetNodeID(),
	}
}

func PullRequestReviewFromAPI(orgID string, repoID string, pullRequestID string, prr *api.PullRequestReview) *storage.PullRequestReview {
	return &storage.PullRequestReview{
		OrgID:               orgID,
		RepoID:              repoID,
		PullRequestID:       pullRequestID,
		PullRequestReviewID: prr.GetNodeID(),
		Body:                prr.GetBody(),
		SubmittedAt:         prr.GetSubmittedAt(),
		AuthorID:            prr.GetUser().GetNodeID(),
		State:               prr.GetState(),
	}
}
