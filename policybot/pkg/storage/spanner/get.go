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
	"time"

	"cloud.google.com/go/spanner"
)

type getActivityResults struct {
	LastIssueEvent time.Time
	Actor          string
}

type getCommentResults struct {
	LastIssueCommentEvent time.Time
	Actor                 string
}

func (s store) GetLatestIssueMemberActivity(context context.Context, orgLogin string, repoName string, issueNumber int) (time.Time, error) {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT CreatedAt as LastIssueEvent, Actor FROM
		(SELECT CreatedAt, Actor FROM IssueEvents WHERE IssueNumber = %d AND OrgLogin = "%s" AND RepoName = "%s"
			UNION ALL
			SELECT CreatedAt, Actor FROM IssueCommentEvents WHERE IssueNumber = %d AND OrgLogin="%s" AND RepoName = "%s")
			LEFT JOIN Members ON Actor = UserLogin
			WHERE OrgLogin is not null
			ORDER BY LastIssueEvent DESC
			LIMIT 1;`, issueNumber, orgLogin, repoName, issueNumber, orgLogin, repoName)})

	var result getActivityResults
	err := iter.Do(func(row *spanner.Row) error {
		return rowToStruct(row, &result)
	})

	return result.LastIssueEvent, err
}

func (s store) GetLatestIssueMemberComment(context context.Context, orgLogin string, repoName string, issueNumber int) (time.Time, error) {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT CreatedAt as LastIssueCommentEvent, Actor FROM
		(SELECT CreatedAt, Actor FROM IssueCommentEvents WHERE IssueNumber = %d AND OrgLogin = "%s" AND RepoName = "%s")
		LEFT JOIN Members ON Actor = UserLogin
		WHERE OrgLogin is not null
		ORDER BY LastIssueCommentEvent DESC
		LIMIT 1;`, issueNumber, orgLogin, repoName)})

	var result getCommentResults
	var result2 getCommentResults

	err := iter.Do(func(row *spanner.Row) error {
		return rowToStruct(row, &result)
	})

	if err == nil {
		iter = s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
			`SELECT CreatedAt as LastIssueCommentEvent, Actor FROM
			(SELECT CreatedAt, Actor FROM PullRequestReviewEvents WHERE PullRequestNumber = %d AND OrgLogin = "%s" AND RepoName = "%s")
			LEFT JOIN Members ON Actor = UserLogin
			WHERE OrgLogin is not null
			ORDER BY LastIssueCommentEvent DESC
			LIMIT 1;`, issueNumber, orgLogin, repoName)})

		err = iter.Do(func(row *spanner.Row) error {
			return rowToStruct(row, &result2)
		})

		if err == nil {
			if result2.LastIssueCommentEvent.After(result.LastIssueCommentEvent) {
				result = result2
			}
		}
	}

	if err == nil {
		iter = s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
			`SELECT CreatedAt as LastIssueCommentEvent, Actor FROM
			(SELECT CreatedAt, Actor FROM PullRequestReviewCommentEvents WHERE PullRequestNumber = %d AND OrgLogin = "%s" AND RepoName = "%s")
			LEFT JOIN Members ON Actor = UserLogin
			WHERE OrgLogin is not null
			ORDER BY LastIssueCommentEvent DESC
			LIMIT 1;`, issueNumber, orgLogin, repoName)})

		err = iter.Do(func(row *spanner.Row) error {
			return rowToStruct(row, &result2)
		})

		if err == nil {
			if result2.LastIssueCommentEvent.After(result.LastIssueCommentEvent) {
				result = result2
			}
		}
	}

	return result.LastIssueCommentEvent, err
}
