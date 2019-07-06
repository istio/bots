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

// Maps from a GitHub issue to a storage issue. Also returns the set of
// users discovered in the input.
func ConvertIssue(orgID string, repoID string, issue *api.Issue) (*storage.Issue, []*storage.User) {
	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.GetNodeID()
	}

	discoveredUsers := make([]*storage.User, 0, len(issue.Assignees))

	assignees := make([]string, len(issue.Assignees))
	for i, user := range issue.Assignees {
		assignees[i] = user.GetNodeID()
		discoveredUsers = append(discoveredUsers, ConvertUser(user))
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
	}, discoveredUsers
}

// Maps from a GitHub issue comment to a storage issue comment. Also returns the set of
// users discovered in the input.
func ConvertIssueComment(orgID string, repoID string, issueID string, issueComment *api.IssueComment) (*storage.IssueComment, []*storage.User) {
	discoveredUsers := []*storage.User{
		ConvertUser(issueComment.GetUser()),
	}

	return &storage.IssueComment{
		OrgID:          orgID,
		RepoID:         repoID,
		IssueID:        issueID,
		IssueCommentID: issueComment.GetNodeID(),
		Body:           issueComment.GetBody(),
		CreatedAt:      issueComment.GetCreatedAt(),
		UpdatedAt:      issueComment.GetUpdatedAt(),
		AuthorID:       issueComment.GetUser().GetNodeID(),
	}, discoveredUsers
}

// Maps from a GitHub repo comment to a storage repo comment. Also returns the set of
// users discovered in the input.
func ConvertRepoComment(orgID string, repoID string, comment *api.RepositoryComment) (*storage.RepoComment, []*storage.User) {
	discoveredUsers := []*storage.User{
		ConvertUser(comment.GetUser()),
	}

	return &storage.RepoComment{
		OrgID:     orgID,
		RepoID:    repoID,
		CommentID: comment.GetNodeID(),
		Body:      comment.GetBody(),
		CreatedAt: comment.GetCreatedAt(),
		UpdatedAt: comment.GetUpdatedAt(),
		AuthorID:  comment.GetUser().GetNodeID(),
	}, discoveredUsers
}

// Maps from a GitHub user to a storage user.
func ConvertUser(u *api.User) *storage.User {
	return &storage.User{
		UserID:    u.GetNodeID(),
		Login:     u.GetLogin(),
		Name:      u.GetName(),
		Company:   u.GetCompany(),
		AvatarURL: u.GetAvatarURL(),
	}
}

// Maps from a GitHub org to a storage org.
func ConvertOrg(o *api.Organization) *storage.Org {
	return &storage.Org{
		OrgID: o.GetNodeID(),
		Login: o.GetLogin(),
	}
}

// Maps from a GitHub repo to a storage repo. Also returns the set of
func ConvertRepo(r *api.Repository) *storage.Repo {
	return &storage.Repo{
		OrgID:       r.Organization.GetNodeID(),
		RepoID:      r.GetNodeID(),
		Name:        r.GetName(),
		Description: r.GetDescription(),
		RepoNumber:  r.GetID(),
	}
}

// Maps from a GitHub label to a storage label.
func ConvertLabel(orgID string, repoID string, l *api.Label) *storage.Label {
	return &storage.Label{
		OrgID:       orgID,
		RepoID:      repoID,
		Name:        l.GetName(),
		Description: l.GetDescription(),
	}
}

// Maps from a GitHub pr to a storage pr. Also returns the set of
// users discovered in the input.
func ConvertPullRequest(orgID string, repoID string, pr *api.PullRequest, files []string) (*storage.PullRequest, []*storage.User) {
	labels := make([]string, len(pr.Labels))
	for i, label := range pr.Labels {
		labels[i] = label.GetNodeID()
	}

	discoveredUsers := make([]*storage.User, 0, len(pr.Assignees)+len(pr.RequestedReviewers))

	assignees := make([]string, len(pr.Assignees))
	for i, user := range pr.Assignees {
		assignees[i] = user.GetNodeID()
		discoveredUsers = append(discoveredUsers, ConvertUser(user))
	}

	reviewers := make([]string, len(pr.RequestedReviewers))
	for i, user := range pr.RequestedReviewers {
		reviewers[i] = user.GetNodeID()
		discoveredUsers = append(discoveredUsers, ConvertUser(user))
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
	}, discoveredUsers
}

// Maps from a GitHub pr comment to a storage pr comment. Also returns the set of
// users discovered in the input.
func ConvertPullRequestComment(orgID string, repoID string, prID string, comment *api.IssueComment) (*storage.PullRequestComment, []*storage.User) {
	discoveredUsers := []*storage.User{
		ConvertUser(comment.GetUser()),
	}

	return &storage.PullRequestComment{
		OrgID:                orgID,
		RepoID:               repoID,
		PullRequestID:        prID,
		PullRequestCommentID: comment.GetNodeID(),
		Body:                 comment.GetBody(),
		CreatedAt:            comment.GetCreatedAt(),
		UpdatedAt:            comment.GetUpdatedAt(),
		AuthorID:             comment.GetUser().GetNodeID(),
	}, discoveredUsers
}

// Maps from a GitHub pr review to a storage pr review. Also returns the set of
// users discovered in the input.
func ConvertPullRequestReview(orgID string, repoID string, pullRequestID string, prr *api.PullRequestReview) (*storage.PullRequestReview, []*storage.User) {
	discoveredUsers := []*storage.User{
		ConvertUser(prr.GetUser()),
	}

	return &storage.PullRequestReview{
		OrgID:               orgID,
		RepoID:              repoID,
		PullRequestID:       pullRequestID,
		PullRequestReviewID: prr.GetNodeID(),
		Body:                prr.GetBody(),
		SubmittedAt:         prr.GetSubmittedAt(),
		AuthorID:            prr.GetUser().GetNodeID(),
		State:               prr.GetState(),
	}, discoveredUsers
}
