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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"istio.io/bots/policybot/pkg/pipeline"
	"istio.io/pkg/env"

	"github.com/hashicorp/go-multierror"

	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/blobstorage/gcs"

	"cloud.google.com/go/bigquery"
	"github.com/ghodss/yaml"
	"github.com/google/go-github/v26/github"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/resultgatherer"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/zh"
	"istio.io/pkg/log"
)

// Syncer is responsible for synchronizing state from GitHub and ZenHub into our local store
type Syncer struct {
	bq        *bigquery.Client
	gc        *gh.ThrottledClient
	zc        *zh.ThrottledClient
	store     storage.Store
	orgs      []config.Org
	blobstore blobstorage.Store
}

type FilterFlags int

// the things to sync
const (
	Issues       FilterFlags = 1 << 0
	Prs                      = 1 << 1
	Maintainers              = 1 << 2
	Members                  = 1 << 3
	Labels                   = 1 << 4
	ZenHub                   = 1 << 5
	RepoComments             = 1 << 6
	Events                   = 1 << 7
	TestResults              = 1 << 8
)

// The state in Syncer is immutable once created. syncState on the other hand represents
// the mutable state used during a single sync operation.
type syncState struct {
	syncer *Syncer
	users  map[string]bool
	flags  FilterFlags
	ctx    context.Context
}

var scope = log.RegisterScope("syncer", "The GitHub/ZenHub data syncer", 0)

func New(gc *gh.ThrottledClient, gcpCreds []byte, gcpProject string,
	zc *zh.ThrottledClient, store storage.Store, orgs []config.Org) (*Syncer, error) {
	bq, err := bigquery.NewClient(context.Background(), gcpProject, option.WithCredentialsJSON(gcpCreds))
	if err != nil {
		return nil, fmt.Errorf("unable to create BigQuery client: %v", err)
	}
	bs, err := gcs.NewStore(context.Background(), gcpCreds)
	if err != nil {
		return nil, fmt.Errorf("unable to create gcs client: %v", err)
	}

	return &Syncer{
		gc:        gc,
		bq:        bq,
		zc:        zc,
		store:     store,
		blobstore: bs,
		orgs:      orgs,
	}, nil
}

func ConvFilterFlags(filter string) (FilterFlags, error) {
	if filter == "" {
		// defaults to everything
		return Issues | Prs | Maintainers | Members | Labels | ZenHub | RepoComments | Events | TestResults, nil
	}

	var result FilterFlags
	for _, f := range strings.Split(filter, ",") {
		switch strings.ToLower(f) {
		case "issues":
			result |= Issues
		case "prs":
			result |= Prs
		case "maintainers":
			result |= Maintainers
		case "members":
			result |= Members
		case "labels":
			result |= Labels
		case "zenhub":
			result |= ZenHub
		case "repocomments":
			result |= RepoComments
		case "events":
			result |= Events
		case "testresults":
			result |= TestResults
		default:
			return 0, fmt.Errorf("unknown filter flag %s", f)
		}
	}

	return result, nil
}

func (s *Syncer) Sync(context context.Context, flags FilterFlags) error {
	ss := &syncState{
		syncer: s,
		users:  make(map[string]bool),
		flags:  flags,
		ctx:    context,
	}

	var orgs []*storage.Org
	var repos []*storage.Repo

	// get all the org & repo info
	if err := s.fetchOrgs(ss.ctx, func(org *github.Organization) error {
		storageOrg := gh.ConvertOrg(org)
		orgs = append(orgs, storageOrg)
		return s.fetchRepos(ss.ctx, func(repo *github.Repository) error {
			storageRepo := gh.ConvertRepo(repo)
			repos = append(repos, storageRepo)
			return nil
		})
	}); err != nil {
		return err
	}

	if err := s.store.WriteOrgs(ss.ctx, orgs); err != nil {
		return err
	}

	if err := s.store.WriteRepos(ss.ctx, repos); err != nil {
		return err
	}
	// persist data about storage.Orgs and related
	for _, org := range orgs {
		var orgRepos []*storage.Repo
		for _, repo := range repos {
			if repo.OrgLogin == org.OrgLogin {
				orgRepos = append(orgRepos, repo)
			}
		}

		if flags&(Members|Labels|Issues|Prs|ZenHub|RepoComments|Events) != 0 {
			if err := ss.handleOrg(org, orgRepos); err != nil {
				return err
			}
		}
	}

	// process data to persist about config.orgs and related
	for _, org := range s.orgs {

		if flags&Maintainers != 0 {
			if err := ss.handleMaintainers(&org); err != nil {
				return err
			}
		}

		if flags&TestResults != 0 {
			if err := ss.handleTestResults(&org); err != nil {
				return err
			}
		}
	}

	if err := ss.pushUsers(); err != nil {
		return err
	}

	return nil
}

