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
package notification

import (
	"context"

	"istio.io/bots/policybot/pkg/cmdutil"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
)

//send message to comment under Github Issue 7958 of istio.io
func SendGithubIssueComment(secrets *cmdutil.Secrets, message string) error {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: secrets.GitHubToken},
	)

	gc := github.NewClient(oauth2.NewClient(context.Background(), src))

	issueComment := &github.IssueComment{Body: &message}
	owner := "istio"
	repo := "istio.io"
	num := 7958
	_, _, err := gc.Issues.CreateComment(context.Background(), owner, repo, num, issueComment)
	return err
}
