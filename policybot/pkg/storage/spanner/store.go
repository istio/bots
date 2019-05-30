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
	"fmt"

	"istio.io/pkg/log"

	"google.golang.org/grpc/codes"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"istio.io/bots/policybot/pkg/storage"
)

type store struct {
	client *spanner.Client
	ctx    context.Context
}

var scope = log.RegisterScope("spanner", "Spanner abstraction layer", 0)

func NewStore(ctx context.Context, database string, gcpCreds []byte) (storage.Store, error) {
	client, err := spanner.NewClient(ctx, database, option.WithCredentialsJSON(gcpCreds))
	if err != nil {
		return nil, fmt.Errorf("unable to create Spanner client: %v", err)
	}

	return &store{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *store) Close() error {
	s.client.Close()
	return nil
}

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

func (s *store) ReadOrgByID(org string) (*storage.Org, error) {
	row, err := s.client.Single().ReadRow(s.ctx, orgTable, orgKey(org), orgColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Org
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) ReadOrgByLogin(login string) (*storage.Org, error) {
	iter := s.client.Single().ReadUsingIndex(s.ctx, orgTable, orgLoginIndex, orgLoginKey(login), orgLoginColumns)

	var olr orgLoginRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&olr)
	})

	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &storage.Org{
		OrgID: olr.OrgID,
		Login: olr.Login,
	}, nil
}

func (s *store) ReadRepoByID(org string, repo string) (*storage.Repo, error) {
	row, err := s.client.Single().ReadRow(s.ctx, repoTable, repoKey(org, repo), repoColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Repo
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) ReadRepoByName(org string, name string) (*storage.Repo, error) {
	iter := s.client.Single().ReadUsingIndex(s.ctx, repoTable, repoNameIndex, repoNameKey(org, name), repoNameColumns)

	var rnr repoNameRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&rnr)
	})

	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return s.ReadRepoByID(org, rnr.RepoID)
}

func (s *store) ReadIssueByID(org string, repo string, issue string) (*storage.Issue, error) {
	row, err := s.client.Single().ReadRow(s.ctx, issueTable, issueKey(org, repo, issue), issueColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Issue
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) ReadIssueByNumber(org string, repo string, number int) (*storage.Issue, error) {
	iter := s.client.Single().ReadUsingIndex(s.ctx, issueTable, issueNumberIndex, issueNumberKey(org, repo, number), issueNumberColumns)

	var inr issueNumberRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&inr)
	})

	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return s.ReadIssueByID(org, repo, inr.IssueID)
}

func (s *store) ReadIssueCommentByID(org string, repo string, issue string, issueComment string) (*storage.IssueComment, error) {
	row, err := s.client.Single().ReadRow(s.ctx, issueCommentTable, issueCommentKey(org, repo, issue, issueComment), issueCommentColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.IssueComment
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) ReadPullRequestByID(org string, repo string, issue string) (*storage.PullRequest, error) {
	row, err := s.client.Single().ReadRow(s.ctx, pullRequestTable, pullRequestKey(org, repo, issue), pullRequestColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.PullRequest
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) ReadPullRequestReviewByID(org string, repo string, issue string, pullRequestReview string) (*storage.PullRequestReview, error) {
	row, err := s.client.Single().ReadRow(s.ctx, pullRequestReviewTable, pullRequestReviewKey(org, repo, issue, pullRequestReview), pullRequestReviewColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.PullRequestReview
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) ReadLabelByID(org string, repo string, label string) (*storage.Label, error) {
	row, err := s.client.Single().ReadRow(s.ctx, labelTable, labelKey(org, repo, label), labelColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Label
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) ReadUserByID(user string) (*storage.User, error) {
	row, err := s.client.Single().ReadRow(s.ctx, userTable, userKey(user), userColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.User
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *store) updateStats(repo string, cb func(rsr *repoStatsRow)) error {
	_, err := s.client.ReadWriteTransaction(s.ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, repoStatsTable, repoStatsKey(repo), repoStatsColumns)
		if err != nil {
			return err
		}

		var rsr repoStatsRow
		err = row.ToStruct(&rsr)
		if err != nil {
			return err
		}

		cb(&rsr)

		m, err := spanner.UpdateStruct(repoStatsTable, &rsr)
		if err != nil {
			return err
		}

		return txn.BufferWrite([]*spanner.Mutation{m})
	})

	return err
}

func (s *store) RecordTestNagAdded(repo string) error {
	return s.updateStats(repo, func(rsr *repoStatsRow) {
		rsr.NagsAdded++
	})
}

func (s *store) RecordTestNagRemoved(repo string) error {
	return s.updateStats(repo, func(rsr *repoStatsRow) {
		rsr.NagsRemoved++
	})
}

func (s *store) ReadIssueBySQL(sql string, issueProcessor storage.IssueIterator) error {
	fmt.Println("jianfeih debugging invoke spanner.")
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: sql})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		// fmt.Println("jianfieh debug row ", row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println("jianfeih debug not exists")
		}
		if err := issueProcessor(row); err != nil {
			fmt.Printf("stop reading rows %v\n", err)
			return err
		}
	}
	return nil
}

/*


//SELECT * FROM issues
//WHERE (ARRAY_LENGTH(AssigneesUserID) = 0)AND (RepoID = 'MDEwOlJlcG9zaXRvcnk4MjcwNjk3Ng==') AND (State="open");

func (s *store) FindUnengagedIssues(repo *gh.Repo, cb func(issue *gh.Issue)) error {
	sql := fmt.Sprintf("SELECT * FROM Issues WHERE (ARRAY_LENGTH(AssigneesUserID) = 0) AND (State = 'open');")
	stmt := spanner.Statement{SQL: sql}
	iter := s.client.Single().Query(s.ctx, stmt)
	return iter.Do(func(row *spanner.Row) error {
		issue, err := s.getIssue(repo, row)
		if err != nil {
			return err
		}
		cb(issue)
		return nil
	})
}
*/
