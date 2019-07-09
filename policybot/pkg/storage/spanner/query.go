// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.orgLogin/licenses/LICENSE-2.0
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

func (s store) QueryMembersByOrg(context context.Context, orgLogin string, cb func(*storage.Member) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Members WHERE OrgLogin = '%s'", orgLogin)})
	err := iter.Do(func(row *spanner.Row) error {
		member := &storage.Member{}
		if err := row.ToStruct(member); err != nil {
			return err
		}

		return cb(member)
	})

	return err
}

func (s store) QueryMaintainersByOrg(context context.Context, orgLogin string, cb func(*storage.Maintainer) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Maintainers WHERE OrgLogin = '%s'", orgLogin)})
	err := iter.Do(func(row *spanner.Row) error {
		maintainer := &storage.Maintainer{}
		if err := row.ToStruct(maintainer); err != nil {
			return err
		}

		return cb(maintainer)
	})

	return err
}

func (s store) QueryIssuesByRepo(context context.Context, orgLogin string, repoName string, cb func(*storage.Issue) error) error {
	iter := s.client.Single().Query(context,
		spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Issues WHERE OrgLogin = '%s' AND RepoName = '%s';", orgLogin, repoName)})
	err := iter.Do(func(row *spanner.Row) error {
		issue := &storage.Issue{}
		if err := row.ToStruct(issue); err != nil {
			return err
		}

		return cb(issue)
	})

	return err
}

func (s store) QueryTestResultByName(context context.Context, testName string, cb func(*storage.TestResult) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM TestResults WHERE TestName = '%s';", testName)})
	err := iter.Do(func(row *spanner.Row) error {
		testResult := &storage.TestResult{}
		if err := row.ToStruct(testResult); err != nil {
			return err
		}

		return cb(testResult)
	})

	return err
}

func (s store) QueryTestResultByPrNumber(
	context context.Context, orgLogin string, repoName string, pullRequestNumber int64, cb func(*storage.TestResult) error) error {
	sql := `SELECT * from TestResults
	WHERE OrgLogin = @orgLogin AND 
	RepoName = @repoName AND 
	PullRequestNumber = @pullRequestNumber;`
	stmt := spanner.NewStatement(sql)
	stmt.Params["orgLogin"] = orgLogin
	stmt.Params["repoName"] = repoName
	stmt.Params["pullRequestNumber"] = pullRequestNumber
	scope.Infof("QueryTestResults SQL\n%v", stmt.SQL)

	iter := s.client.Single().Query(context, stmt)
	err := iter.Do(func(row *spanner.Row) error {
		testResult := &storage.TestResult{}
		if err := row.ToStruct(testResult); err != nil {
			return err
		}

		return cb(testResult)
	})

	return err
}

func (s store) QueryTestResultByUndone(context context.Context, cb func(*storage.TestResult) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM TestResults WHERE Done = false")})
	err := iter.Do(func(row *spanner.Row) error {
		testResult := &storage.TestResult{}
		if err := row.ToStruct(testResult); err != nil {
			return err
		}

		return cb(testResult)
	})

	return err
}

// Real all rows from table in Spanner and invokes a call back on the row.
func (s store) QueryAllTestResults(context context.Context, cb func(*storage.TestResult) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM TestResults")})
	err := iter.Do(func(row *spanner.Row) error {
		testResult := &storage.TestResult{}
		if err := row.ToStruct(testResult); err != nil {
			return err
		}

		return cb(testResult)
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

	// prep all the repoName infos
	soughtPaths := make(map[string]map[string]bool)
	for _, mp := range maintainer.Paths {
		slashIndex := strings.Index(mp, "/")
		repoName := mp[0:slashIndex]
		path := mp[slashIndex+1:]

		repoInfo, ok := info.Repos[repoName]
		if !ok {
			repoInfo = &storage.RepoActivityInfo{
				RepoName:                       repoName,
				LastPullRequestCommittedByPath: make(map[string]storage.TimedEntry),
			}
			info.Repos[repoName] = repoInfo
			soughtPaths[repoName] = make(map[string]bool)
		}
		repoInfo.LastPullRequestCommittedByPath[path] = storage.TimedEntry{}

		// track all the specific paths we care about for the repoName
		soughtPaths[repoName][path] = true
	}

	for repoName, repoInfo := range info.Repos {
		iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
			"SELECT * FROM PullRequests WHERE OrgLogin = '%s' AND RepoName = '%s' AND Author = '%s'",
			maintainer.OrgLogin, repoName, maintainer.UserLogin)})

		err := iter.Do(func(row *spanner.Row) error {

			var pr storage.PullRequest
			if err := row.ToStruct(&pr); err != nil {
				return err
			}

			// if the pr affects any files in any of the maintainer's paths, update the timed entry for the path
			for sp := range soughtPaths[repoName] {
				for _, file := range pr.Files {
					if strings.HasPrefix(file, sp) {
						repoInfo.LastPullRequestCommittedByPath[sp] = storage.TimedEntry{
							Time: pr.MergedAt,
							ID:   pr.PullRequestNumber,
						}
						delete(soughtPaths[repoName], sp)
						break
					}
				}
			}

			if len(soughtPaths[repoName]) == 0 {
				// all the paths for this repo have been handled, move on
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
