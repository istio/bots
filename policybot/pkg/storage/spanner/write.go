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
		if mutations[i], err = insertOrUpdateStruct(orgTable, orgs[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(repoTable, repos[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(repoCommentTable, comments[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(issueTable, issues[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(issueCommentTable, issueComments[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(pullRequestTable, prs[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePullRequestReviewComments(context context.Context, prComments []*storage.PullRequestReviewComment) error {
	scope.Debugf("Writing %d pr review comments", len(prComments))

	mutations := make([]*spanner.Mutation, len(prComments))
	for i := 0; i < len(prComments); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(pullRequestReviewCommentTable, prComments[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(pullRequestReviewTable, prReviews[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(userTable, users[i]); err != nil {
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
		if mutations[i], err = insertOrUpdateStruct(labelTable, labels[i]); err != nil {
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
		if mutations[i], err = insertStruct(memberTable, member); err != nil {
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
		if mutations[i], err = insertStruct(maintainerTable, maintainer); err != nil {
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

func (s store) WriteAllUserAffiliations(ctx1 context.Context, affiliations []*storage.UserAffiliation) error {
	scope.Debugf("Writing %d user affiliations", len(affiliations))

	mutations := make([]*spanner.Mutation, len(affiliations))
	for i, a := range affiliations {
		var err error
		if mutations[i], err = insertStruct(userAffiliationTable, a); err != nil {
			return err
		}
	}

	_, err := s.client.ReadWriteTransaction(ctx1, func(ctx2 context.Context, txn *spanner.ReadWriteTransaction) error {
		// Remove all existing affiliations
		iter := txn.Query(ctx2, spanner.Statement{SQL: "DELETE FROM UserAffiliation WHERE true;"})
		if err := iter.Do(func(_ *spanner.Row) error { return nil }); err != nil {
			return err
		}

		// write all the new members
		return txn.BufferWrite(mutations)
	})

	return err
}

func (s store) WriteBotActivities(context context.Context, activities []*storage.BotActivity) error {
	scope.Debugf("Writing %d activities", len(activities))

	mutations := make([]*spanner.Mutation, len(activities))
	for i := 0; i < len(activities); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(botActivityTable, activities[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteTestResults(context context.Context, testResults []*storage.TestResult) error {
	scope.Debugf("Writing %d test results", len(testResults))

	mutations := make([]*spanner.Mutation, len(testResults))
	for i := 0; i < len(testResults); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(testResultTable, testResults[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePostSumbitTestResults(context context.Context, postSubmitTestResults []*storage.PostSubmitTestResult) error {
	scope.Debugf("Writing %d post submit test results", len(postSubmitTestResults))

	mutations := make([]*spanner.Mutation, len(postSubmitTestResults))
	for i := 0; i < len(postSubmitTestResults); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(postSubmitTestResultTable, postSubmitTestResults[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteSuiteOutcome(context context.Context, suiteOutcomes []*storage.SuiteOutcome) error {
	scope.Debugf("Writing %d suite outcome", len(suiteOutcomes))

	mutations := make([]*spanner.Mutation, len(suiteOutcomes))
	for i := 0; i < len(suiteOutcomes); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(suiteOutcomesTable, suiteOutcomes[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteTestOutcome(context context.Context, testOutcomes []*storage.TestOutcome) error {
	scope.Debugf("Writing %d test outcome", len(testOutcomes))

	mutations := make([]*spanner.Mutation, len(testOutcomes))
	for i := 0; i < len(testOutcomes); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(testOutcomeTable, testOutcomes[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteFeatureLabel(context context.Context, featureLabels []*storage.FeatureLabel) error {
	scope.Debugf("Writing %d feature label", len(featureLabels))

	mutations := make([]*spanner.Mutation, len(featureLabels))
	for i := 0; i < len(featureLabels); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(featureLabelTable, featureLabels[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteIssueEvents(context context.Context, events []*storage.IssueEvent) error {
	scope.Debugf("Writing %d issue events", len(events))

	mutations := make([]*spanner.Mutation, len(events))
	for i := 0; i < len(events); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(issueEventTable, events[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteIssueCommentEvents(context context.Context, events []*storage.IssueCommentEvent) error {
	scope.Debugf("Writing %d issue comment events", len(events))

	mutations := make([]*spanner.Mutation, len(events))
	for i := 0; i < len(events); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(issueCommentEventTable, events[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePullRequestEvents(context context.Context, events []*storage.PullRequestEvent) error {
	scope.Debugf("Writing %d pull request events", len(events))

	mutations := make([]*spanner.Mutation, len(events))
	for i := 0; i < len(events); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(pullRequestEventTable, events[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePullRequestReviewCommentEvents(context context.Context, events []*storage.PullRequestReviewCommentEvent) error {
	scope.Debugf("Writing %d pull request review comment events", len(events))

	mutations := make([]*spanner.Mutation, len(events))
	for i := 0; i < len(events); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(pullRequestReviewCommentEventTable, events[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WritePullRequestReviewEvents(context context.Context, events []*storage.PullRequestReviewEvent) error {
	scope.Debugf("Writing %d pull request review events", len(events))

	mutations := make([]*spanner.Mutation, len(events))
	for i := 0; i < len(events); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(pullRequestReviewEventTable, events[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteRepoCommentEvents(context context.Context, events []*storage.RepoCommentEvent) error {
	scope.Debugf("Writing %d repo comment events", len(events))

	mutations := make([]*spanner.Mutation, len(events))
	for i := 0; i < len(events); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(repoCommentEventTable, events[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}

func (s store) WriteCoverageData(context context.Context, data []*storage.CoverageData) error {
	scope.Debugf("Writing %d coverage data", len(data))

	mutations := make([]*spanner.Mutation, len(data))
	for i := 0; i < len(data); i++ {
		var err error
		if mutations[i], err = insertOrUpdateStruct(coverageDataTable, data[i]); err != nil {
			return err
		}
	}

	_, err := s.client.Apply(context, mutations)
	return err
}
