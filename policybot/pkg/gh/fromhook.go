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

func IssueFromHook(ip *hook.IssuesPayload) *storage.Issue {
	labels := make([]string, len(ip.Issue.Labels))
	for i, label := range ip.Issue.Labels {
		labels[i] = label.NodeID
	}

	assignees := make([]string, len(ip.Issue.Assignees))
	for i, user := range ip.Issue.Assignees {
		assignees[i] = user.NodeID
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
	}
}

func IssueCommentFromHook(icp *hook.IssueCommentPayload) *storage.IssueComment {
	return &storage.IssueComment{
		OrgID:          icp.Repository.Owner.NodeID,
		RepoID:         icp.Repository.NodeID,
		IssueID:        icp.Issue.NodeID,
		IssueCommentID: icp.Comment.NodeID,
		Body:           icp.Comment.Body,
		CreatedAt:      icp.Comment.CreatedAt,
		UpdatedAt:      icp.Comment.UpdatedAt,
		AuthorID:       icp.Comment.User.NodeID,
	}
}

func PullRequestFromHook(prp *hook.PullRequestPayload) (*storage.PullRequest, *storage.Issue) {
	labels := make([]string, len(prp.PullRequest.Labels))
	for i, label := range prp.PullRequest.Labels {
		labels[i] = label.NodeID
	}

	assignees := make([]string, len(prp.PullRequest.Assignees))
	for i, user := range prp.PullRequest.Assignees {
		assignees[i] = user.NodeID
	}

	reviewers := make([]string, len(prp.PullRequest.RequestedReviewers))
	for i, user := range prp.PullRequest.RequestedReviewers {
		reviewers[i] = user.NodeID
	}

	var p hook.IssuesPayload
	p.Repository.NodeID = prp.Repository.NodeID
	p.Repository.Name = prp.Repository.Name
	p.Repository.FullName = prp.Repository.FullName
	p.Repository.Owner.NodeID = prp.Repository.Owner.NodeID
	p.Issue.NodeID = prp.PullRequest.NodeID
	p.Issue.Number = prp.PullRequest.Number
	p.Issue.Title = prp.PullRequest.Title
	p.Issue.Body = prp.PullRequest.Body
	p.Issue.CreatedAt = prp.PullRequest.CreatedAt
	p.Issue.UpdatedAt = prp.PullRequest.UpdatedAt
	p.Issue.ClosedAt = prp.PullRequest.ClosedAt
	p.Issue.Assignees = prp.PullRequest.Assignees
	p.Issue.Labels = prp.PullRequest.Labels
	p.Issue.State = prp.PullRequest.State

	return &storage.PullRequest{
		OrgID:                prp.Repository.Owner.NodeID,
		RepoID:               prp.Repository.NodeID,
		IssueID:              prp.PullRequest.NodeID,
		RequestedReviewerIDs: reviewers,
		UpdatedAt:            prp.PullRequest.UpdatedAt,
	}, IssueFromHook(&p)
}

func PullRequestReviewFromHook(prrp *hook.PullRequestReviewPayload) *storage.PullRequestReview {
	return &storage.PullRequestReview{
		OrgID:               prrp.Repository.Owner.NodeID,
		RepoID:              prrp.Repository.NodeID,
		IssueID:             prrp.Review.NodeID,
		PullRequestReviewID: prrp.Review.NodeID,
		Body:                prrp.Review.Body,
		SubmittedAt:         prrp.Review.SubmittedAt,
		AuthorID:            prrp.Review.User.NodeID,
		State:               prrp.Review.State,
	}
}
