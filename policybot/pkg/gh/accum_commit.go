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

import "istio.io/bots/policybot/pkg/storage"

// Commit all the accumulated objects to the cache and to durable storage.
// Note that if any errors occur in the middle of this, partial updates to the
// cache and DB will happen. Sorry 'bout that. Either way, the accumulator is also
// clean upon return so it can be reused
func (a *Accumulator) Commit() error {
	var err error
	if err = a.commitUsers(); err == nil {
		if err = a.commitOrgs(); err == nil {
			if err = a.commitRepos(); err == nil {
				if err = a.commitLabels(); err == nil {
					if err = a.commitIssues(); err == nil {
						if err = a.commitIssueComments(); err == nil {
							if err = a.commitPullRequests(); err == nil {
								if err = a.commitPullRequestReviews(); err == nil {
									err = a.commitMembers()
								}
							}
						}
					}
				}
			}
		}
	}

	a.Reset()
	return err
}

func (a *Accumulator) commitLabels() error {
	if len(a.labels) == 0 {
		return nil
	}

	labels := make([]*storage.Label, 0, len(a.labels))
	for _, l := range a.labels {
		labels = append(labels, l)
	}

	if err := a.ghs.store.WriteLabels(labels); err != nil {
		return err
	}

	for _, l := range a.labels {
		a.ghs.labelCache.Set(l.LabelID, l)
	}

	return nil
}

func (a *Accumulator) commitUsers() error {
	if len(a.users) == 0 {
		return nil
	}

	users := make([]*storage.User, 0, len(a.users))
	for _, u := range a.users {
		users = append(users, u)
	}

	if err := a.ghs.store.WriteUsers(users); err != nil {
		return err
	}

	for _, l := range a.users {
		a.ghs.userCache.Set(l.UserID, l)
	}

	return nil
}

func (a *Accumulator) commitOrgs() error {
	if len(a.orgs) == 0 {
		return nil
	}

	orgs := make([]*storage.Org, 0, len(a.orgs))
	for _, o := range a.orgs {
		orgs = append(orgs, o)
	}

	if err := a.ghs.store.WriteOrgs(orgs); err != nil {
		return err
	}

	for _, l := range a.orgs {
		a.ghs.orgCache.Set(l.OrgID, l)
	}

	return nil
}

func (a *Accumulator) commitRepos() error {
	if len(a.repos) == 0 {
		return nil
	}

	repos := make([]*storage.Repo, 0, len(a.repos))
	for _, r := range a.repos {
		repos = append(repos, r)
	}

	if err := a.ghs.store.WriteRepos(repos); err != nil {
		return err
	}

	for _, l := range a.repos {
		a.ghs.repoCache.Set(l.RepoID, l)
	}

	return nil
}

func (a *Accumulator) commitIssues() error {
	if len(a.issues) == 0 {
		return nil
	}

	issues := make([]*storage.Issue, 0, len(a.issues))
	for _, is := range a.issues {
		issues = append(issues, is)
	}

	if err := a.ghs.store.WriteIssues(issues); err != nil {
		return err
	}

	for _, l := range a.issues {
		a.ghs.issueCache.Set(l.IssueID, l)
	}

	return nil
}

func (a *Accumulator) commitIssueComments() error {
	if len(a.issueComments) == 0 {
		return nil
	}

	ics := make([]*storage.IssueComment, 0, len(a.issueComments))
	for _, ic := range a.issueComments {
		ics = append(ics, ic)
	}

	if err := a.ghs.store.WriteIssueComments(ics); err != nil {
		return err
	}

	for _, l := range a.issueComments {
		a.ghs.issueCommentCache.Set(l.IssueCommentID, l)
	}

	return nil
}

func (a *Accumulator) commitPullRequests() error {
	if len(a.pullRequests) == 0 {
		return nil
	}

	prs := make([]*storage.PullRequest, 0, len(a.pullRequests))
	for _, pr := range a.pullRequests {
		prs = append(prs, pr)
	}

	if err := a.ghs.store.WritePullRequests(prs); err != nil {
		return err
	}

	for _, l := range a.pullRequests {
		a.ghs.pullRequestCache.Set(l.IssueID, l)
	}

	return nil
}

func (a *Accumulator) commitPullRequestReviews() error {
	if len(a.pullRequestReviews) == 0 {
		return nil
	}

	prrs := make([]*storage.PullRequestReview, 0, len(a.pullRequestReviews))
	for _, prr := range a.pullRequestReviews {
		prrs = append(prrs, prr)
	}

	if err := a.ghs.store.WritePullRequestReviews(prrs); err != nil {
		return err
	}

	for _, l := range a.pullRequestReviews {
		a.ghs.pullRequestReviewCache.Set(l.PullRequestReviewID, l)
	}

	return nil
}

func (a *Accumulator) commitMembers() error {
	if len(a.members) == 0 {
		return nil
	}

	members := make([]*storage.Member, 0, len(a.members))
	for _, member := range a.members {
		members = append(members, member)
	}

	if err := a.ghs.store.WriteAllMembers(members); err != nil {
		return err
	}

	return nil
}
