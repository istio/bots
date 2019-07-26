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
	"strings"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/storage"
)

// Maps from a GitHub issue to a storage issue. Also returns the set of
// users discovered in the input.
func ConvertIssue(orgLogin string, repoName string, issue *github.Issue) *storage.Issue {
	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.GetName()
	}

	assignees := make([]string, len(issue.Assignees))
	for i, user := range issue.Assignees {
		assignees[i] = user.GetLogin()
	}

	return &storage.Issue{
		OrgLogin:    orgLogin,
		RepoName:    repoName,
		IssueNumber: int64(issue.GetNumber()),
		Title:       issue.GetTitle(),
		Body:        issue.GetBody(),
		Labels:      labels,
		CreatedAt:   issue.GetCreatedAt(),
		UpdatedAt:   issue.GetUpdatedAt(),
		ClosedAt:    issue.GetClosedAt(),
		State:       issue.GetState(),
		Author:      issue.GetUser().GetLogin(),
		Assignees:   assignees,
	}
}

// Maps from a GitHub issue comment to a storage issue comment. Also returns the set of
// users discovered in the input.
func ConvertIssueComment(orgLogin string, repoName string, issueNumber int, issueComment *github.IssueComment) *storage.IssueComment {
	return &storage.IssueComment{
		OrgLogin:       orgLogin,
		RepoName:       repoName,
		IssueNumber:    int64(issueNumber),
		IssueCommentID: issueComment.GetID(),
		Body:           issueComment.GetBody(),
		CreatedAt:      issueComment.GetCreatedAt(),
		UpdatedAt:      issueComment.GetUpdatedAt(),
		Author:         issueComment.GetUser().GetLogin(),
	}
}

// Maps from a GitHub repo comment to a storage repo comment. Also returns the set of
// users discovered in the input.
func ConvertRepoComment(orgLogin string, repoName string, comment *github.RepositoryComment) *storage.RepoComment {
	return &storage.RepoComment{
		OrgLogin:  orgLogin,
		RepoName:  repoName,
		CommentID: comment.GetID(),
		Body:      comment.GetBody(),
		CreatedAt: comment.GetCreatedAt(),
		UpdatedAt: comment.GetUpdatedAt(),
		Author:    comment.GetUser().GetLogin(),
	}
}

// Maps from a GitHub user to a storage user.
func ConvertUser(u *github.User) *storage.User {
	return &storage.User{
		UserLogin: u.GetLogin(),
		Name:      u.GetName(),
		Company:   u.GetCompany(),
		AvatarURL: u.GetAvatarURL(),
	}
}

// Maps from a GitHub org to a storage org.
func ConvertOrg(o *github.Organization) *storage.Org {
	return &storage.Org{
		OrgLogin:    o.GetLogin(),
		Company:     o.GetCompany(),
		Description: o.GetDescription(),
		AvatarURL:   o.GetAvatarURL(),
	}
}

// Maps from a GitHub repo to a storage repo. Also returns the set of
func ConvertRepo(r *github.Repository) *storage.Repo {
	return &storage.Repo{
		OrgLogin:    r.Organization.GetLogin(),
		RepoName:    r.GetName(),
		Description: r.GetDescription(),
		RepoNumber:  r.GetID(),
	}
}

// Maps from a GitHub label to a storage label.
func ConvertLabel(orgLogin string, repoName string, l *github.Label) *storage.Label {
	return &storage.Label{
		OrgLogin:    orgLogin,
		RepoName:    repoName,
		LabelName:   l.GetName(),
		Description: l.GetDescription(),
		Color:       l.GetColor(),
	}
}

// Maps from a GitHub pr to a storage pr. Also returns the set of
// users discovered in the input.
func ConvertPullRequest(orgLogin string, repoName string, pr *github.PullRequest, files []string) *storage.PullRequest {
	labels := make([]string, len(pr.Labels))
	for i, label := range pr.Labels {
		labels[i] = label.GetName()
	}

	assignees := make([]string, len(pr.Assignees))
	for i, user := range pr.Assignees {
		assignees[i] = user.GetLogin()
	}

	reviewers := make([]string, len(pr.RequestedReviewers))
	for i, user := range pr.RequestedReviewers {
		reviewers[i] = user.GetLogin()
	}

	sha := pr.GetMergeCommitSHA()
	if sha == "" { // Not merged, so use the current head
		sha = pr.GetHead().GetSHA()
	}
	base := pr.GetBase().GetLabel()
	branch := base[strings.Index(base, ":")+1:]

	return &storage.PullRequest{
		OrgLogin:           orgLogin,
		RepoName:           repoName,
		PullRequestNumber:  int64(pr.GetNumber()),
		UpdatedAt:          pr.GetUpdatedAt(),
		CreatedAt:          pr.GetCreatedAt(),
		ClosedAt:           pr.GetClosedAt(),
		MergedAt:           pr.GetMergedAt(),
		Files:              files,
		Labels:             labels,
		Assignees:          assignees,
		RequestedReviewers: reviewers,
		State:              pr.GetState(),
		Title:              pr.GetTitle(),
		Body:               pr.GetBody(),
		Author:             pr.GetUser().GetLogin(),
		HeadCommit:         sha,
		BranchName:         branch,
	}
}

// Maps from a GitHub pr comment to a storage pr comment. Also returns the set of
// users discovered in the input.
func ConvertPullRequestReviewComment(orgLogin string, repoName string, prNumber int,
	comment *github.PullRequestComment) *storage.PullRequestReviewComment {

	return &storage.PullRequestReviewComment{
		OrgLogin:                   orgLogin,
		RepoName:                   repoName,
		PullRequestNumber:          int64(prNumber),
		PullRequestReviewCommentID: comment.GetID(),
		Body:                       comment.GetBody(),
		CreatedAt:                  comment.GetCreatedAt(),
		UpdatedAt:                  comment.GetUpdatedAt(),
		Author:                     comment.GetUser().GetLogin(),
	}
}

// Maps from a GitHub pr review to a storage pr review. Also returns the set of
// users discovered in the input.
func ConvertPullRequestReview(orgLogin string, repoName string, prNumber int, prr *github.PullRequestReview) *storage.PullRequestReview {
	return &storage.PullRequestReview{
		OrgLogin:            orgLogin,
		RepoName:            repoName,
		PullRequestNumber:   int64(prNumber),
		PullRequestReviewID: prr.GetID(),
		Body:                prr.GetBody(),
		SubmittedAt:         prr.GetSubmittedAt(),
		Author:              prr.GetUser().GetLogin(),
		State:               prr.GetState(),
	}
}
