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
								err = a.commitPullRequestReviews()
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

	if err := a.ghs.store.WriteLabels(a.labels); err != nil {
		return err
	}

	for _, l := range a.labels {
		a.ghs.cache.Set(l.LabelID, l)
	}

	return nil
}

func (a *Accumulator) commitUsers() error {
	if len(a.users) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteUsers(a.users); err != nil {
		return err
	}

	for _, l := range a.users {
		a.ghs.cache.Set(l.UserID, l)
	}

	return nil
}

func (a *Accumulator) commitOrgs() error {
	if len(a.orgs) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteOrgs(a.orgs); err != nil {
		return err
	}

	for _, l := range a.orgs {
		a.ghs.cache.Set(l.OrgID, l)
	}

	return nil
}

func (a *Accumulator) commitRepos() error {
	if len(a.repos) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteRepos(a.repos); err != nil {
		return err
	}

	for _, l := range a.repos {
		a.ghs.cache.Set(l.RepoID, l)
	}

	return nil
}

func (a *Accumulator) commitIssues() error {
	if len(a.issues) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteIssues(a.issues); err != nil {
		return err
	}

	for _, l := range a.issues {
		a.ghs.cache.Set(l.IssueID, l)
	}

	return nil
}

func (a *Accumulator) commitIssueComments() error {
	if len(a.issueComments) == 0 {
		return nil
	}

	if err := a.ghs.store.WriteIssueComments(a.issueComments); err != nil {
		return err
	}

	for _, l := range a.issueComments {
		a.ghs.cache.Set(l.IssueCommentID, l)
	}

	return nil
}

func (a *Accumulator) commitPullRequests() error {
	if len(a.pullRequests) == 0 {
		return nil
	}

	if err := a.ghs.store.WritePullRequests(a.pullRequests); err != nil {
		return err
	}

	for _, l := range a.pullRequests {
		a.ghs.cache.Set(l.IssueID+pullRequestIDSuffix, l)
	}

	return nil
}

func (a *Accumulator) commitPullRequestReviews() error {
	if len(a.pullRequestReviews) == 0 {
		return nil
	}

	if err := a.ghs.store.WritePullRequestReviews(a.pullRequestReviews); err != nil {
		return err
	}

	for _, l := range a.pullRequestReviews {
		a.ghs.cache.Set(l.PullRequestReviewID, l)
	}

	return nil
}
