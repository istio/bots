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

package refresher

import (
	"context"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/handlers/githubwebhook"
	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/coverage"
	"istio.io/bots/policybot/pkg/gh"
	gatherer "istio.io/bots/policybot/pkg/resultgatherer"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// Updates the DB based on incoming GitHub webhook events.
type Refresher struct {
	cache *cache.Cache
	store storage.Store
	gc    *gh.ThrottledClient
	bs    blobstorage.Store
	reg   *config.Registry
}

var scope = log.RegisterScope("refresher", "Dynamic database refresher", 0)

func NewRefresher(cache *cache.Cache, store storage.Store, bs blobstorage.Store, gc *gh.ThrottledClient, reg *config.Registry) githubwebhook.Filter {
	return &Refresher{
		cache: cache,
		store: store,
		bs:    bs,
		gc:    gc,
		reg:   reg,
	}
}

// accept an event arriving from GitHub
func (r *Refresher) Handle(context context.Context, event interface{}) {
	switch p := event.(type) {
	case *github.IssuesEvent:
		scope.Infof("Received IssuesEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		if _, ok := r.reg.SingleRecord(RecordType, p.GetRepo().GetFullName()); !ok {
			scope.Infof("Ignoring issue %d from repo %s since there aren't matching refreshers", p.GetIssue().GetNumber(), p.GetRepo().GetFullName())
			return
		}

		issue := gh.ConvertIssue(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetIssue())
		issues := []*storage.Issue{issue}
		if err := r.cache.WriteIssues(context, issues); err != nil {
			scope.Errorf(err.Error())
			return
		}

		event := &storage.IssueEvent{
			OrgLogin:    issue.OrgLogin,
			RepoName:    issue.RepoName,
			IssueNumber: issue.IssueNumber,
			CreatedAt:   p.GetIssue().GetUpdatedAt(),
			Actor:       p.GetSender().GetLogin(),
			Action:      p.GetAction(),
		}

		events := []*storage.IssueEvent{event}
		if err := r.store.WriteIssueEvents(context, events); err != nil {
			scope.Error(err.Error())
			return
		}

		r.syncUsers(context, issue.Author)
		r.syncUsers(context, issue.Assignees...)

	case *github.IssueCommentEvent:
		scope.Infof("Received IssueCommentEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetIssue().GetNumber(), p.GetAction())

		if _, ok := r.reg.SingleRecord(RecordType, p.GetRepo().GetFullName()); !ok {
			scope.Infof("Ignoring issue comment for issue %d from repo %s since there are no matching refreshers", p.GetIssue().GetNumber(), p.GetRepo().GetFullName())
			return
		}

		issueComment := gh.ConvertIssueComment(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetIssue().GetNumber(),
			p.GetComment())
		issueComments := []*storage.IssueComment{issueComment}
		if err := r.cache.WriteIssueComments(context, issueComments); err != nil {
			scope.Error(err.Error())
			return
		}

		event := &storage.IssueCommentEvent{
			OrgLogin:       issueComment.OrgLogin,
			RepoName:       issueComment.RepoName,
			IssueNumber:    issueComment.IssueNumber,
			IssueCommentID: p.GetComment().GetID(),
			CreatedAt:      p.GetComment().GetUpdatedAt(),
			Actor:          p.GetSender().GetLogin(),
			Action:         p.GetAction(),
		}

		events := []*storage.IssueCommentEvent{event}
		if err := r.store.WriteIssueCommentEvents(context, events); err != nil {
			scope.Error(err.Error())
			return
		}

		r.syncUsers(context, issueComment.Author)

	case *github.PullRequestEvent:
		scope.Infof("Received PullRequestEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetNumber(), p.GetAction())

		action := p.GetAction()

		rec, ok := r.reg.SingleRecord(RecordType, p.GetRepo().GetFullName())
		if ok {
			ref := rec.(*TestOutputRecord)
			orgLogin := p.GetRepo().GetOwner().GetLogin()
			repoName := p.GetRepo().GetName()
			prNum := p.GetNumber()

			if action == "opened" || p.GetAction() == "synchronize" {
				_, err := coverage.GetConfig(orgLogin, repoName)
				if err != nil {
					scope.Errorf("Unable to fetch coverage config for repo %s/%s: %v", orgLogin, repoName, err)
				} else {
					cov := coverage.Client{
						OrgLogin:      orgLogin,
						Repo:          repoName,
						BlobClient:    r.bs,
						StorageClient: r.store,
						GithubClient:  r.gc,
					}
					cov.SetCoverageStatus(context, p.GetPullRequest().GetHead().GetSHA(), coverage.Pending,
						"Waiting for test results.")
				}
			}

			tg := gatherer.TestResultGatherer{
				Client:           r.bs,
				BucketName:       ref.BucketName,
				PreSubmitPrefix:  ref.PreSubmitTestPath,
				PostSubmitPrefix: ref.PostSubmitTestPath,
			}

			testResults, err := tg.CheckTestResultsForPr(context, orgLogin, repoName, int64(prNum))
			if err != nil {
				scope.Errorf("Unable to get test result for PR %d in repo %s: %v", prNum, p.GetRepo().GetFullName(), err)
			} else if err = r.cache.WriteTestResults(context, testResults); err != nil {
				scope.Errorf("Unable to write test results: %v", err)
			}
		}

		if action == "opened" || action == "edited" {
			opt := &github.ListOptions{
				PerPage: 100,
			}

			// get the set of files comprising this PR since the payload didn't supply them
			var allFiles []string
			for {
				files, resp, err := r.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
					return client.PullRequests.ListFiles(context, p.GetRepo().GetOwner().GetLogin(), p.GetRepo().GetName(), p.GetNumber(), opt)
				})

				if err != nil {
					scope.Errorf("Unable to list all files for pull request %d in repo %s: %v\n", p.GetNumber(), p.GetRepo().GetFullName(), err)
					return
				}

				for _, f := range files.([]*github.CommitFile) {
					allFiles = append(allFiles, f.GetFilename())
				}

				if resp.NextPage == 0 {
					break
				}

				opt.Page = resp.NextPage
			}

			pr := gh.ConvertPullRequest(
				p.GetOrganization().GetLogin(),
				p.GetRepo().GetName(),
				p.GetPullRequest(),
				allFiles)
			prs := []*storage.PullRequest{pr}
			if err := r.cache.WritePullRequests(context, prs); err != nil {
				scope.Errorf(err.Error())
				return
			}

			r.syncUsers(context, pr.Author)
			r.syncUsers(context, pr.Assignees...)
			r.syncUsers(context, pr.RequestedReviewers...)
		}

		event := &storage.PullRequestEvent{
			OrgLogin:          p.GetOrganization().GetLogin(),
			RepoName:          p.GetRepo().GetName(),
			PullRequestNumber: int64(p.GetPullRequest().GetNumber()),
			CreatedAt:         p.GetPullRequest().GetUpdatedAt(),
			Actor:             p.GetSender().GetLogin(),
			Action:            p.GetAction(),
			Merged:            p.GetPullRequest().GetMerged(),
		}

		events := []*storage.PullRequestEvent{event}
		if err := r.store.WritePullRequestEvents(context, events); err != nil {
			scope.Error(err.Error())
			return
		}

		r.syncUsers(context, event.Actor)

	case *github.PullRequestReviewEvent:
		scope.Infof("Received PullRequestReviewEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		if _, ok := r.reg.SingleRecord(RecordType, p.GetRepo().GetFullName()); !ok {
			scope.Infof("Ignoring PR review for PR %d from repo %s since there are no matching refreshers", p.GetPullRequest().GetNumber(), p.GetRepo().GetFullName())
			return
		}

		review := gh.ConvertPullRequestReview(
			p.GetOrganization().GetLogin(),
			p.GetRepo().GetName(),
			p.GetPullRequest().GetNumber(),
			p.GetReview())
		reviews := []*storage.PullRequestReview{review}
		if err := r.cache.WritePullRequestReviews(context, reviews); err != nil {
			scope.Errorf(err.Error())
			return
		}

		event := &storage.PullRequestReviewEvent{
			OrgLogin:            review.OrgLogin,
			RepoName:            review.RepoName,
			PullRequestNumber:   review.PullRequestNumber,
			PullRequestReviewID: p.GetReview().GetID(),
			CreatedAt:           p.GetReview().GetSubmittedAt(),
			Actor:               p.GetSender().GetLogin(),
			Action:              p.GetAction(),
		}

		events := []*storage.PullRequestReviewEvent{event}
		if err := r.store.WritePullRequestReviewEvents(context, events); err != nil {
			scope.Error(err.Error())
			return
		}

		r.syncUsers(context, review.Author)

	case *github.PullRequestReviewCommentEvent:
		scope.Infof("Received PullRequestReviewCommentEvent: %s, %d, %s", p.GetRepo().GetFullName(), p.GetPullRequest().GetNumber(), p.GetAction())

		if _, ok := r.reg.SingleRecord(RecordType, p.GetRepo().GetFullName()); !ok {
			scope.Infof("Ignoring PR review comment for PR %d from repo %s since there are no matching refreshers",
				p.GetPullRequest().GetNumber(), p.GetRepo().GetFullName())
			return
		}

		comment := gh.ConvertPullRequestReviewComment(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetPullRequest().GetNumber(),
			p.GetComment())
		comments := []*storage.PullRequestReviewComment{comment}
		if err := r.cache.WritePullRequestReviewComments(context, comments); err != nil {
			scope.Errorf(err.Error())
		}

		event := &storage.PullRequestReviewCommentEvent{
			OrgLogin:                   comment.OrgLogin,
			RepoName:                   comment.RepoName,
			PullRequestNumber:          comment.PullRequestNumber,
			PullRequestReviewCommentID: p.GetComment().GetID(),
			CreatedAt:                  p.GetComment().GetUpdatedAt(),
			Actor:                      p.GetSender().GetLogin(),
			Action:                     p.GetAction(),
		}

		events := []*storage.PullRequestReviewCommentEvent{event}
		if err := r.store.WritePullRequestReviewCommentEvents(context, events); err != nil {
			scope.Error(err.Error())
			return
		}

		r.syncUsers(context, comment.Author)

	case *github.CommitCommentEvent:
		scope.Infof("Received CommitCommentEvent: %s, %s", p.GetRepo().GetFullName(), p.GetAction())

		if _, ok := r.reg.SingleRecord(RecordType, p.GetRepo().GetFullName()); !ok {
			scope.Infof("Ignoring repo comment from repo %s since there are no matching refreshers", p.GetRepo().GetFullName())
			return
		}

		comment := gh.ConvertRepoComment(
			p.GetRepo().GetOwner().GetLogin(),
			p.GetRepo().GetName(),
			p.GetComment())
		comments := []*storage.RepoComment{comment}
		if err := r.cache.WriteRepoComments(context, comments); err != nil {
			scope.Errorf(err.Error())
		}

		event := &storage.RepoCommentEvent{
			OrgLogin:      comment.OrgLogin,
			RepoName:      comment.RepoName,
			RepoCommentID: p.GetComment().GetID(),
			CreatedAt:     p.GetComment().GetUpdatedAt(),
			Actor:         p.GetSender().GetLogin(),
			Action:        p.GetAction(),
		}

		events := []*storage.RepoCommentEvent{event}
		if err := r.store.WriteRepoCommentEvents(context, events); err != nil {
			scope.Error(err.Error())
			return
		}

		r.syncUsers(context, comment.Author)

	case *github.StatusEvent:
		scope.Infof("Received StatusEvent: %s", p.GetRepo().GetFullName())

		if p.GetState() == "pending" {
			scope.Infof("Ignoring StatusEvent from repo %s because it's pending", p.GetRepo().GetFullName())
			return
		}

		rec, ok := r.reg.SingleRecord(RecordType, p.GetRepo().GetFullName())
		if !ok {
			scope.Infof("Ignoring status event from repo %s since there are no matching refreshers", p.GetRepo().GetFullName())
			return
		}

		ref := rec.(*TestOutputRecord)
		orgLogin := p.GetRepo().GetOwner().GetLogin()
		repoName := p.GetRepo().GetName()

		sha := p.GetCommit().GetSHA()
		pr, err := r.gc.GetPRForSHA(context, orgLogin, repoName, sha)
		if err != nil {
			scope.Errorf("Unable to fetch pull request info for commit %s in repo %s: %v", sha, p.GetRepo().GetFullName(), err)
			return
		}

		prNum := int64(pr.GetNumber())
		scope.Debugf("Commit %s corresponds to pull request %d", sha, prNum)

		tg := gatherer.TestResultGatherer{
			Client:           r.bs,
			BucketName:       ref.BucketName,
			PreSubmitPrefix:  ref.PreSubmitTestPath,
			PostSubmitPrefix: ref.PostSubmitTestPath,
		}

		testResults, err := tg.CheckTestResultsForPr(context, orgLogin, repoName, prNum)
		if err != nil {
			scope.Errorf("Unable to get test result for PR %d in repo %s: %v", prNum, p.GetRepo().GetFullName(), err)
			return
		}

		if err = r.cache.WriteTestResults(context, testResults); err != nil {
			scope.Errorf("Unable to write test results: %v", err)
			return
		}

		cov := coverage.Client{
			OrgLogin:      orgLogin,
			Repo:          repoName,
			BlobClient:    r.bs,
			StorageClient: r.store,
			GithubClient:  r.gc,
		}

		if err = cov.CheckCoverage(context, pr, sha); err != nil {
			scope.Errorf("unable to check coverage for PR %d in repo %s: %v", prNum, err, p.GetRepo().GetFullName())
			return
		}

	default:
		// not what we're looking for
		scope.Debugf("Unknown event received: %T %+v", p, p)
		return
	}
}

func (r *Refresher) syncUsers(context context.Context, discoveredUsers ...string) {
	users := make([]*storage.User, 0, len(discoveredUsers))
	for _, discoveredUser := range discoveredUsers {
		if u, err := r.cache.ReadUser(context, discoveredUser); err != nil {
			scope.Errorf("Unable to read info for user %s from storage: %v", discoveredUser, err)
			return
		} else if u == nil {
			if ghUser, _, err := r.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
				return client.Users.Get(context, discoveredUser)
			}); err == nil {
				users = append(users, gh.ConvertUser(ghUser.(*github.User)))
			} else {
				scope.Warnf("couldn't get information for user %s: %v", discoveredUser, err)

				// go with what we know
				users = append(users, &storage.User{UserLogin: discoveredUser})
			}
		}
	}

	if err := r.cache.WriteUsers(context, users); err != nil {
		scope.Errorf("Unable to write users: %v", err)
	}
}
