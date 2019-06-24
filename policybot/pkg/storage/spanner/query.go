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
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"istio.io/bots/policybot/pkg/storage"
)

func (s *store) QueryMembersByOrg(orgID string, cb func(*storage.Member) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Members WHERE OrgID = '%s'", orgID)})
	err := iter.Do(func(row *spanner.Row) error {
		member := &storage.Member{}
		if err := row.ToStruct(member); err != nil {
			return err
		}

		return cb(member)
	})

	return err
}

func (s *store) QueryMaintainersByOrg(orgID string, cb func(*storage.Maintainer) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Maintainers WHERE OrgID = '%s'", orgID)})
	err := iter.Do(func(row *spanner.Row) error {
		maintainer := &storage.Maintainer{}
		if err := row.ToStruct(maintainer); err != nil {
			return err
		}

		return cb(maintainer)
	})

	return err
}

func (s *store) QueryIssuesByRepo(orgID string, repoID string, cb func(*storage.Issue) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Issues WHERE OrgID = '%s' AND RepoID = '%s';", orgID, repoID)})
	err := iter.Do(func(row *spanner.Row) error {
		issue := &storage.Issue{}
		if err := row.ToStruct(issue); err != nil {
			return err
		}

		return cb(issue)
	})

	return err
}

func (s *store) QueryAllUsers(cb func(*storage.User) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: "SELECT * FROM Users;"})
	err := iter.Do(func(row *spanner.Row) error {
		user := &storage.User{}
		if err := row.ToStruct(user); err != nil {
			return err
		}

		return cb(user)
	})

	return err
}

func (s *store) QueryTestFlakeByTestName(testName string, cb func(*storage.TestFlake) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM TestFlakes WHERE TestName = '%s'", testName)})
	err := iter.Do(func(row *spanner.Row) error {
		flake := &storage.TestFlake{}
		if err := row.ToStruct(flake); err != nil {
			return err
		}

		if err := cb(flake); err != nil {
			iter.Stop()
			return err
		}

		return nil
	})

	return err
}

func (s *store) QueryTestFlakeByPrNumber(prNum int64, cb func(*storage.TestFlake) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM TestFlakes WHERE PrNum = '%v'", prNum)})
	err := iter.Do(func(row *spanner.Row) error {
		flake := &storage.TestFlake{}
		if err := row.ToStruct(flake); err != nil {
			return err
		}

		if err := cb(flake); err != nil {
			iter.Stop()
			return err
		}

		return nil
	})

	return err
}

func (s *store) QueryMaintainerInfo(maintainer *storage.Maintainer) (*storage.MaintainerInfo, error) {
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
		iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf(
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
