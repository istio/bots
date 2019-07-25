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

func (s store) ReadOrg(context context.Context, orgLogin string) (*storage.Org, error) {
	row, err := s.client.Single().ReadRow(context, orgTable, orgKey(orgLogin), orgColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Org
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadRepo(context context.Context, orgLogin string, repoName string) (*storage.Repo, error) {
	row, err := s.client.Single().ReadRow(context, repoTable, repoKey(orgLogin, repoName), repoColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Repo
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadIssue(context context.Context, orgLogin string, repoName string, issueNumber int) (*storage.Issue, error) {
	row, err := s.client.Single().ReadRow(context, issueTable, issueKey(orgLogin, repoName, int64(issueNumber)), issueColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Issue
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadIssueComment(context context.Context, orgLogin string, repoName string, issueNumber int, issueCommentID int) (*storage.IssueComment, error) {
	row, err := s.client.Single().ReadRow(context, issueCommentTable, issueCommentKey(orgLogin, repoName, int64(issueNumber), int64(issueCommentID)),
		issueCommentColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.IssueComment
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadIssuePipeline(context context.Context, orgLogin string, repoName string, issueNumber int) (*storage.IssuePipeline, error) {
	row, err := s.client.Single().ReadRow(context, issuePipelineTable, issuePipelineKey(orgLogin, repoName, int64(issueNumber)), issuePipelineColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.IssuePipeline
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadPullRequest(context context.Context, orgLogin string, repoName string, prNumber int) (*storage.PullRequest, error) {
	row, err := s.client.Single().ReadRow(context, pullRequestTable, pullRequestKey(orgLogin, repoName, int64(prNumber)), pullRequestColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.PullRequest
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadPullRequestReviewComment(context context.Context, orgLogin string, repoName string, prNumber int,
	prCommentID int) (*storage.PullRequestReviewComment, error) {
	row, err := s.client.Single().ReadRow(context, pullRequestReviewCommentTable, pullRequestReviewCommentKey(orgLogin, repoName,
		int64(prNumber), int64(prCommentID)), pullRequestReviewCommentColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.PullRequestReviewComment
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadPullRequestReview(context context.Context, orgLogin string, repoName string, issueNumber int,
	pullRequestReviewID int) (*storage.PullRequestReview, error) {
	row, err := s.client.Single().ReadRow(context, pullRequestReviewTable, pullRequestReviewKey(orgLogin, repoName,
		int64(issueNumber), int64(pullRequestReviewID)), pullRequestReviewColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.PullRequestReview
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadLabel(context context.Context, orgLogin string, repoName string, labelName string) (*storage.Label, error) {
	row, err := s.client.Single().ReadRow(context, labelTable, labelKey(orgLogin, repoName, labelName), labelColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Label
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadUser(context context.Context, userLogin string) (*storage.User, error) {
	row, err := s.client.Single().ReadRow(context, userTable, userKey(userLogin), userColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.User
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadBotActivity(context context.Context, orgLogin string, repoName string) (*storage.BotActivity, error) {
	row, err := s.client.Single().ReadRow(context, botActivityTable, botActivityKey(orgLogin, repoName), botActivityColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.BotActivity
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadTestResult(context context.Context, orgLogin string,
	repoName string, testName string, pullRequestNumber int64, runNum int64) (*storage.TestResult, error) {
	row, err := s.client.Single().ReadRow(context, testResultTable, testResultKey(orgLogin, repoName, testName, pullRequestNumber, runNum), testResultColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var result storage.TestResult
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadMaintainer(context context.Context, orgLogin string, userLogin string) (*storage.Maintainer, error) {
	row, err := s.client.Single().ReadRow(context, maintainerTable, maintainerKey(orgLogin, userLogin), maintainerColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Maintainer
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s store) ReadMember(context context.Context, orgLogin string, userLogin string) (*storage.Member, error) {
	row, err := s.client.Single().ReadRow(context, memberTable, memberKey(orgLogin, userLogin), memberColumns)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var result storage.Member
	if err := rowToStruct(row, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
