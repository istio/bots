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
	"time"

	hook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/pkg/storage"
)

// Maps from a GitHub webhook event to a storage issue. Also returns the set of
// users discovered in the event in a map of {UserID:Login}.
func IssueFromHook(ip *hook.IssuesPayload) (*storage.Issue, map[string]string) {
	labels := make([]string, len(ip.Issue.Labels))
	for i, label := range ip.Issue.Labels {
		labels[i] = label.NodeID
	}

	discoveredUsers := make(map[string]string, len(ip.Issue.Assignees))

	assignees := make([]string, len(ip.Issue.Assignees))
	for i, user := range ip.Issue.Assignees {
		assignees[i] = user.NodeID
		discoveredUsers[user.NodeID] = user.Login
	}

	var closedAt time.Time
	if ip.Issue.ClosedAt != nil {
		closedAt = *ip.Issue.ClosedAt
	}

	return &storage.Issue{
		OrgID:       ip.Repository.Owner.NodeID,
		RepoID:      ip.Repository.NodeID,
		IssueID:     ip.Issue.NodeID,
		Number:      ip.Issue.Number,
		Title:       ip.Issue.Title,
		Body:        ip.Issue.Body,
		LabelIDs:    labels,
		CreatedAt:   ip.Issue.CreatedAt,
		UpdatedAt:   ip.Issue.UpdatedAt,
		ClosedAt:    closedAt,
		State:       ip.Issue.State,
		AuthorID:    ip.Issue.User.NodeID,
		AssigneeIDs: assignees,
	}, discoveredUsers
}

// Maps from a GitHub webhook event to a storage issue comment. Also returns the set of
// users discovered in the event in a map of {UserID:Login}.
func IssueCommentFromHook(icp *hook.IssueCommentPayload) (*storage.IssueComment, map[string]string) {
	discoveredUsers := map[string]string{
		icp.Comment.User.NodeID: icp.Comment.User.Login,
	}

	return &storage.IssueComment{
		OrgID:          icp.Repository.Owner.NodeID,
		RepoID:         icp.Repository.NodeID,
		IssueID:        icp.Issue.NodeID,
		IssueCommentID: icp.Comment.NodeID,
		Body:           icp.Comment.Body,
		CreatedAt:      icp.Comment.CreatedAt,
		UpdatedAt:      icp.Comment.UpdatedAt,
		AuthorID:       icp.Comment.User.NodeID,
	}, discoveredUsers
}

// Maps from a GitHub webhook event to a storage repo comment.
// Also returns the set of
// users discovered in the event in a map of {UserID:Login}.
func RepoCommentFromHook(icp *hook.CommitCommentPayload) (*storage.RepoComment, map[string]string) {
	discoveredUsers := map[string]string{
		icp.Comment.User.NodeID: icp.Comment.User.Login,
	}

	return &storage.RepoComment{
		OrgID:     icp.Repository.Owner.NodeID,
		RepoID:    icp.Repository.NodeID,
		CommentID: icp.Comment.NodeID,
		Body:      icp.Comment.Body,
		CreatedAt: icp.Comment.CreatedAt,
		UpdatedAt: icp.Comment.UpdatedAt,
		AuthorID:  icp.Comment.User.NodeID,
	}, discoveredUsers
}

// Maps from a GitHub webhook event to a storage pr. Also returns the set of
// users discovered in the event in a map of {UserID:Login}.
//
// WARNING: sadly, the webhook doesn't supply the set of files affected by the PR
// so the Files field of the returned storage.PullRequest will not have been
// populated and will need to be retrieved separately by calling the GitHub API
func PullRequestFromHook(prp *hook.PullRequestPayload) (*storage.PullRequest, map[string]string) {
	labels := make([]string, len(prp.PullRequest.Labels))
	for i, label := range prp.PullRequest.Labels {
		labels[i] = label.NodeID
	}

	discoveredUsers := make(map[string]string, len(prp.PullRequest.Assignees)+len(prp.PullRequest.RequestedReviewers))

	assignees := make([]string, len(prp.PullRequest.Assignees))
	for i, user := range prp.PullRequest.Assignees {
		assignees[i] = user.NodeID
		discoveredUsers[user.NodeID] = user.Login
	}

	reviewers := make([]string, len(prp.PullRequest.RequestedReviewers))
	for i, user := range prp.PullRequest.RequestedReviewers {
		reviewers[i] = user.NodeID
		discoveredUsers[user.NodeID] = user.Login
	}

	var closedAt time.Time
	if prp.PullRequest.ClosedAt != nil {
		closedAt = *prp.PullRequest.ClosedAt
	}

	var mergedAt time.Time
	if prp.PullRequest.MergedAt != nil {
		closedAt = *prp.PullRequest.MergedAt
	}

	return &storage.PullRequest{
		OrgID:                prp.Repository.Owner.NodeID,
		RepoID:               prp.Repository.NodeID,
		PullRequestID:        prp.PullRequest.NodeID,
		CreatedAt:            prp.PullRequest.CreatedAt,
		UpdatedAt:            prp.PullRequest.UpdatedAt,
		ClosedAt:             closedAt,
		MergedAt:             mergedAt,
		State:                prp.PullRequest.State,
		Title:                prp.PullRequest.Title,
		Body:                 prp.PullRequest.Body,
		Number:               prp.PullRequest.Number,
		LabelIDs:             labels,
		AssigneeIDs:          assignees,
		RequestedReviewerIDs: reviewers,
	}, discoveredUsers
}

// Maps from a GitHub webhook event to a storage pr comment. Also returns the set of
// users discovered in the event in a map of {UserID:Login}.
func PullRequestCommentFromHook(icp *hook.IssueCommentPayload) (*storage.PullRequestComment, map[string]string) {
	discoveredUsers := map[string]string{
		icp.Comment.User.NodeID: icp.Comment.User.Login,
	}

	return &storage.PullRequestComment{
		OrgID:                icp.Repository.Owner.NodeID,
		RepoID:               icp.Repository.NodeID,
		PullRequestID:        icp.Issue.NodeID,
		PullRequestCommentID: icp.Comment.NodeID,
		Body:                 icp.Comment.Body,
		CreatedAt:            icp.Comment.CreatedAt,
		UpdatedAt:            icp.Comment.UpdatedAt,
		AuthorID:             icp.Comment.User.NodeID,
	}, discoveredUsers
}

// Maps from a GitHub webhook event to a storage pr review. Also returns the set of
// users discovered in the event in a map of {UserID:Login}.
func PullRequestReviewFromHook(prrp *hook.PullRequestReviewPayload) (*storage.PullRequestReview, map[string]string) {
	discoveredUsers := map[string]string{
		prrp.Review.User.NodeID: prrp.Review.User.Login,
	}

	return &storage.PullRequestReview{
		OrgID:               prrp.Repository.Owner.NodeID,
		RepoID:              prrp.Repository.NodeID,
		PullRequestID:       prrp.PullRequest.NodeID,
		PullRequestReviewID: prrp.Review.NodeID,
		Body:                prrp.Review.Body,
		SubmittedAt:         prrp.Review.SubmittedAt,
		AuthorID:            prrp.Review.User.NodeID,
		State:               prrp.Review.State,
	}, discoveredUsers
}
