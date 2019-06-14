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

func (s *store) WriteOrgs(orgs []*storage.Org) error {
	scope.Debugf("Writing %d orgs", len(orgs))

	mutations := make([]*spanner.Mutation, len(orgs))
	for i := 0; i < len(orgs); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(orgTable, orgs[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteRepos(repos []*storage.Repo) error {
	scope.Debugf("Writing %d repos", len(repos))

	mutations := make([]*spanner.Mutation, len(repos))
	for i := 0; i < len(repos); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(repoTable, repos[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteIssues(issues []*storage.Issue) error {
	scope.Debugf("Writing %d issues", len(issues))

	mutations := make([]*spanner.Mutation, len(issues))
	for i := 0; i < len(issues); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(issueTable, issues[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteIssueComments(issueComments []*storage.IssueComment) error {
	scope.Debugf("Writing %d issue comments", len(issueComments))

	mutations := make([]*spanner.Mutation, len(issueComments))
	for i := 0; i < len(issueComments); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(issueCommentTable, issueComments[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteIssuePipelines(issuePipelines []*storage.IssuePipeline) error {
	scope.Debugf("Writing %d issue pipelines", len(issuePipelines))

	mutations := make([]*spanner.Mutation, len(issuePipelines))
	for i := 0; i < len(issuePipelines); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(issuePipelineTable, issuePipelines[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WritePullRequests(prs []*storage.PullRequest) error {
	scope.Debugf("Writing %d pull requests", len(prs))

	mutations := make([]*spanner.Mutation, len(prs))
	for i := 0; i < len(prs); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(pullRequestTable, prs[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WritePullRequestComments(prComments []*storage.PullRequestComment) error {
	scope.Debugf("Writing %d pr comments", len(prComments))

	mutations := make([]*spanner.Mutation, len(prComments))
	for i := 0; i < len(prComments); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(pullRequestCommentTable, prComments[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WritePullRequestReviews(prReviews []*storage.PullRequestReview) error {
	scope.Debugf("Writing %d pull request reviews", len(prReviews))

	mutations := make([]*spanner.Mutation, len(prReviews))
	for i := 0; i < len(prReviews); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(pullRequestReviewTable, prReviews[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteUsers(users []*storage.User) error {
	scope.Debugf("Writing %d users", len(users))

	mutations := make([]*spanner.Mutation, len(users))
	for i := 0; i < len(users); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(userTable, users[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteLabels(labels []*storage.Label) error {
	scope.Debugf("Writing %d labels", len(labels))

	mutations := make([]*spanner.Mutation, len(labels))
	for i := 0; i < len(labels); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(labelTable, labels[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteAllMembers(members []*storage.Member) error {
	scope.Debugf("Writing %d members", len(members))

	mutations := make([]*spanner.Mutation, len(members))
	for i, member := range members {
		var err error
		if mutations[i], err = spanner.InsertStruct(memberTable, member); err != nil {
			return err
		}
	}

	_, err := s.client.ReadWriteTransaction(s.ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Remove all existing members
		iter := txn.Query(ctx, spanner.Statement{SQL: "DELETE FROM Members WHERE true;"})
		if err := iter.Do(func(_ *spanner.Row) error { return nil }); err != nil {
			return err
		}

		// write all the new members
		return txn.BufferWrite(mutations)
	})

	return err
}

func (s *store) WriteAllMaintainers(maintainers []*storage.Maintainer) error {
	scope.Debugf("Writing %d maintainers", len(maintainers))

	mutations := make([]*spanner.Mutation, len(maintainers))
	for i, maintainer := range maintainers {
		var err error
		if mutations[i], err = spanner.InsertStruct(maintainerTable, maintainer); err != nil {
			return err
		}
	}

	_, err := s.client.ReadWriteTransaction(s.ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Remove all existing maintainers
		iter := txn.Query(ctx, spanner.Statement{SQL: "DELETE FROM Maintainers WHERE true;"})
		if err := iter.Do(func(_ *spanner.Row) error { return nil }); err != nil {
			return err
		}

		// write all the new maintainers
		return txn.BufferWrite(mutations)
	})

	return err
}

func (s *store) WriteBotActivities(activities []*storage.BotActivity) error {
	scope.Debugf("Writing %d activities", len(activities))

	mutations := make([]*spanner.Mutation, len(activities))
	for i := 0; i < len(activities); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(botActivityTable, activities[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteTestFlakes(flakes []*storage.TestFlake) error {
	scope.Debugf("Writing %d test flakes", len(flakes))

	mutations := make([]*spanner.Mutation, len(flakes))
	for i := 0; i < len(flakes); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(flakeTable, flakes[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *store) WriteFlakeOccurrences(flakeOccurrences []*storage.FlakeOccurrence) error {
	scope.Debugf("Writing %d test flake occurrences", len(flakeOccurrences))

	mutations := make([]*spanner.Mutation, len(flakeOccurrences))
	for i := 0; i < len(flakeOccurrences); i++ {
		var err error
		if mutations[i], err = spanner.InsertOrUpdateStruct(flakeOccurrenceTable, flakeOccurrences[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}