func (ss *syncState) pushUsers() error {
	users := make([]*storage.User, 0, len(ss.users))
	for login := range ss.users {
		if u, err := ss.syncer.store.ReadUser(ss.ctx, login); err != nil {
			return fmt.Errorf("unable to read info for user %s from storage: %v", login, err)
		} else if u == nil || u.Name == "" {
			if ghUser, _, err := ss.syncer.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
				return client.Users.Get(ss.ctx, login)
			}); err == nil {
				users = append(users, gh.ConvertUser(ghUser.(*github.User)))
			} else {
				scope.Warnf("couldn't get information for user %s: %v", login, err)

				// go with what we know
				users = append(users, &storage.User{UserLogin: login})
			}
		}
	}

	if err := ss.syncer.store.WriteUsers(ss.ctx, users); err != nil {
		return err
	}

	return nil
}

func (ss *syncState) handleOrg(org *storage.Org, repos []*storage.Repo) error {
	scope.Infof("Syncing org %s", org.OrgLogin)

	for _, repo := range repos {
		if err := ss.handleRepo(repo); err != nil {
			return err
		}
	}

	if ss.flags&Members != 0 {
		if err := ss.handleMembers(org, repos); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) handleRepo(repo *storage.Repo) error {
	scope.Infof("Syncing repo %s/%s", repo.OrgLogin, repo.RepoName)

	if ss.flags&Labels != 0 {
		if err := ss.handleLabels(repo); err != nil {
			return err
		}
	}

	if ss.flags&Issues != 0 {
		if err := ss.handleActivity(repo, ss.handleIssues, func(activity *storage.BotActivity) *time.Time {
			return &activity.LastIssueSyncStart
		}); err != nil {
			return err
		}

		if err := ss.handleActivity(repo, ss.handleIssueComments, func(activity *storage.BotActivity) *time.Time {
			return &activity.LastIssueCommentSyncStart
		}); err != nil {
			return err
		}

	}

	if ss.flags&ZenHub != 0 {
		if err := ss.handleZenHub(repo); err != nil {
			return err
		}
	}

	if ss.flags&Prs != 0 {
		if err := ss.handlePullRequests(repo); err != nil {
			return err
		}

		if err := ss.handleActivity(repo, ss.handlePullRequestReviewComments, func(activity *storage.BotActivity) *time.Time {
			return &activity.LastPullRequestReviewCommentSyncStart
		}); err != nil {
			return err
		}
	}

	if ss.flags&RepoComments != 0 {
		if err := ss.handleRepoComments(repo); err != nil {
			return err
		}
	}

	if ss.flags&Events != 0 {
		if err := ss.handleEvents(repo); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) handleActivity(repo *storage.Repo, cb func(*storage.Repo, time.Time) error,
	getField func(*storage.BotActivity) *time.Time) error {

	start := time.Now().UTC()
	priorStart := time.Time{}

	if activity, _ := ss.syncer.store.ReadBotActivity(ss.ctx, repo.OrgLogin, repo.RepoName); activity != nil {
		priorStart = *getField(activity)
	}

	if err := cb(repo, priorStart); err != nil {
		return err
	}

	if err := ss.syncer.store.UpdateBotActivity(ss.ctx, repo.OrgLogin, repo.RepoName, func(act *storage.BotActivity) error {
		if *getField(act) == priorStart {
			*getField(act) = start
		}
		return nil
	}); err != nil {
		scope.Warnf("unable to update bot activity for repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
	}

	return nil
}

func (ss *syncState) handleMembers(org *storage.Org, repos []*storage.Repo) error {
	scope.Debugf("Getting members from org %s", org.OrgLogin)

	var storageMembers []*storage.Member
	if err := ss.syncer.fetchMembers(ss.ctx, org, func(members []*github.User) error {
		for _, member := range members {
			ss.addUsers(member.GetLogin())
			storageMembers = append(storageMembers, &storage.Member{OrgLogin: org.OrgLogin, UserLogin: member.GetLogin()})
		}

		return nil
	}); err != nil {
		return err
	}

	// now compute the contribution info for each maintainer

	var repoNames []string
	for _, repo := range repos {
		repoNames = append(repoNames, repo.RepoName)
	}

	for _, member := range storageMembers {
		info, err := ss.syncer.store.QueryMemberActivity(ss.ctx, member, repoNames)
		if err != nil {
			scope.Warnf("Couldn't get contribution info for member %s: %v", member.UserLogin, err)
			member.CachedInfo = ""
			continue
		}

		result, err := json.Marshal(info)
		if err != nil {
			scope.Warnf("Couldn't encode contribution info for member %s: %v", member.UserLogin, err)
			member.CachedInfo = ""
			continue
		}

		scope.Debugf("Saving cached contribution info for member %s", member.UserLogin)
		member.CachedInfo = string(result)
	}

	return ss.syncer.store.WriteAllMembers(ss.ctx, storageMembers)
}

func (ss *syncState) handleLabels(repo *storage.Repo) error {
	scope.Debugf("Getting labels from repo %s/%s", repo.OrgLogin, repo.RepoName)

	return ss.syncer.fetchLabels(ss.ctx, repo, func(labels []*github.Label) error {
		storageLabels := make([]*storage.Label, 0, len(labels))
		for _, label := range labels {
			storageLabels = append(storageLabels, gh.ConvertLabel(repo.OrgLogin, repo.RepoName, label))
		}

		return ss.syncer.store.WriteLabels(ss.ctx, storageLabels)
	})
}

func (ss *syncState) handleEvents(repo *storage.Repo) error {
	scope.Debugf("Getting events from repo %s/%s", repo.OrgLogin, repo.RepoName)

	type eventRecord struct {
		Type      string
		OrgLogin  string
		RepoName  string
		Actor     string
		CreatedAt time.Time
		Payload   string
	}

	now := time.Now()

	// TODO: we should look at what the latest event is in the DB and only update from there.
	// TODO: there's a limit of 1000 days that can be queried at once, so this logic should take
	//       that into account and issue multiple queries if needed

	q := ss.syncer.bq.Query(fmt.Sprintf(`
		SELECT * FROM (
		  SELECT type as Type, payload as Payload, org.login as OrgLogin, repo.name as RepoName, actor.login as Actor, created_at as CreatedAt,
			JSON_EXTRACT(payload, '$.action') as event
		  FROM (TABLE_DATE_RANGE([githubarchive:day.],
			TIMESTAMP('2019-07-01'),
			TIMESTAMP('%4d-%2d-%2d')
		  ))
		  WHERE (type = 'IssuesEvent'
					OR type = 'IssueCommentEvent'
					OR type = 'PullRequestEvent'
					OR type = 'PullRequestReviewEvent'
					OR type = 'PullRequestReviewCommentEvent')
				AND repo.name = '%s/%s'
		);`, now.Year(), now.Month(), now.Day(), repo.OrgLogin, repo.RepoName))

	q.UseLegacySQL = true

	it, err := q.Read(ss.ctx)
	if err != nil {
		return fmt.Errorf("unable to read events from GitHub archive: %v", err)
	}

	var issueEvents []*storage.IssueEvent
	var issueCommentEvents []*storage.IssueCommentEvent
	var prEvents []*storage.PullRequestEvent
	var prCommentEvents []*storage.PullRequestReviewCommentEvent
	var prReviewEvents []*storage.PullRequestReviewEvent

	count := 0
	done := false
	for {
		var r eventRecord
		err := it.Next(&r)
		if err == iterator.Done {
			done = true
			r.Type = ""
		} else if err != nil {
			return fmt.Errorf("unable to iterate BigQuery result: %v", err)
		}

		ss.addUsers(r.Actor)

		switch r.Type {
		case "IssuesEvent":
			var e github.IssuesEvent
			err = json.Unmarshal([]byte(r.Payload), &e)
			if err != nil {
				return fmt.Errorf("unable to unmarshal event payload: %v", err)
			}

			issueEvents = append(issueEvents, &storage.IssueEvent{
				OrgLogin:    r.OrgLogin,
				RepoName:    r.RepoName[strings.Index(r.RepoName, "/")+1:],
				CreatedAt:   r.CreatedAt,
				Actor:       r.Actor,
				Action:      e.GetAction(),
				IssueNumber: int64(e.GetIssue().GetNumber()),
			})

		case "IssueCommentEvent":
			var e github.IssueCommentEvent
			err = json.Unmarshal([]byte(r.Payload), &e)
			if err != nil {
				return fmt.Errorf("unable to unmarshal event payload: %v", err)
			}

			issueCommentEvents = append(issueCommentEvents, &storage.IssueCommentEvent{
				OrgLogin:       r.OrgLogin,
				RepoName:       r.RepoName[strings.Index(r.RepoName, "/")+1:],
				CreatedAt:      r.CreatedAt,
				Actor:          r.Actor,
				Action:         e.GetAction(),
				IssueNumber:    int64(e.GetIssue().GetNumber()),
				IssueCommentID: e.GetComment().GetID(),
			})

		case "PullRequestEvent":
			var e github.PullRequestEvent
			err = json.Unmarshal([]byte(r.Payload), &e)
			if err != nil {
				return fmt.Errorf("unable to unmarshal event payload: %v", err)
			}

			prEvents = append(prEvents, &storage.PullRequestEvent{
				OrgLogin:          r.OrgLogin,
				RepoName:          r.RepoName[strings.Index(r.RepoName, "/")+1:],
				CreatedAt:         r.CreatedAt,
				Actor:             r.Actor,
				Action:            e.GetAction(),
				PullRequestNumber: int64(e.GetPullRequest().GetNumber()),
				Merged:            e.GetPullRequest().GetMerged(),
			})

		case "PullRequestReviewCommentEvent":
			var e github.PullRequestReviewCommentEvent
			err = json.Unmarshal([]byte(r.Payload), &e)
			if err != nil {
				return fmt.Errorf("unable to unmarshal event payload: %v", err)
			}

			prCommentEvents = append(prCommentEvents, &storage.PullRequestReviewCommentEvent{
				OrgLogin:                   r.OrgLogin,
				RepoName:                   r.RepoName[strings.Index(r.RepoName, "/")+1:],
				CreatedAt:                  r.CreatedAt,
				Actor:                      r.Actor,
				Action:                     e.GetAction(),
				PullRequestNumber:          int64(e.GetPullRequest().GetNumber()),
				PullRequestReviewCommentID: e.GetComment().GetID(),
			})

		case "PullRequestReviewEvent":
			var e github.PullRequestReviewEvent
			err = json.Unmarshal([]byte(r.Payload), &e)
			if err != nil {
				return fmt.Errorf("unable to unmarshal event payload: %v", err)
			}

			prReviewEvents = append(prReviewEvents, &storage.PullRequestReviewEvent{
				OrgLogin:            r.OrgLogin,
				RepoName:            r.RepoName[strings.Index(r.RepoName, "/")+1:],
				CreatedAt:           r.CreatedAt,
				Actor:               r.Actor,
				Action:              e.GetAction(),
				PullRequestNumber:   int64(e.GetPullRequest().GetNumber()),
				PullRequestReviewID: e.GetReview().GetID(),
			})
		}

		count++
		if count%1000 == 0 || done {
			scope.Infof("Received %d events", count)

			if len(issueEvents) > 0 {
				if err := ss.syncer.store.WriteIssueEvents(ss.ctx, issueEvents); err != nil {
					return fmt.Errorf("unable to write issue events to storage: %v", err)
				}
				issueEvents = issueEvents[:0]
			}

			if len(issueCommentEvents) > 0 {
				if err := ss.syncer.store.WriteIssueCommentEvents(ss.ctx, issueCommentEvents); err != nil {
					return fmt.Errorf("unable to write issue comment events to storage: %v", err)
				}
				issueCommentEvents = issueCommentEvents[:0]
			}

			if len(prEvents) > 0 {
				if err := ss.syncer.store.WritePullRequestEvents(ss.ctx, prEvents); err != nil {
					return fmt.Errorf("unable to write pull request events to storage: %v", err)
				}
				prEvents = prEvents[:0]
			}

			if len(prCommentEvents) > 0 {
				if err := ss.syncer.store.WritePullRequestReviewCommentEvents(ss.ctx, prCommentEvents); err != nil {
					return fmt.Errorf("unable to write pull request review comment events to storage: %v", err)
				}
				prCommentEvents = prCommentEvents[:0]
			}

			if len(prReviewEvents) > 0 {
				if err := ss.syncer.store.WritePullRequestReviewEvents(ss.ctx, prReviewEvents); err != nil {
					return fmt.Errorf("unable to write pull request review events to storage: %v", err)
				}
				prReviewEvents = prReviewEvents[:0]
			}

			if done {
				return nil
			}
		}
	}
}

func (ss *syncState) handleRepoComments(repo *storage.Repo) error {
	scope.Debugf("Getting comments for repo %s/%s", repo.OrgLogin, repo.RepoName)

	return ss.syncer.fetchRepoComments(ss.ctx, repo, func(comments []*github.RepositoryComment) error {
		storageComments := make([]*storage.RepoComment, 0, len(comments))
		for _, comment := range comments {
			t := gh.ConvertRepoComment(repo.OrgLogin, repo.RepoName, comment)
			storageComments = append(storageComments, t)
			ss.addUsers(t.Author)
		}

		return ss.syncer.store.WriteRepoComments(ss.ctx, storageComments)
	})
}

func (ss *syncState) handleIssues(repo *storage.Repo, startTime time.Time) error {
	scope.Debugf("Getting issues from repo %s/%s", repo.OrgLogin, repo.RepoName)

	total := 0
	return ss.syncer.fetchIssues(ss.ctx, repo, startTime, func(issues []*github.Issue) error {
		var storageIssues []*storage.Issue

		total += len(issues)
		scope.Infof("Received %d issues", total)

		for _, issue := range issues {
			t := gh.ConvertIssue(repo.OrgLogin, repo.RepoName, issue)
			storageIssues = append(storageIssues, t)
			ss.addUsers(t.Author)
			ss.addUsers(t.Assignees...)
		}

		return ss.syncer.store.WriteIssues(ss.ctx, storageIssues)
	})
}

func (ss *syncState) handleIssueComments(repo *storage.Repo, startTime time.Time) error {
	scope.Debugf("Getting issue comments from repo %s/%s", repo.OrgLogin, repo.RepoName)

	total := 0
	return ss.syncer.fetchIssueComments(ss.ctx, repo, startTime, func(comments []*github.IssueComment) error {
		var storageIssueComments []*storage.IssueComment

		total += len(comments)
		scope.Infof("Received %d issue comments", total)

		for _, comment := range comments {
			issueURL := comment.GetIssueURL()
			issueNumber, _ := strconv.Atoi(issueURL[strings.LastIndex(issueURL, "/")+1:])
			t := gh.ConvertIssueComment(repo.OrgLogin, repo.RepoName, issueNumber, comment)
			storageIssueComments = append(storageIssueComments, t)
			ss.addUsers(t.Author)
		}

		return ss.syncer.store.WriteIssueComments(ss.ctx, storageIssueComments)
	})
}

func (ss *syncState) handleZenHub(repo *storage.Repo) error {
	scope.Debugf("Getting ZenHub issue data for repo %s/%s", repo.OrgLogin, repo.RepoName)

	// get all the issues
	var issues []*storage.Issue
	if err := ss.syncer.store.QueryIssuesByRepo(ss.ctx, repo.OrgLogin, repo.RepoName, func(issue *storage.Issue) error {
		issues = append(issues, issue)
		return nil
	}); err != nil {
		return fmt.Errorf("unable to read issues from repo %s/%s: %v", repo.OrgLogin, repo.RepoName, err)
	}

	// now get the ZenHub data for all issues
	var pipelines []*storage.IssuePipeline
	for _, issue := range issues {
		issueData, err := ss.syncer.zc.ThrottledCall(func(client *zh.Client) (interface{}, error) {
			return client.GetIssueData(int(repo.RepoNumber), int(issue.IssueNumber))
		})

		if err != nil {
			if err == zh.ErrNotFound {
				// not found, so nothing to do...
				return nil
			}

			return fmt.Errorf("unable to get issue data from ZenHub for issue %d in repo %s/%s: %v", issue.IssueNumber, repo.OrgLogin, repo.RepoName, err)
		}

		pipelines = append(pipelines, &storage.IssuePipeline{
			OrgLogin:    repo.OrgLogin,
			RepoName:    repo.RepoName,
			IssueNumber: issue.IssueNumber,
			Pipeline:    issueData.(*zh.IssueData).Pipeline.Name,
		})

		if len(pipelines)%100 == 0 {
			if err = ss.syncer.store.WriteIssuePipelines(ss.ctx, pipelines); err != nil {
				return err
			}
			pipelines = pipelines[:0]
		}
	}

	return ss.syncer.store.WriteIssuePipelines(ss.ctx, pipelines)
}

func (ss *syncState) handlePullRequests(repo *storage.Repo) error {
	scope.Debugf("Getting pull requests from repo %s/%s", repo.OrgLogin, repo.RepoName)

	total := 0
	return ss.syncer.fetchPullRequests(ss.ctx, repo, func(prs []*github.PullRequest) error {
		var storagePRs []*storage.PullRequest
		var storagePRReviews []*storage.PullRequestReview

		total += len(prs)
		scope.Infof("Received %d pull requests", total)

		for _, pr := range prs {
			// if this pr is already known to us and is up to date, skip further processing
			if existing, _ := ss.syncer.store.ReadPullRequest(ss.ctx, repo.OrgLogin, repo.RepoName, pr.GetNumber()); existing != nil {
				if existing.UpdatedAt == pr.GetUpdatedAt() {
					continue
				}
			}

			if err := ss.syncer.fetchReviews(ss.ctx, repo, pr.GetNumber(), func(reviews []*github.PullRequestReview) error {
				for _, review := range reviews {
					t := gh.ConvertPullRequestReview(repo.OrgLogin, repo.RepoName, pr.GetNumber(), review)
					storagePRReviews = append(storagePRReviews, t)
					ss.addUsers(t.Author)
				}

				return nil
			}); err != nil {
				return err
			}

			var prFiles []string
			if err := ss.syncer.fetchFiles(ss.ctx, repo, pr.GetNumber(), func(files []string) error {
				prFiles = append(prFiles, files...)
				return nil
			}); err != nil {
				return err
			}

			t := gh.ConvertPullRequest(repo.OrgLogin, repo.RepoName, pr, prFiles)
			storagePRs = append(storagePRs, t)
			ss.addUsers(t.Author)
			ss.addUsers(t.Assignees...)
			ss.addUsers(t.RequestedReviewers...)
		}

		err := ss.syncer.store.WritePullRequests(ss.ctx, storagePRs)
		if err == nil {
			err = ss.syncer.store.WritePullRequestReviews(ss.ctx, storagePRReviews)
		}

		return err
	})
}

func (ss *syncState) handlePullRequestReviewComments(repo *storage.Repo, start time.Time) error {
	scope.Debugf("Getting pull requests review comments from repo %s/%s", repo.OrgLogin, repo.RepoName)

	total := 0
	return ss.syncer.fetchPullRequestReviewComments(ss.ctx, repo, start, func(comments []*github.PullRequestComment) error {
		var storagePRComments []*storage.PullRequestReviewComment

		total += len(comments)
		scope.Infof("Received %d pull request review comments", total)

		for _, comment := range comments {
			prURL := comment.GetPullRequestURL()
			prNumber, _ := strconv.Atoi(prURL[strings.LastIndex(prURL, "/")+1:])
			t := gh.ConvertPullRequestReviewComment(repo.OrgLogin, repo.RepoName, prNumber, comment)
			storagePRComments = append(storagePRComments, t)
			ss.addUsers(t.Author)
		}

		return ss.syncer.store.WritePullRequestReviewComments(ss.ctx, storagePRComments)
	})
}

func (ss *syncState) handleTestResults(org *config.Org) error {
	scope.Debugf("Getting test results for org %s", org.Name)
	g := resultgatherer.TestResultGatherer{Client: ss.syncer.blobstore, BucketName: org.BucketName,
		PreSubmitPrefix: org.PreSubmitTestPath, PostSubmitPrefix: org.PostSubmitTestPath}
	prMin := env.RegisterIntVar("PR_MIN", 0, "The minimum PR to scan for test results").Get()
	prMax := env.RegisterIntVar("PR_MAX", -1, "The maximum PR to scan for test results").Get()
	for _, repo := range org.Repos {
		var completedTests map[string]bool
		ctLock := sync.RWMutex{}
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			ctLock.Lock()
			defer ctLock.Unlock()
			_ = ss.syncer.store.QueryTestResultByDone(ss.ctx, org.Name, repo.Name,
				func(result *storage.TestResult) error {
					completedTests[result.RunPath] = true
					return nil
				})
			wg.Done()
		}()
		prPaths := g.GetAllPullRequestsChan(ss.ctx, org.Name, repo.Name)
		// I think a composition syntax would be better here...
		errorChan := prPaths.Transform(func(prPathi interface{}) (prNum interface{}, err error) {
			prPath := prPathi.(string)
			prParts := strings.Split(prPath, "/")
			if len(prParts) < 2 {
				err = errors.New("too few segments in pr path")
				return
			}
			prNum = prParts[len(prParts)-2]
			if prInt, err := strconv.Atoi(prParts[len(prParts)-2]); err == nil {
				// skip this PR if it's outside the min and max inclusive
				if prInt < prMin || (prMax > -1 && prInt > prMax) {
					err = pipeline.Skip
				}
			}
			return
		}).WithContext(ss.ctx).OnError(func(e error) {
			// TODO: this should probably be reported out or something...
			scope.Warnf("error processing test: %s", e)
		}).WithParallelism(5).Transform(func(prNumi interface{}) (testRunPaths interface{}, err error) {
			prNum := prNumi.(string)
			tests, err := g.GetTestsForPR(ss.ctx, org.Name, repo.Name, prNum)
			var result [][]string

			// Wait for a comprehensive list of completed tests
			wg.Wait()
			ctLock.RLock()
			defer ctLock.RUnlock()

			for testName, runPaths := range tests {
				for _, runPath := range runPaths {
					if _, ok := completedTests[runPath]; !ok {
						result = append(result, []string{testName, runPath})
					}
				}
			}
			testRunPaths = result
			return
		}).Expand().Transform(func(testRunPathi interface{}) (i interface{}, err error) {
			inputArray := testRunPathi.([]string)
			testRunPath := inputArray[1]
			testName := inputArray[0]
			fmt.Printf("doing stuff: %v\n", inputArray)
			return g.GetTestResult(ss.ctx, testName, testRunPath)
		}).Batch(100).To(func(input interface{}) error {
			testResult := input.([]*storage.TestResult)
			err := ss.syncer.store.WriteTestResults(ss.ctx, testResult)
			if err != nil {
				return err
			}
			return nil
		}).Go()
		var result error
		for err := range errorChan {
			result = multierror.Append(err.Err())
		}
		return result
		// TODO: check Post Submit tests as well.
	}

	return nil
}

func (ss *syncState) handleMaintainers(org *config.Org) error {
	scope.Debugf("Getting maintainers for org %s", org.Name)

	maintainers := make(map[string]*storage.Maintainer)

	for _, repo := range org.Repos {
		fc, _, _, err := ss.syncer.gc.ThrottledCallTwoResult(func(client *github.Client) (interface{}, interface{}, *github.Response, error) {
			return client.Repositories.GetContents(ss.ctx, org.Name, repo.Name, "CODEOWNERS", nil)
		})

		if err == nil {
			err = ss.handleCODEOWNERS(org, &repo, maintainers, fc.(*github.RepositoryContent))
		} else {
			err = ss.handleOWNERS(org, &repo, maintainers)
		}

		if err != nil {
			scope.Warnf("Unable to establish maintainers for repo %s/%s: %v", org.Name, repo.Name, err)
		}
	}

	// get the correct case for the maintainer login names, since they are case insensitive in the CODEOWNERS/OWNERS files
	storageMaintainers := make([]*storage.Maintainer, 0, len(maintainers))
	for _, maintainer := range maintainers {
		if u, err := ss.syncer.store.ReadUser(ss.ctx, maintainer.UserLogin); err != nil {
			return fmt.Errorf("unable to read info for maintainer %s from storage: %v", maintainer.UserLogin, err)
		} else if u == nil || u.Name == "" {
			if ghUser, _, err := ss.syncer.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
				return client.Users.Get(ss.ctx, maintainer.UserLogin)
			}); err == nil {
				maintainer.UserLogin = ghUser.(*github.User).GetLogin()
				storageMaintainers = append(storageMaintainers, maintainer)
			} else {
				scope.Warnf("Couldn't get information for maintainer %s: %v", maintainer.UserLogin, err)
			}
		} else {
			storageMaintainers = append(storageMaintainers, maintainer)
		}
	}

	// now compute the contribution info for each maintainer
	for _, maintainer := range maintainers {
		info, err := ss.syncer.store.QueryMaintainerActivity(ss.ctx, maintainer)
		if err != nil {
			scope.Warnf("Couldn't get contribution info for maintainer %s: %v", maintainer.UserLogin, err)
			maintainer.CachedInfo = ""
			continue
		}

		result, err := json.Marshal(info)
		if err != nil {
			scope.Warnf("Couldn't encode contribution info for maintainer %s: %v", maintainer.UserLogin, err)
			maintainer.CachedInfo = ""
			continue
		}

		scope.Debugf("Saving cached contribution info for maintainer %s", maintainer.UserLogin)
		maintainer.CachedInfo = string(result)
	}

	return ss.syncer.store.WriteAllMaintainers(ss.ctx, storageMaintainers)
}

