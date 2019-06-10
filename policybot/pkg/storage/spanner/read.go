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
	"google.golang.org/grpc/codes"

	"cloud.google.com/go/spanner"

	"istio.io/bots/policybot/pkg/storage"
)

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
	iter := s.client.Single().ReadUsingIndex(s.ctx, issueTable, issueNumberIndex, issueNumberKey(repo, number), issueNumberColumns)

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

func (s *store) ReadUserByLogin(login string) (*storage.User, error) {
	iter := s.client.Single().ReadUsingIndex(s.ctx, userTable, userLoginIndex, userLoginKey(login), userLoginColumns)

	var ulr userLoginRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&ulr)
	})

	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return s.ReadUserByID(ulr.UserID)
}

func (s *store) ReadBotActivityByID(orgID string, repoID string) (*storage.BotActivity, error) {
	row, err := s.client.Single().ReadRow(s.ctx, botActivityTable, botActivityKey(orgID, repoID), botActivityColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.BotActivity
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
