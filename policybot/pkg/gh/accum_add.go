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

// Add an object to the accumulator. The object is only added if it doesn't
// match the existing content of the cache. If the object is not added, then
// the object already in the cache is returned, otherwise the input is returned.
func (a *Accumulator) addObj(id string, object interface{}) interface{} {
	if existing, ok := a.ghs.cache.Get(id); ok {
		if reflect.DeepEqual(existing, object) {
			return existing
		}
	}

	a.objects[id] = object
	return object
}

func (a *Accumulator) addLabel(label *storage.Label) *storage.Label {
	o := a.addObj(label.LabelID, label).(*storage.Label)
	if o != label {
		a.labels = append(a.labels, label)
	}
	return o
}

func (a *Accumulator) addUser(user *storage.User) *storage.User {
	o := a.addObj(user.UserID, user).(*storage.User)
	if o == user {
		a.users = append(a.users, user)
	}
	return o
}

func (a *Accumulator) addOrg(org *storage.Org) *storage.Org {
	o := a.addObj(org.OrgID, org).(*storage.Org)
	if o == org {
		a.orgs = append(a.orgs, org)
	}
	return o
}

func (a *Accumulator) addRepo(repo *storage.Repo) *storage.Repo {
	o := a.addObj(repo.RepoID, repo).(*storage.Repo)
	if o == repo {
		a.repos = append(a.repos, repo)
	}
	return o
}

func (a *Accumulator) addIssue(issue *storage.Issue) *storage.Issue {
	o := a.addObj(issue.IssueID, issue).(*storage.Issue)
	if o == issue {
		a.issues = append(a.issues, issue)
	}
	return o
}

func (a *Accumulator) addIssueComment(issueComment *storage.IssueComment) *storage.IssueComment {
	o := a.addObj(issueComment.IssueCommentID, issueComment).(*storage.IssueComment)
	if o == issueComment {
		a.issueComments = append(a.issueComments, issueComment)
	}
	return o
}

func (a *Accumulator) addPullRequest(pr *storage.PullRequest) *storage.PullRequest {
	o := a.addObj(pr.IssueID+pullRequestIDSuffix, pr).(*storage.PullRequest)
	if o == pr {
		a.pullRequests = append(a.pullRequests, pr)
	}
	return o
}

func (a *Accumulator) addPullRequestReview(prc *storage.PullRequestReview) *storage.PullRequestReview {
	o := a.addObj(prc.PullRequestReviewID, prc).(*storage.PullRequestReview)
	if o == prc {
		a.pullRequestReviews = append(a.pullRequestReviews, prc)
	}
	return o
}