func (ss *syncState) handleCODEOWNERS(org *config.Org, repo *config.Repo, maintainers map[string]*storage.Maintainer, fc *github.RepositoryContent) error {
	content, err := fc.GetContent()
	if err != nil {
		return fmt.Errorf("unable to read CODEOWNERS body from repo %s/%s: %v", org.Name, repo.Name, err)
	}

	lines := strings.Split(content, "\n")

	scope.Debugf("%d lines in CODEOWNERS file for repo %s/%s", len(lines), org.Name, repo.Name)

	// go through each line of the CODEOWNERS file
	for _, line := range lines {
		l := strings.Trim(line, " \t")
		if strings.HasPrefix(l, "#") || l == "" {
			// skip comment lines or empty lines
			continue
		}

		fields := strings.Fields(l)
		logins := fields[1:]

		for _, login := range logins {
			login = strings.TrimPrefix(login, "@")
			login = strings.TrimSuffix(login, ",")

			names, err := ss.expandTeam(org, login)
			if err != nil {
				return err
			}

			for _, name := range names {

				// add the path to this maintainer's list
				path := strings.TrimPrefix(fields[0], "/")
				path = strings.TrimSuffix(path, "/*")
				if path == "*" {
					path = ""
				}

				scope.Debugf("User '%s' can review path '%s/%s/%s'", name, org.Name, repo.Name, path)

				maintainer, err := ss.getMaintainer(org, maintainers, name)
				if maintainer == nil || err != nil {
					scope.Warnf("Couldn't get info on potential maintainer %s: %v", name, err)
					continue
				}

				maintainer.Paths = append(maintainer.Paths, repo.Name+"/"+path)
			}
		}
	}

	return nil
}

