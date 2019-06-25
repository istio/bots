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
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"istio.io/bots/policybot/pkg/storage"
)

func (s store) QueryMembersByOrg(context context.Context, orgID string, cb func(*storage.Member) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Members WHERE OrgID = '%s'", orgID)})
	err := iter.Do(func(row *spanner.Row) error {
		member := &storage.Member{}
		if err := row.ToStruct(member); err != nil {
			return err
		}

		return cb(member)
	})

	return err
}

func (s store) QueryMaintainersByOrg(context context.Context, orgID string, cb func(*storage.Maintainer) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Maintainers WHERE OrgID = '%s'", orgID)})
	err := iter.Do(func(row *spanner.Row) error {
		maintainer := &storage.Maintainer{}
		if err := row.ToStruct(maintainer); err != nil {
			return err
		}

		return cb(maintainer)
	})

	return err
}

func (s store) QueryIssuesByRepo(context context.Context, orgID string, repoID string, cb func(*storage.Issue) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Issues WHERE OrgID = '%s' AND RepoID = '%s';", orgID, repoID)})
	err := iter.Do(func(row *spanner.Row) error {
		issue := &storage.Issue{}
		if err := row.ToStruct(issue); err != nil {
			return err
		}

		return cb(issue)
	})

	return err
}

func (s store) QueryTestFlakeIssues(context context.Context, inactiveDays, createdDays int) ([]*storage.Issue, error) {
	sql := `SELECT * from Issues
	WHERE TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), UpdatedAt, DAY) > @inactiveDays AND 
				TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), CreatedAt, DAY) < @createdDays AND
				State = 'open' AND
				( REGEXP_CONTAINS(title, 'flak[ey]') OR 
  				  REGEXP_CONTAINS(body, 'flake[ey]')
				);`
	stmt := spanner.NewStatement(sql)
	stmt.Params["inactiveDays"] = inactiveDays
	stmt.Params["createdDays"] = createdDays
	scope.Infof("QueryTestFlakeIssues SQL\n%v", stmt.SQL)
	var issues []*storage.Issue
	getIssue := func(row *spanner.Row) error {
		issue := storage.Issue{}
		if err := row.ToStruct(&issue); err != nil {
			return err
		}
		issues = append(issues, &issue)
		return nil
	}
	iter := s.client.Single().Query(context, stmt)
	if err := iter.Do(getIssue); err != nil {
		return nil, fmt.Errorf("error in fetching flaky test issues, %v", err)
	}
	return issues, nil
}

func (s store) QueryMaintainerInfo(context context.Context, maintainer *storage.Maintainer) (*storage.MaintainerInfo, error) {
	info := &storage.MaintainerInfo{
		Repos: make(map[string]*storage.RepoActivityInfo),
	}

	// prep all the repo infos
	soughtPaths := make(map[string]map[string]bool)
	for _, mp := range maintainer.Paths {
		slashIndex := strings.Index(mp, "/")
		repoID := mp[0:slashIndex]
		path := mp[slashIndex+1:]

		repoInfo, ok := info.Repos[repoID]
		if !ok {
			repoInfo = &storage.RepoActivityInfo{
				RepoID:                         repoID,
				LastPullRequestCommittedByPath: make(map[string]storage.TimedEntry),
			}
			info.Repos[repoID] = repoInfo
			soughtPaths[repoID] = make(map[string]bool)
		}
		repoInfo.LastPullRequestCommittedByPath[path] = storage.TimedEntry{}

		// track all the specific paths we care about for the repo
		soughtPaths[repoID][path] = true
	}

	for repoID, repoInfo := range info.Repos {
		iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
			"SELECT * FROM PullRequests WHERE OrgID = '%s' AND RepoID = '%s' AND AuthorID = '%s'",
			maintainer.OrgID, repoID, maintainer.UserID)})

		err := iter.Do(func(row *spanner.Row) error {

			var pr storage.PullRequest
			if err := row.ToStruct(&pr); err != nil {
				return err
			}

			// if the pr affects any files in any of the maintainer's paths, update the timed entry for the path
			for sp := range soughtPaths[repoID] {
				for _, file := range pr.Files {
					if strings.HasPrefix(file, sp) {
						repoInfo.LastPullRequestCommittedByPath[sp] = storage.TimedEntry{
							Time: pr.MergedAt,
							ID:   pr.PullRequestID,
						}
						delete(soughtPaths[repoID], sp)
						break
					}
				}
			}

			if len(soughtPaths[repoID]) == 0 {
				// all the path for this repo have been handled, move on
				//				fmt.Printf("All sought paths have been found\n")
				return iterator.Done
			}

			return nil
		})

		if err == iterator.Done {
			err = nil
		}

		if err != nil {
			return nil, err
		}
	}

	return info, nil
}
