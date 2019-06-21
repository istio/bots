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
	"google.golang.org/grpc/codes"

	"istio.io/bots/policybot/pkg/storage"
)

func (s store) ReadOrgByID(context context.Context, org string) (*storage.Org, error) {
	row, err := s.client.Single().ReadRow(context, orgTable, orgKey(org), orgColumns)
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

func (s store) ReadOrgByLogin(context context.Context, login string) (*storage.Org, error) {
	iter := s.client.Single().ReadUsingIndex(context, orgTable, orgLoginIndex, orgLoginKey(login), orgLoginColumns)

	var olr orgLoginRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&olr)
	})

	if olr.OrgID == "" {
		return nil, nil // not found
	} else if err != nil {
		return nil, err
	}

	return &storage.Org{
		OrgID: olr.OrgID,
		Login: olr.Login,
	}, nil
}

func (s store) ReadRepoByID(context context.Context, org string, repo string) (*storage.Repo, error) {
	row, err := s.client.Single().ReadRow(context, repoTable, repoKey(org, repo), repoColumns)
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

func (s store) ReadRepoByName(context context.Context, org string, name string) (*storage.Repo, error) {
	iter := s.client.Single().ReadUsingIndex(context, repoTable, repoNameIndex, repoNameKey(org, name), repoNameColumns)

	var rnr repoNameRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&rnr)
	})

	if rnr.OrgID == "" {
		return nil, nil // not found
	} else if err != nil {
		return nil, err
	}

	return s.ReadRepoByID(context, org, rnr.RepoID)
}

func (s store) ReadIssueByID(context context.Context, org string, repo string, issue string) (*storage.Issue, error) {
	row, err := s.client.Single().ReadRow(context, issueTable, issueKey(org, repo, issue), issueColumns)
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

func (s store) ReadIssueByNumber(context context.Context, org string, repo string, number int) (*storage.Issue, error) {
	iter := s.client.Single().ReadUsingIndex(context, issueTable, issueNumberIndex, issueNumberKey(repo, number), issueNumberColumns)

	var inr issueNumberRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&inr)
	})

	if inr.OrgID == "" {
		return nil, nil // not found
	} else if err != nil {
		return nil, err
	}

	return s.ReadIssueByID(context, org, repo, inr.IssueID)
}

func (s store) ReadIssueCommentByID(context context.Context, org string, repo string, issue string, issueComment string) (*storage.IssueComment, error) {
	row, err := s.client.Single().ReadRow(context, issueCommentTable, issueCommentKey(org, repo, issue, issueComment), issueCommentColumns)
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

func (s store) ReadIssuePipelineByNumber(context context.Context, orgID string, repoID string, number int) (*storage.IssuePipeline, error) {
	row, err := s.client.Single().ReadRow(context, issuePipelineTable, issuePipelineKey(orgID, repoID, number), issuePipelineColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.IssuePipeline
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadPullRequestByID(context context.Context, org string, repo string, issue string) (*storage.PullRequest, error) {
	row, err := s.client.Single().ReadRow(context, pullRequestTable, pullRequestKey(org, repo, issue), pullRequestColumns)
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

func (s store) ReadPullRequestCommentByID(context context.Context, orgID string, repoID string, prID string,
	prCommentID string) (*storage.PullRequestComment, error) {
	row, err := s.client.Single().ReadRow(context, pullRequestCommentTable, pullRequestCommentKey(orgID, repoID, prID, prCommentID), pullRequestCommentColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.PullRequestComment
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadPullRequestReviewByID(context context.Context, org string, repo string, issue string,
	pullRequestReview string) (*storage.PullRequestReview, error) {
	row, err := s.client.Single().ReadRow(context, pullRequestReviewTable, pullRequestReviewKey(org, repo, issue, pullRequestReview), pullRequestReviewColumns)
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

func (s store) ReadLabelByID(context context.Context, org string, repo string, label string) (*storage.Label, error) {
	row, err := s.client.Single().ReadRow(context, labelTable, labelKey(org, repo, label), labelColumns)
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

func (s store) ReadUserByID(context context.Context, user string) (*storage.User, error) {
	row, err := s.client.Single().ReadRow(context, userTable, userKey(user), userColumns)
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

func (s store) ReadUserByLogin(context context.Context, login string) (*storage.User, error) {
	iter := s.client.Single().ReadUsingIndex(context, userTable, userLoginIndex, userLoginKey(login), userLoginColumns)

	var ulr userLoginRow

	err := iter.Do(func(row *spanner.Row) error {
		return row.ToStruct(&ulr)
	})

	if ulr.UserID == "" {
		return nil, nil // not found
	} else if err != nil {
		return nil, err
	}

	return s.ReadUserByID(context, ulr.UserID)
}

func (s store) ReadBotActivityByID(context context.Context, orgID string, repoID string) (*storage.BotActivity, error) {
	row, err := s.client.Single().ReadRow(context, botActivityTable, botActivityKey(orgID, repoID), botActivityColumns)
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

func (s store) ReadMaintainerByID(context context.Context, orgID string, userID string) (*storage.Maintainer, error) {
	row, err := s.client.Single().ReadRow(context, maintainerTable, maintainerKey(orgID, userID), maintainerColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Maintainer
	if err := row.ToStruct(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
