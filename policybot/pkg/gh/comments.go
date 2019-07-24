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
	"strings"

	"github.com/google/go-github/v26/github"
)

// AddOrReplaceBotComment injects a comment from the bot into an issue or PR. It first removes any other
// comment it finds with the same signature
func AddOrReplaceBotComment(context context.Context, gc *ThrottledClient, orgLogin string, repoName string, number int, message string,
	signature string) error {
	msg := message + signature
	pc := &github.IssueComment{
		Body: &msg,
	}

	existing, id, err := FindBotComment(context, gc, orgLogin, repoName, number, signature)
	if err != nil {
		return err
	}

	if existing == msg {
		// bot comment is already present
		return nil
	} else if existing != "" {
		// try to delete the previous version
		if _, err := gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
			return client.Issues.DeleteComment(context, orgLogin, repoName, id)
		}); err != nil {
			return fmt.Errorf("unable to delete comment in issue/PR %d from repo %s/%s: %v", number, orgLogin, repoName, err)
		}
	}

	_, _, err = gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.CreateComment(context, orgLogin, repoName, number, pc)
	})

	if err != nil {
		return fmt.Errorf("unable to attach bot comment to issue/PR %d from repo %s/%s: %v", number, orgLogin, repoName, err)
	}

	return nil
}

// RemoveBotComment removes a comment from the bot in an issue or PR
func RemoveBotComment(context context.Context, gc *ThrottledClient, orgLogin string, repoName string, number int, signature string) error {
	existing, id, err := FindBotComment(context, gc, orgLogin, repoName, number, signature)
	if err != nil {
		return err
	}

	if existing != "" {
		if _, err = gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
			return client.Issues.DeleteComment(context, orgLogin, repoName, id)
		}); err != nil {
			return fmt.Errorf("unable to delete bot comment in issue/PR %d from repo %s/%s: %v", number, orgLogin, repoName, err)
		}
	}

	return nil
}

// FindBotComment looks for a bot comment in an issue or PR
func FindBotComment(context context.Context, gc *ThrottledClient, orgLogin string, repoName string, number int, signature string) (string, int64, error) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.ListComments(context, orgLogin, repoName, number, opt)
		})

		if err != nil {
			return "", -1, fmt.Errorf("unable to list comments for issue/PR %d in repo %s/%s: %v", number, orgLogin, repoName, err)
		}

		for _, comment := range comments.([]*github.IssueComment) {
			body := comment.GetBody()
			if strings.Contains(body, signature) {
				return body, comment.GetID(), nil
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return "", -1, nil
}
