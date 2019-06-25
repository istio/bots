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

package spanner

import (
	"context"

	"cloud.google.com/go/spanner"

	"istio.io/bots/policybot/pkg/storage"
)

func (s store) WriteOrgs(context context.Context, orgs []*storage.Org) error {
	scope.Debugf("Writing %d orgs", len(orgs))

	mutations := make([]*spanner.Mutation, len(orgs))
	for i := 0; i < len(orgs); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(orgTable, orgs[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteRepos(context context.Context, repos []*storage.Repo) error {
	scope.Debugf("Writing %d repos", len(repos))

	mutations := make([]*spanner.Mutation, len(repos))
	for i := 0; i < len(repos); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(repoTable, repos[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteRepoComments(context context.Context, comments []*storage.RepoComment) error {
	scope.Debugf("Writing %d repo comments", len(comments))

	mutations := make([]*spanner.Mutation, len(comments))
	for i := 0; i < len(comments); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(repoCommentTable, comments[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteIssues(context context.Context, issues []*storage.Issue) error {
	scope.Debugf("Writing %d issues", len(issues))

	mutations := make([]*spanner.Mutation, len(issues))
	for i := 0; i < len(issues); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(issueTable, issues[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteIssueComments(context context.Context, issueComments []*storage.IssueComment) error {
	scope.Debugf("Writing %d issue comments", len(issueComments))

	mutations := make([]*spanner.Mutation, len(issueComments))
	for i := 0; i < len(issueComments); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(issueCommentTable, issueComments[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteIssuePipelines(context context.Context, issuePipelines []*storage.IssuePipeline) error {
	scope.Debugf("Writing %d issue pipelines", len(issuePipelines))

	mutations := make([]*spanner.Mutation, len(issuePipelines))
	for i := 0; i < len(issuePipelines); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(issuePipelineTable, issuePipelines[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePullRequests(context context.Context, prs []*storage.PullRequest) error {
	scope.Debugf("Writing %d pull requests", len(prs))

	mutations := make([]*spanner.Mutation, len(prs))
	for i := 0; i < len(prs); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(pullRequestTable, prs[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePullRequestComments(context context.Context, prComments []*storage.PullRequestComment) error {
	scope.Debugf("Writing %d pr comments", len(prComments))

	mutations := make([]*spanner.Mutation, len(prComments))
	for i := 0; i < len(prComments); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(pullRequestCommentTable, prComments[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePullRequestReviews(context context.Context, prReviews []*storage.PullRequestReview) error {
	scope.Debugf("Writing %d pull request reviews", len(prReviews))

	mutations := make([]*spanner.Mutation, len(prReviews))
	for i := 0; i < len(prReviews); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(pullRequestReviewTable, prReviews[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteUsers(context context.Context, users []*storage.User) error {
	scope.Debugf("Writing %d users", len(users))

	mutations := make([]*spanner.Mutation, len(users))
	for i := 0; i < len(users); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(userTable, users[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteLabels(context context.Context, labels []*storage.Label) error {
	scope.Debugf("Writing %d labels", len(labels))

	mutations := make([]*spanner.Mutation, len(labels))
	for i := 0; i < len(labels); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(labelTable, labels[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteAllMembers(ctx1 context.Context, members []*storage.Member) error {
	scope.Debugf("Writing %d members", len(members))

	mutations := make([]*spanner.Mutation, len(members))
	for i, member := range members {
		var err error
		if mutations[i], err = spanner.InsertStruct(memberTable, member); err != nil {
			return err
		}
	}

	_, err := s.client.ReadWriteTransaction(ctx1, func(ctx2 context.Context, txn *spanner.ReadWriteTransaction) error {
		// Remove all existing members
		iter := txn.Query(ctx2, spanner.Statement{SQL: "DELETE FROM Members WHERE true;"})
		if err := iter.Do(func(_ *spanner.Row) error { return nil }); err != nil {
			return err
		}

		// write all the new members
		return txn.BufferWrite(mutations)
	})

	return err
}

func (s store) WriteAllMaintainers(ctx1 context.Context, maintainers []*storage.Maintainer) error {
	scope.Debugf("Writing %d maintainers", len(maintainers))

	mutations := make([]*spanner.Mutation, len(maintainers))
	for i, maintainer := range maintainers {
		var err error
		if mutations[i], err = spanner.InsertStruct(maintainerTable, maintainer); err != nil {
			return err
		}
	}

	_, err := s.client.ReadWriteTransaction(ctx1, func(ctx2 context.Context, txn *spanner.ReadWriteTransaction) error {
		// Remove all existing maintainers
		iter := txn.Query(ctx2, spanner.Statement{SQL: "DELETE FROM Maintainers WHERE true;"})
		if err := iter.Do(func(_ *spanner.Row) error { return nil }); err != nil {
			return err
		}

		// write all the new maintainers
		return txn.BufferWrite(mutations)
	})

	return err
}

func (s store) WriteBotActivities(context context.Context, activities []*storage.BotActivity) error {
	scope.Debugf("Writing %d activities", len(activities))

	mutations := make([]*spanner.Mutation, len(activities))
	for i := 0; i < len(activities); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(botActivityTable, activities[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}
