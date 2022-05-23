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

package gh

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v26/github"
)

func (tc *ThrottledClient) FetchRepoComments(context context.Context, orgLogin string, repoName string, cb func([]*github.RepositoryComment) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		comments, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Repositories.ListComments(context, orgLogin, repoName, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list comments for repo %s/%s: %v", orgLogin, repoName, err)
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

func (tc *ThrottledClient) FetchRepoEvents(context context.Context, orgLogin string, repoName string, cb func([]*github.Event) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		events, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Activity.ListRepositoryEvents(context, orgLogin, repoName, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list events for repo %s/%s: %v", orgLogin, repoName, err)
		}

		if err := cb(events.([]*github.Event)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}

func (tc *ThrottledClient) FetchIssueEvents(context context.Context, orgLogin string, repoName string, cb func([]*github.IssueEvent) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		events, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Activity.ListIssueEventsForRepository(context, orgLogin, repoName, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list issue events for repo %s/%s: %v", orgLogin, repoName, err)
		}

		if err := cb(events.([]*github.IssueEvent)); err != nil {
			return err
		}

		if resp.NextPage == 0 {
			return nil
		}

		opt.Page = resp.NextPage
	}
}

func (tc *ThrottledClient) FetchMembers(context context.Context, orgLogin string, cb func([]*github.User) error) error {
	opt := &github.ListMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		members, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Organizations.ListMembers(context, orgLogin, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list members of org %s: %v", orgLogin, err)
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

func (tc *ThrottledClient) FetchLabels(context context.Context, orgLogin string, repoName string, cb func([]*github.Label) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		labels, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListLabels(context, orgLogin, repoName, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list all labels in repo %s/%s: %v", orgLogin, repoName, err)
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

func (tc *ThrottledClient) FetchIssues(context context.Context, orgLogin string, repoName string, startTime time.Time, cb func([]*github.Issue) error) error {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		issues, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListByRepo(context, orgLogin, repoName, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list all issues in repo %s/%s: %v", orgLogin, repoName, err)
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

func (tc *ThrottledClient) FetchIssueComments(context context.Context, orgLogin string, repoName string, startTime time.Time,
	cb func([]*github.IssueComment) error,
) error {
	opt := &github.IssueListCommentsOptions{
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListComments(context, orgLogin, repoName, 0, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list comments for repo %s/%s: %v", orgLogin, repoName, err)
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

func (tc *ThrottledClient) FetchPullRequestReviewComments(context context.Context, orgLogin string, repoName string, startTime time.Time,
	cb func([]*github.PullRequestComment) error,
) error {
	opt := &github.PullRequestListCommentsOptions{
		Since: startTime,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.ListComments(context, orgLogin, repoName, 0, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list review comments for repo %s/%s: %v", orgLogin, repoName, err)
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

func (tc *ThrottledClient) FetchFiles(context context.Context, orgLogin string, repoName string, prNumber int, cb func([]string) error) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		files, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.ListFiles(context, orgLogin, repoName, prNumber, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list files for pull request %d in repo %s/%s: %v", prNumber, orgLogin, repoName, err)
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

func (tc *ThrottledClient) FetchPullRequests(context context.Context, orgLogin string, repoName string, cb func([]*github.PullRequest) error) error {
	opt := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		prs, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.List(context, orgLogin, repoName, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list pull requests in repo %s/%s: %v", orgLogin, repoName, err)
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

func (tc *ThrottledClient) FetchReviews(context context.Context, orgLogin string, repoName string, prNumber int,
	cb func([]*github.PullRequestReview) error,
) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		reviews, resp, err := tc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.PullRequests.ListReviews(context, orgLogin, repoName, prNumber, opt)
		})
		if err != nil {
			return fmt.Errorf("unable to list comments for pr %d in repo %s/%s: %v", prNumber, orgLogin, repoName, err)
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
