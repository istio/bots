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
		org, _, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Organizations.Get(context, o.Name)
		})

		if err != nil {
			return fmt.Errorf("unable to get information for org %s: %v", o.Name, err)
		}

		if err = cb(org.(*github.Organization)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Syncer) fetchRepos(context context.Context, cb func(repo *github.Repository) error) error {
	for _, o := range s.orgs {
		for _, r := range o.Repos {
			repo, _, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
				return client.Repositories.Get(context, o.Name, r.Name)
			})

			if err != nil {
				return fmt.Errorf("unable to get information for repo %s/%s: %v", o.Name, r.Name, err)
			}

			if err = cb(repo.(*github.Repository)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Syncer) fetchRepoComments(context context.Context, repo *storage.Repo, cb func([]*github.RepositoryComment) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		comments, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Repositories.ListComments(context, repo.OrgLogin, repo.RepoName, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list comments for repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
		}

		if err := cb(comments.([]*github.RepositoryComment)); err != nil {
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
		members, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Organizations.ListMembers(context, org.OrgLogin, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list members of org %s: %v", org.OrgLogin, err)
		}

		if err := cb(members.([]*github.User)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func (s *Syncer) fetchLabels(context context.Context, repo *storage.Repo, cb func([]*github.Label) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		labels, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListLabels(context, repo.OrgLogin, repo.RepoName, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list all labels in repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
		}

		if err := cb(labels.([]*github.Label)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}

func (s *Syncer) fetchIssues(context context.Context, repo *storage.Repo, startTime time.Time, cb func([]*github.Issue) error) error {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		issues, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListByRepo(context, repo.OrgLogin, repo.RepoName, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list all issues in repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
		}

		if err := cb(issues.([]*github.Issue)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func (s *Syncer) fetchIssueComments(context context.Context, repo *storage.Repo, startTime time.Time,
	cb func([]*github.IssueComment) error) error {
	opt := &github.IssueListCommentsOptions{
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListComments(context, repo.OrgLogin, repo.RepoName, 0, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list comments for repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
		}

		if err := cb(comments.([]*github.IssueComment)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func (s *Syncer) fetchPullRequestReviewComments(context context.Context, repo *storage.Repo, startTime time.Time,
	cb func([]*github.PullRequestComment) error) error {
	opt := &github.PullRequestListCommentsOptions{
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.ListComments(context, repo.OrgLogin, repo.RepoName, 0, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list review comments for repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
		}

		if err := cb(comments.([]*github.PullRequestComment)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.ListOptions.Page = resp.NextPage
	}
}

func (s *Syncer) fetchFiles(context context.Context, repo *storage.Repo, prNumber int, cb func([]string) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		files, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.ListFiles(context, repo.OrgLogin, repo.RepoName, prNumber, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list files for pull request %d in repo %s/%s: %v", prNumber, repo.OrgLogin, repo.RepoName, err)
		}

		var result []string
		for _, f := range files.([]*github.CommitFile) {
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

func (s *Syncer) fetchPullRequests(context context.Context, repo *storage.Repo, cb func([]*github.PullRequest) error) error {
	opt := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.List(context, repo.OrgLogin, repo.RepoName, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list pull requests in repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
		}

		if err := cb(prs.([]*github.PullRequest)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			break
		}

		opt.ListOptions.Page = resp.NextPage
	}

	return nil
}

func (s *Syncer) fetchReviews(context context.Context, repo *storage.Repo, prNumber int, cb func([]*github.PullRequestReview) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		reviews, resp, err := s.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.ListReviews(context, repo.OrgLogin, repo.RepoName, prNumber, opt)
		})

		if err != nil {
			return fmt.Errorf("unable to list comments for pr %d in repo %s/%s: %v", prNumber, repo.OrgLogin, repo.RepoName, err)
		}

		if err = cb(reviews.([]*github.PullRequestReview)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}
