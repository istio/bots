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
	hook "github.com/go-playground/webhooks/github"

	"istio.io/bots/policybot/pkg/storage"
)

// Note that the equivalent XXXFromAPI functions do a deep insertion of
// secondary objects into the accumulator. Things like the user or the labels
// associated with te main object. These functions do not, because the state
// obtained from the hooks is often insufficient to fully produce these secondary
// objects. For example, the user info supplied in a hook doesn't include the user's
// full name.
//
// So we don't produce these secondary objects here and that can result in dangling references in the
// DB. Those are OK and will hopefully be addressed eventually by the syncer plugin which will
// introduce any missing entries into the DB.

func (a *Accumulator) IssueFromHook(ip *hook.IssuesPayload) *storage.Issue {
	if result := a.objects[ip.Issue.NodeID]; result != nil {
		return result.(*storage.Issue)
	}

	return a.addIssue(IssueFromHook(ip))
}

func (a *Accumulator) IssueCommentFromHook(icp *hook.IssueCommentPayload) *storage.IssueComment {
	if result := a.objects[icp.Comment.NodeID]; result != nil {
		return result.(*storage.IssueComment)
	}

	return a.addIssueComment(IssueCommentFromHook(icp))
}

func (a *Accumulator) PullRequestFromHook(prp *hook.PullRequestPayload) (*storage.PullRequest, *storage.Issue) {
	if result1 := a.objects[prp.PullRequest.NodeID+pullRequestIDSuffix]; result1 != nil {
		if result2 := a.objects[prp.PullRequest.NodeID]; result2 != nil {
			return result1.(*storage.PullRequest), result2.(*storage.Issue)
		}
	}

	pr, issue := PullRequestFromHook(prp)
	return a.addPullRequest(pr), a.addIssue(issue)
}

func (a *Accumulator) PullRequestReviewFromHook(prrp *hook.PullRequestReviewPayload) *storage.PullRequestReview {
	if result := a.objects[prrp.Review.NodeID]; result != nil {
		return result.(*storage.PullRequestReview)
	}

	return a.addPullRequestReview(PullRequestReviewFromHook(prrp))
}
