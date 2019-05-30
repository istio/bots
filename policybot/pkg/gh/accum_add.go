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
	"reflect"

	"istio.io/bots/policybot/pkg/storage"
)

// These functions add an object to the accumulator. The object is only added if it doesn't
// match the existing content of the cache. If the object is not added, then
// the object already in the cache is returned, otherwise the input is returned.

func (a *Accumulator) addLabel(label *storage.Label) *storage.Label {
	if existing, ok := a.ghs.labelCache.Get(label.LabelID); ok {
		if reflect.DeepEqual(existing, label) {
			return existing.(*storage.Label)
		}
	}

	a.labels[label.LabelID] = label
	return label
}

func (a *Accumulator) addUser(user *storage.User) *storage.User {
	if existing, ok := a.ghs.userCache.Get(user.UserID); ok {
		if reflect.DeepEqual(existing, user) {
			return existing.(*storage.User)
		}
	}

	a.users[user.UserID] = user
	return user
}

func (a *Accumulator) addOrg(org *storage.Org) *storage.Org {
	if existing, ok := a.ghs.orgCache.Get(org.OrgID); ok {
		if reflect.DeepEqual(existing, org) {
			return existing.(*storage.Org)
		}
	}

	a.orgs[org.OrgID] = org
	return org
}

func (a *Accumulator) addRepo(repo *storage.Repo) *storage.Repo {
	if existing, ok := a.ghs.repoCache.Get(repo.RepoID); ok {
		if reflect.DeepEqual(existing, repo) {
			return existing.(*storage.Repo)
		}
	}

	a.repos[repo.RepoID] = repo
	return repo
}

func (a *Accumulator) addIssue(issue *storage.Issue) *storage.Issue {
	if existing, ok := a.ghs.issueCache.Get(issue.IssueID); ok {
		if reflect.DeepEqual(existing, issue) {
			return existing.(*storage.Issue)
		}
	}

	a.issues[issue.IssueID] = issue
	return issue
}

func (a *Accumulator) addIssueComment(issueComment *storage.IssueComment) *storage.IssueComment {
	if existing, ok := a.ghs.issueCommentCache.Get(issueComment.IssueCommentID); ok {
		if reflect.DeepEqual(existing, issueComment) {
			return existing.(*storage.IssueComment)
		}
	}

	a.issueComments[issueComment.IssueCommentID] = issueComment
	return issueComment
}

func (a *Accumulator) addPullRequest(pr *storage.PullRequest) *storage.PullRequest {
	if existing, ok := a.ghs.pullRequestCache.Get(pr.IssueID); ok {
		if reflect.DeepEqual(existing, pr) {
			return existing.(*storage.PullRequest)
		}
	}

	a.pullRequests[pr.IssueID] = pr
	return pr
}

func (a *Accumulator) addPullRequestReview(prc *storage.PullRequestReview) *storage.PullRequestReview {
	if existing, ok := a.ghs.pullRequestReviewCache.Get(prc.PullRequestReviewID); ok {
		if reflect.DeepEqual(existing, prc) {
			return existing.(*storage.PullRequestReview)
		}
	}

	a.pullRequestReviews[prc.PullRequestReviewID] = prc
	return prc
}
