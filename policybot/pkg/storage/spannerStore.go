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

package storage

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"

	"cloud.google.com/go/spanner"
	"github.com/google/go-github/v25/github"
)

type spannerStore struct {
	client *spanner.Client
	ctx    context.Context
}

func NewSpannerStore(ctx context.Context, database string, gcpCreds []byte) (Store, error) {
	client, err := spanner.NewClient(ctx, database, option.WithCredentialsJSON(gcpCreds))
	if err != nil {
		return nil, fmt.Errorf("unable to create Spanner client: %v", err)
	}

	return &spannerStore{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *spannerStore) Close() error {
	s.client.Close()
	return nil
}

func (s *spannerStore) WriteOrgAndRepos(org *github.Organization, repos []*github.Repository) error {
	mutations := make([]*spanner.Mutation, 1+len(repos))
	mutations[0] = spanner.InsertOrUpdate("orgs",
		[]string{"OrgID", "Login", "Name", "Company"},
		[]interface{}{org.GetNodeID(), org.GetLogin(), org.GetName(), org.GetCompany()})

	for i, repo := range repos {
		mutations[i+1] = spanner.InsertOrUpdate("repos",
			[]string{
				"OrgId",
				"RepoID",
				"Description",
				"FullName",
				"Name",
			},
			[]interface{}{
				org.GetNodeID(),
				repo.GetNodeID(),
				repo.GetDescription(),
				repo.GetFullName(),
				repo.GetName(),
			})
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *spannerStore) WriteIssueAndComments(org *github.Organization, repo *github.Repository, issue *github.Issue, comments []*github.IssueComment) error {
	assignees := make([]string, len(issue.Assignees))
	for i, assignee := range issue.Assignees {
		assignees[i] = assignee.GetNodeID()
	}

	labels := make([]string, len(issue.Labels))
	for i, label := range issue.Labels {
		labels[i] = label.GetName()
	}

	mutations := make([]*spanner.Mutation, 1+len(comments))
	mutations[0] = spanner.Replace("issues",
		[]string{
			"OrgID",
			"RepoID",
			"IssueID",
			"Number",
			"Title",
			"AuthorUserID",
			"CreatedAt",
			"UpdatedAt",
			"Body",
			"State",
			"AssigneesUserID",
			"Labels",
			"IsPullRequest",
		},
		[]interface{}{
			org.GetNodeID(),
			repo.GetNodeID(),
			issue.GetNodeID(),
			issue.GetNumber(),
			issue.GetTitle(),
			issue.GetUser().GetNodeID(),
			issue.GetCreatedAt(),
			issue.GetUpdatedAt(),
			issue.GetBody(),
			issue.GetState(),
			assignees,
			labels,
			issue.IsPullRequest(),
		})

	for i, comment := range comments {
		mutations[i+1] = spanner.Replace("issue_comments",
			[]string{
				"OrgID",
				"RepoID",
				"IssueID",
				"CommentID",
				"Body",
				"AuthorUserID",
				"CreatedAt",
				"UpdatedAt",
			},
			[]interface{}{
				org.GetNodeID(),
				repo.GetNodeID(),
				issue.GetNodeID(),
				comment.GetNodeID(),
				comment.GetBody(),
				comment.GetUser().GetNodeID(),
				comment.GetCreatedAt(),
				comment.GetUpdatedAt(),
			})
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *spannerStore) WriteUsers(users []*github.User) error {
	mutations := make([]*spanner.Mutation, len(users))
	for i, user := range users {
		mutations[i] = spanner.Replace("users",
			[]string{
				"UserID",
				"Company",
				"Login",
				"Name",
			},
			[]interface{}{
				user.GetNodeID(),
				user.GetCompany(),
				user.GetLogin(),
				user.GetName(),
			})
	}

	_, err := s.client.Apply(s.ctx, mutations)
	return err
}

func (s *spannerStore) ReadIssue(org *github.Organization, repo *github.Repository, id string) (*github.Issue, error) {
	row, err := s.client.Single().ReadRow(s.ctx, "issues",
		spanner.Key{
			org.GetNodeID(),
			repo.GetNodeID(),
			id,
		},
		[]string{"UpdatedAt"})

	if spanner.ErrCode(err) == codes.NotFound {
		// issue doesn't exist
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	issue := github.Issue{UpdatedAt: &time.Time{}}
	if err = row.Columns(issue.UpdatedAt); err != nil {
		return nil, err
	}

	return &issue, nil
}
