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

package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/storage"
)

func (s *Syncer) fetchOrgs(context context.Context, cb func(organization *github.Organization) error) error {
	for _, o := range s.orgs {
		org, _, err := s.ght.Get(context).Organizations.Get(context, o.Name)
		if err != nil {
			return fmt.Errorf("unable to get information for org %s: %v", o.Name, err)
		}

		if err = cb(org); err != nil {
			return err
		}
	}

	return nil
}

func (s *Syncer) fetchRepos(context context.Context, cb func(repo *github.Repository) error) error {
	for _, o := range s.orgs {
		for _, r := range o.Repos {
			repo, _, err := s.ght.Get(context).Repositories.Get(context, o.Name, r.Name)
			if err != nil {
				return fmt.Errorf("unable to get information for repo %s/%s: %v", o.Name, r.Name, err)
			}

			if err = cb(repo); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Syncer) fetchRepoComments(context context.Context, org *storage.Org, repo *storage.Repo, cb func([]*github.RepositoryComment) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		comments, resp, err := s.ght.Get(context).Repositories.ListComments(context, org.Login, repo.Name, opt)
		if err != nil {
			return fmt.Errorf("unable to list comments for repo %s/%s: %v", org.Login, repo.Name, err)
		}

		if err := cb(comments); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}

func (s *Syncer) fetchMembers(context context.Context, org *storage.Org, cb func([]*github.User) error) error {
	opt := &github.ListMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		members, resp, err := s.ght.Get(context).Organizations.ListMembers(context, org.Login, opt)
		if err != nil {
			return fmt.Errorf("unable to list members of org %s: %v", org.Login, err)
		}

		// sadly, member info doesn't include the user name, so fetch the full user data explicitly
		for i := range members {
			if u, _, err := s.ght.Get(context).Users.Get(context, members[i].GetLogin()); err == nil {
				members[i] = u
			}
		}

		if err = cb(members); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func (s *Syncer) fetchLabels(context context.Context, org *storage.Org, repo *storage.Repo, cb func([]*github.Label) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		labels, resp, err := s.ght.Get(context).Issues.ListLabels(context, org.Login, repo.Name, opt)
		if err != nil {
			return fmt.Errorf("unable to list all labels in repo %s/%s: %v", org.Login, repo.Name, err)
		}

		if err := cb(labels); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}

func (s *Syncer) fetchIssues(context context.Context, org *storage.Org, repo *storage.Repo, startTime time.Time, cb func([]*github.Issue) error) error {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	total := 0
	for {
		issues, resp, err := s.ght.Get(context).Issues.ListByRepo(context, org.Login, repo.Name, opt)
		if err != nil {
			return fmt.Errorf("unable to list all issues in repo %s/%s: %v", org.Login, repo.Name, err)
		}

		if err := cb(issues); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.ListOptions.Page = resp.NextPage

		total += len(issues)
		scope.Infof("Synced %d issues in repo %s/%s", total, org.Login, repo.Name)
	}
}

func (s *Syncer) fetchIssueComments(context context.Context, org *storage.Org, repo *storage.Repo, issueNumber int, startTime time.Time,
	cb func([]*github.IssueComment) error) error {
	opt := &github.IssueListCommentsOptions{
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := s.ght.Get(context).Issues.ListComments(context, org.Login, repo.Name, issueNumber, opt)
		if err != nil {
			return fmt.Errorf("unable to list comments for issue %d in repo %s/%s: %v", issueNumber, org.Login, repo.Name, err)
		}

		if err := cb(comments); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func (s *Syncer) fetchFiles(context context.Context, org *storage.Org, repo *storage.Repo, prNumber int, cb func([]string) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		files, resp, err := s.ght.Get(context).PullRequests.ListFiles(context, org.Login, repo.Name, prNumber, opt)
		if err != nil {
			return fmt.Errorf("unable to list files for pull request %d in repo %s/%s: %v", prNumber, org.Login, repo.Name, err)
		}

		var result []string
		for _, f := range files {
			result = append(result, f.GetFilename())
		}

		if err = cb(result); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}

func (s *Syncer) fetchPullRequests(context context.Context, org *storage.Org, repo *storage.Repo, cb func([]*github.PullRequest) error) error {
	opt := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	total := 0
	for {
		prs, resp, err := s.ght.Get(context).PullRequests.List(context, org.Login, repo.Name, opt)
		if err != nil {
			return fmt.Errorf("unable to list pull requests in repo %s/%s: %v", org.Login, repo.Name, err)
		}

		if err := cb(prs); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage

		total += len(prs)
		scope.Infof("Synced %d pull requests in repo %s/%s", total, org.Login, repo.Name)
	}

	return nil
}

func (s *Syncer) fetchReviews(context context.Context, org *storage.Org, repo *storage.Repo, prNumber int, cb func([]*github.PullRequestReview) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		reviews, resp, err := s.ght.Get(context).PullRequests.ListReviews(context, org.Login, repo.Name, prNumber, opt)
		if err != nil {
			return fmt.Errorf("unable to list comments for pr %d in repo %s/%s: %v", prNumber, org.Login, repo.Name, err)
		}

		if err = cb(reviews); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}