func (ss *syncState) expandTeam(org *config.Org, login string) ([]string, error) {
	index := strings.Index(login, "/")
	if index < 0 {
		return []string{login}, nil
	}

	team, _, err := ss.syncer.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Teams.GetTeamBySlug(ss.ctx, org.Name, login[index+1:])
	})

	if err != nil {
		return nil, fmt.Errorf("unable to get information on team %s: %v", login, err)
	}

	id := team.(*github.Team).GetID()

	ghUsers, _, err := ss.syncer.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Teams.ListTeamMembers(ss.ctx, id, nil)
	})

	if err != nil {
		return nil, fmt.Errorf("unable to get members of team %s: %v", login, err)
	}

	var users []string
	for _, u := range ghUsers.([]*github.User) {
		users = append(users, u.GetLogin())
	}

	return users, nil
}

type ownersFile struct {
	Approvers []string `json:"approvers"`
	Reviewers []string `json:"reviewers"`
}

func (ss *syncState) handleOWNERS(org *config.Org, repo *config.Repo, maintainers map[string]*storage.Maintainer) error {
	opt := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	// TODO: we need to get the SHA for the latest commit on the master branch, not just any branch
	rc, _, err := ss.syncer.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Repositories.ListCommits(ss.ctx, org.Name, repo.Name, opt)
	})

	if err != nil {
		return fmt.Errorf("unable to get latest commit in repo %s/%s: %v", org.Name, repo.Name, err)
	}

	tree, _, err := ss.syncer.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Git.GetTree(ss.ctx, org.Name, repo.Name, rc.([]*github.RepositoryCommit)[0].GetSHA(), true)
	})

	if err != nil {
		return fmt.Errorf("unable to get tree in repo %s/%s: %v", org.Name, repo.Name, err)
	}

	files := make(map[string]ownersFile)
	for _, entry := range tree.(*github.Tree).Entries {
		components := strings.Split(entry.GetPath(), "/")
		if components[len(components)-1] == "OWNERS" && components[0] != "vendor" { // HACK: skip Go's vendor directory

			url := "https://raw.githubusercontent.com/" + org.Name + "/" + repo.Name + "/master/" + entry.GetPath()

			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("unable to get %s: %v", url, err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if err != nil {
				return fmt.Errorf("unable to read body for %s: %v", url, err)
			}

			var f ownersFile
			if err := yaml.Unmarshal(body, &f); err != nil {
				return fmt.Errorf("unable to parse body for %s: %v", url, err)
			}

			files[entry.GetPath()] = f
		}
	}

	scope.Debugf("%d OWNERS files found in repo %s/%s", len(files), org.Name, repo.Name)

	for path, file := range files {
		for _, user := range file.Approvers {
			maintainer, err := ss.getMaintainer(org, maintainers, user)
			if maintainer == nil || err != nil {
				scope.Warnf("Couldn't get info on potential maintainer %s: %v", user, err)
				continue
			}

			p := strings.TrimSuffix(path, "OWNERS")

			scope.Debugf("User '%s' can approve path %s/%s/%s", user, org.Name, repo.Name, p)

			maintainer.Paths = append(maintainer.Paths, repo.Name+"/"+p)
		}
	}

	return nil
}

func (ss *syncState) addUsers(users ...string) {
	for _, user := range users {
		ss.users[user] = true
	}
}

func (ss *syncState) getMaintainer(org *config.Org, maintainers map[string]*storage.Maintainer, user string) (*storage.Maintainer, error) {
	maintainer, ok := maintainers[strings.ToUpper(user)]
	if ok {
		// already created a struct
		return maintainer, nil
	}

	ss.addUsers(user)

	maintainer, err := ss.syncer.store.ReadMaintainer(ss.ctx, org.Name, user)
	if err != nil {
		return nil, err
	} else if maintainer == nil {
		// unknown maintainer, so create a record
		maintainer = &storage.Maintainer{
			OrgLogin:  org.Name,
			UserLogin: user,
		}
	}

	maintainers[strings.ToUpper(user)] = maintainer

	return maintainer, nil
}
