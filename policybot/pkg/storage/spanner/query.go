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

func (s store) QueryMembersByOrg(context context.Context, orgLogin string, cb func(*storage.Member) error) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Members WHERE OrgLogin = '%s'", orgLogin)})
	err := iter.Do(func(row *spanner.Row) error {
		member := &storage.Member{}
		if err := rowToStruct(row, member); err != nil {
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
		if err := rowToStruct(row, maintainer); err != nil {
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
		if err := rowToStruct(row, issue); err != nil {
			return err
		}

		return cb(issue)
	})

	return err
}

func (s store) QueryTestResultByTestName(context context.Context, orgLogin string, repoName string, testName string, cb func(*storage.TestResult) error) error {
	sql := `SELECT * from TestResults
	WHERE OrgLogin = @orgLogin AND
	RepoName = @repoName AND
	TestName = @testName;`
	stmt := spanner.NewStatement(sql)
	stmt.Params["orgLogin"] = orgLogin
	stmt.Params["repoName"] = repoName
	stmt.Params["testName"] = testName
	iter := s.client.Single().Query(context, stmt)
	err := iter.Do(func(row *spanner.Row) error {
		testResult := &storage.TestResult{}
		if err := rowToStruct(row, testResult); err != nil {
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
		if err := rowToStruct(row, testResult); err != nil {
			return err
		}

		return cb(testResult)
	})

	return err
}

func (s store) QueryTestResultByUndone(context context.Context, orgLogin string, repoName string, cb func(*storage.TestResult) error) error {
	sql := `SELECT * from TestResults
	WHERE OrgLogin = @orgLogin AND
	RepoName = @repoName AND
	Done = false;`
	stmt := spanner.NewStatement(sql)
	stmt.Params["orgLogin"] = orgLogin
	stmt.Params["repoName"] = repoName
	iter := s.client.Single().Query(context, stmt)
	err := iter.Do(func(row *spanner.Row) error {
		testResult := &storage.TestResult{}
		if err := rowToStruct(row, testResult); err != nil {
			return err
		}

		return cb(testResult)
	})

	return err
}

// Read all rows from table in Spanner and invokes a call back on the row.
func (s store) QueryAllTestResults(context context.Context, orgLogin string, repoName string, cb func(*storage.TestResult) error) error {
	sql := `SELECT * from TestResults
	WHERE OrgLogin = @orgLogin AND
	RepoName = @repoName AND `
	stmt := spanner.NewStatement(sql)
	stmt.Params["orgLogin"] = orgLogin
	stmt.Params["repoName"] = repoName
	iter := s.client.Single().Query(context, stmt)
	err := iter.Do(func(row *spanner.Row) error {
		testResult := &storage.TestResult{}
		if err := rowToStruct(row, testResult); err != nil {
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
		if err := rowToStruct(row, &issue); err != nil {
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

func (s store) QueryMaintainerActivity(context context.Context, maintainer *storage.Maintainer) (*storage.ActivityInfo, error) {
	info := &storage.ActivityInfo{
		Repos: make(map[string]*storage.RepoActivityInfo),
	}

	// prep all the repo infos
	soughtPaths := make(map[string]map[string]bool)
	for _, mp := range maintainer.Paths {
		slashIndex := strings.Index(mp, "/")
		repoName := mp[0:slashIndex]
		path := mp[slashIndex+1:]

		repoInfo, ok := info.Repos[repoName]
		if !ok {
			repoInfo = &storage.RepoActivityInfo{
				Paths: make(map[string]storage.RepoPathActivityInfo),
			}
			info.Repos[repoName] = repoInfo
			soughtPaths[repoName] = make(map[string]bool)
		}
		repoInfo.Paths[path] = storage.RepoPathActivityInfo{}

		// track all the specific paths we care about for the repo
		soughtPaths[repoName][path] = true
	}

	// find the last time the maintainer updated files in the maintained paths
	for repoName, repoInfo := range info.Repos {
		iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
			`SELECT * FROM PullRequests
			WHERE
				OrgLogin = '%s'
				AND RepoName = '%s'
				AND Author = '%s'
			ORDER BY MergedAt DESC`,
			maintainer.OrgLogin, repoName, maintainer.UserLogin)})

		err := iter.Do(func(row *spanner.Row) error {

			var pr storage.PullRequest
			if err := rowToStruct(row, &pr); err != nil {
				return err
			}

			// if the pr affects any files in any of the maintainer's paths, update the timed entry for the path
			for sp := range soughtPaths[repoName] {
				for _, file := range pr.Files {
					if strings.HasPrefix(file, sp) {
						pai := repoInfo.Paths[sp]
						pai.LastPullRequestSubmitted = storage.TimedEntry{
							Time:   pr.MergedAt,
							Number: pr.PullRequestNumber,
						}
						repoInfo.Paths[sp] = pai

						if pr.MergedAt.After(info.LastActivity) {
							info.LastActivity = pr.MergedAt
						}

						delete(soughtPaths[repoName], sp)
						break
					}
				}
			}

			if len(soughtPaths[repoName]) == 0 {
				// all the paths for this repo have been handled, move on
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

	// reset the soughtPaths map
	for _, mp := range maintainer.Paths {
		slashIndex := strings.Index(mp, "/")
		repoName := mp[0:slashIndex]
		path := mp[slashIndex+1:]
		soughtPaths[repoName][path] = true
	}

	// find the last time the maintainer reviewed a PR that updated files in the maintained paths
	for repoName, repoInfo := range info.Repos {
		iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
			`SELECT * FROM PullRequestReviewEvents
			WHERE
				OrgLogin = '%s'
				AND RepoName = '%s'
				AND Actor = '%s'`,
			maintainer.OrgLogin, repoName, maintainer.UserLogin)})

		err := iter.Do(func(row *spanner.Row) error {

			var e storage.PullRequestReviewEvent
			if err := rowToStruct(row, &e); err != nil {
				return err
			}

			pr, err := s.ReadPullRequest(context, maintainer.OrgLogin, repoName, int(e.PullRequestNumber))
			if err != nil {
				return err
			}

			// if the pr affects any files in any of the maintainer's paths, update the timed entry for the path
			for sp := range soughtPaths[repoName] {
				for _, file := range pr.Files {
					if strings.HasPrefix(file, sp) {
						pai := repoInfo.Paths[sp]
						pai.LastPullRequestReviewed = storage.TimedEntry{
							Time:   e.CreatedAt,
							Number: pr.PullRequestNumber,
						}
						repoInfo.Paths[sp] = pai

						if e.CreatedAt.After(info.LastActivity) {
							info.LastActivity = e.CreatedAt
						}

						delete(soughtPaths[repoName], sp)
						break
					}
				}
			}

			if len(soughtPaths[repoName]) == 0 {
				// all the paths for this repo have been handled, move on
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

	// reset the soughtPaths map
	for _, mp := range maintainer.Paths {
		slashIndex := strings.Index(mp, "/")
		repoName := mp[0:slashIndex]
		path := mp[slashIndex+1:]
		soughtPaths[repoName][path] = true
	}

	// find the last time the maintainer commented on a PR that updated files in the maintained paths
	for repoName, repoInfo := range info.Repos {
		iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
			`SELECT * FROM PullRequestReviewCommentEvents
			WHERE
				OrgLogin = '%s'
				AND RepoName = '%s'
				AND Actor = '%s'`,
			maintainer.OrgLogin, repoName, maintainer.UserLogin)})

		err := iter.Do(func(row *spanner.Row) error {

			var e storage.PullRequestReviewCommentEvent
			if err := rowToStruct(row, &e); err != nil {
				return err
			}

			pr, err := s.ReadPullRequest(context, maintainer.OrgLogin, repoName, int(e.PullRequestNumber))
			if err != nil {
				return err
			} else if pr == nil {
				return nil
			}

			// if the pr affects any files in any of the maintainer's paths, update the timed entry for the path
			for sp := range soughtPaths[repoName] {
				for _, file := range pr.Files {
					if strings.HasPrefix(file, sp) {
						pai := repoInfo.Paths[sp]
						pai.LastPullRequestReviewed = storage.TimedEntry{
							Time:   e.CreatedAt,
							Number: pr.PullRequestNumber,
						}
						repoInfo.Paths[sp] = pai

						if e.CreatedAt.After(info.LastActivity) {
							info.LastActivity = e.CreatedAt
						}

						delete(soughtPaths[repoName], sp)
						break
					}
				}
			}

			if len(soughtPaths[repoName]) == 0 {
				// all the paths for this repo have been handled, move on
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

	// now figure out issue activity for all repos
	for repoName, repoInfo := range info.Repos {
		if err := s.getIssueActivity(context, maintainer.OrgLogin, repoName, maintainer.UserLogin, info, repoInfo); err != nil {
			return nil, err
		}
	}

	return info, nil
}

func (s store) QueryMemberActivity(context context.Context, member *storage.Member, repoNames []string) (*storage.ActivityInfo, error) {
	info := &storage.ActivityInfo{
		Repos: make(map[string]*storage.RepoActivityInfo),
	}

	for _, repoName := range repoNames {
		repoInfo := &storage.RepoActivityInfo{}
		repoInfo.Paths = make(map[string]storage.RepoPathActivityInfo)
		repoInfo.Paths["/"] = storage.RepoPathActivityInfo{}

		if err := s.getIssueActivity(context, member.OrgLogin, repoName, member.UserLogin, info, repoInfo); err != nil {
			return nil, err
		}

		if err := s.getPRActivity(context, member.OrgLogin, repoName, member.UserLogin, info, repoInfo); err != nil {
			return nil, err
		}

		// if any activity was detected, keep track of the repo
		if repoInfo.LastIssueCommented.Number != 0 ||
			repoInfo.LastIssueClosed.Number != 0 ||
			repoInfo.LastIssueTriaged.Number != 0 ||
			repoInfo.Paths["/"].LastPullRequestReviewed.Number != 0 ||
			repoInfo.Paths["/"].LastPullRequestSubmitted.Number != 0 {
			info.Repos[repoName] = repoInfo
		}
	}

	return info, nil
}

func (s *store) getIssueActivity(context context.Context, orgLogin string, repoName string, userLogin string,
	info *storage.ActivityInfo, repoInfo *storage.RepoActivityInfo) error {
	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT * FROM IssueCommentEvents
			WHERE
				OrgLogin = '%s' 
				AND RepoName = '%s' 
				AND Actor = '%s' 
				AND (Action = 'created'
					OR Action = 'edited')
			ORDER BY CreatedAt DESC
			LIMIT 1;`,
		orgLogin, repoName, userLogin)})
	err := iter.Do(func(row *spanner.Row) error {
		var e storage.IssueCommentEvent
		if err := rowToStruct(row, &e); err != nil {
			return err
		}

		repoInfo.LastIssueCommented = storage.TimedEntry{
			Time:   e.CreatedAt,
			Number: e.IssueNumber,
		}

		if e.CreatedAt.After(info.LastActivity) {
			info.LastActivity = e.CreatedAt
		}

		return nil
	})

	if err != nil {
		return err
	}

	iter = s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT * FROM IssueEvents
			WHERE 
				OrgLogin = '%s'
				AND RepoName = '%s'
				AND Actor = '%s'
				AND (Action = 'labeled' 
					OR Action = 'unlabaled' 
					OR Action = 'milestoned' 
					OR Action = 'unmilestoned' 
					OR Action = 'assigned'
					OR Action = 'unassigned')
			ORDER BY CreatedAt DESC
			LIMIT 1;`,
		orgLogin, repoName, userLogin)})
	err = iter.Do(func(row *spanner.Row) error {
		var e storage.IssueEvent
		if err := rowToStruct(row, &e); err != nil {
			return err
		}

		repoInfo.LastIssueTriaged = storage.TimedEntry{
			Time:   e.CreatedAt,
			Number: e.IssueNumber,
		}

		if e.CreatedAt.After(info.LastActivity) {
			info.LastActivity = e.CreatedAt
		}

		return nil
	})

	if err != nil {
		return err
	}

	iter = s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT * FROM IssueEvents
			WHERE
				OrgLogin = '%s'
				AND RepoName = '%s'
				AND Actor = '%s'
				AND Action = 'closed'
			ORDER BY CreatedAt DESC
			LIMIT 1;`,
		orgLogin, repoName, userLogin)})
	err = iter.Do(func(row *spanner.Row) error {
		var e storage.IssueEvent
		if err := rowToStruct(row, &e); err != nil {
			return err
		}

		repoInfo.LastIssueClosed = storage.TimedEntry{
			Time:   e.CreatedAt,
			Number: e.IssueNumber,
		}

		if e.CreatedAt.After(info.LastActivity) {
			info.LastActivity = e.CreatedAt
		}

		return nil
	})

	return err
}

func (s *store) getPRActivity(context context.Context, orgLogin string, repoName string, userLogin string,
	info *storage.ActivityInfo, repoInfo *storage.RepoActivityInfo) error {

	pathInfo := repoInfo.Paths["/"]

	iter := s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT * FROM PullRequestEvents
			WHERE
				OrgLogin = '%s' 
				AND RepoName = '%s' 
				AND Actor = '%s' 
				AND Action = 'closed'
			ORDER BY CreatedAt DESC
			LIMIT 1;`,
		orgLogin, repoName, userLogin)})
	err := iter.Do(func(row *spanner.Row) error {
		var e storage.PullRequestEvent
		if err := rowToStruct(row, &e); err != nil {
			return err
		}

		pathInfo.LastPullRequestSubmitted = storage.TimedEntry{
			Time:   e.CreatedAt,
			Number: e.PullRequestNumber,
		}

		if e.CreatedAt.After(info.LastActivity) {
			info.LastActivity = e.CreatedAt
		}

		return nil
	})

	if err != nil {
		return err
	}

	iter = s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT * FROM PullRequestReviewEvents
			WHERE 
				OrgLogin = '%s'
				AND RepoName = '%s'
				AND Actor = '%s'
			ORDER BY CreatedAt DESC
			LIMIT 1;`,
		orgLogin, repoName, userLogin)})
	err = iter.Do(func(row *spanner.Row) error {
		var e storage.PullRequestReviewEvent
		if err := rowToStruct(row, &e); err != nil {
			return err
		}

		pathInfo.LastPullRequestReviewed = storage.TimedEntry{
			Time:   e.CreatedAt,
			Number: e.PullRequestNumber,
		}

		if e.CreatedAt.After(info.LastActivity) {
			info.LastActivity = e.CreatedAt
		}

		return nil
	})

	if err != nil {
		return err
	}

	iter = s.client.Single().Query(context, spanner.Statement{SQL: fmt.Sprintf(
		`SELECT * FROM PullRequestReviewCommentEvents
			WHERE 
				OrgLogin = '%s'
				AND RepoName = '%s'
				AND Actor = '%s'
			ORDER BY CreatedAt DESC
			LIMIT 1;`,
		orgLogin, repoName, userLogin)})
	err = iter.Do(func(row *spanner.Row) error {
		var e storage.PullRequestReviewCommentEvent
		if err := rowToStruct(row, &e); err != nil {
			return err
		}

		if pathInfo.LastPullRequestReviewed.Time.Before(e.CreatedAt) {
			pathInfo.LastPullRequestReviewed = storage.TimedEntry{
				Time:   e.CreatedAt,
				Number: e.PullRequestNumber,
			}
		}

		if e.CreatedAt.After(info.LastActivity) {
			info.LastActivity = e.CreatedAt
		}

		return nil
	})

	repoInfo.Paths["/"] = pathInfo

	return err
}
