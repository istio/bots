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

package syncmgr

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

	"github.com/hashicorp/go-multierror"

	"istio.io/bots/policybot/pkg/pipeline"
	"istio.io/pkg/env"

	"cloud.google.com/go/bigquery"
	"github.com/ghodss/yaml"
	"github.com/google/go-github/v26/github"
	"google.golang.org/api/iterator"

	"istio.io/bots/policybot/handlers/githubwebhook/refresher"
	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/resultgatherer"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/zh"
	"istio.io/pkg/log"
)

// SyncMgr is responsible for synchronizing state from GitHub and ZenHub into our local store
type SyncMgr struct {
	bq        *bigquery.Client
	gc        *gh.ThrottledClient
	zc        *zh.ThrottledClient
	store     storage.Store
	blobstore blobstorage.Store
	robots    map[string]bool
	reg       *config.Registry
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
	Users                    = 1 << 9
)

// The state in SyncMgr is immutable once created. syncState on the other hand represents
// the mutable state used during a single sync operation.
type syncState struct {
	mgr    *SyncMgr
	users  map[string]bool
	flags  FilterFlags
	ctx    context.Context
	dryRun bool
}

var scope = log.RegisterScope("syncmgr", "The GitHub/ZenHub data syncer", 0)

func New(gc *gh.ThrottledClient, zc *zh.ThrottledClient, store storage.Store, bq *bigquery.Client, bs blobstorage.Store,
	reg *config.Registry, robots []string) *SyncMgr {
	r := make(map[string]bool, len(robots))
	for _, robot := range robots {
		r[robot] = true
	}

	return &SyncMgr{
		gc:        gc,
		bq:        bq,
		zc:        zc,
		store:     store,
		blobstore: bs,
		robots:    r,
		reg:       reg,
	}
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
		case "users":
			result |= Users
		default:
			return 0, fmt.Errorf("unknown filter flag %s", f)
		}
	}

	return result, nil
}

func (sm *SyncMgr) Sync(context context.Context, flags FilterFlags, dryRun bool) error {
	ss := &syncState{
		mgr:    sm,
		users:  make(map[string]bool),
		flags:  flags,
		ctx:    context,
		dryRun: dryRun,
	}

	reposByOrg := make(map[string][]string)
	for _, repo := range ss.mgr.reg.Repos() {
		reposByOrg[repo.OrgLogin] = append(reposByOrg[repo.OrgLogin], repo.RepoName)
	}

	var orgs []*storage.Org
	var repos []*storage.Repo

	processedOrgs := make(map[string]bool)
	for _, repo := range sm.reg.Repos() {
		if !processedOrgs[repo.OrgLogin] {
			processedOrgs[repo.OrgLogin] = true

			org, _, err := sm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
				return client.Organizations.Get(context, repo.OrgLogin)
			})

			if err != nil {
				return fmt.Errorf("unable to get information for org %s: %v", repo.OrgLogin, err)
			}

			storageOrg := gh.ConvertOrg(org.(*github.Organization))
			orgs = append(orgs, storageOrg)
		}

		repo, _, err := sm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Repositories.Get(context, repo.OrgLogin, repo.RepoName)
		})

		if err != nil {
			return fmt.Errorf("unable to get information for repo %s: %v", repo, err)
		}

		storageRepo := gh.ConvertRepo(repo.(*github.Repository))
		repos = append(repos, storageRepo)
	}

	if !ss.dryRun {
		if err := sm.store.WriteOrgs(ss.ctx, orgs); err != nil {
			return err
		}

		if err := sm.store.WriteRepos(ss.ctx, repos); err != nil {
			return err
		}
	} else {
		scope.Infof("Would have written %d orgs and %d repos to storage", len(orgs), len(repos))
	}

	for _, repo := range sm.reg.Repos() {
		if err := ss.handleRepo(repo); err != nil {
			return err
		}
	}

	if flags&Maintainers != 0 {
		if err := ss.handleMaintainers(); err != nil {
			return err
		}
	}

	if flags&TestResults != 0 {
		if err := ss.handleTestResults(); err != nil {
			return err
		}
	}

	if err := ss.pushUsers(); err != nil {
		return err
	}

	if ss.flags&Members != 0 {
		if err := ss.handleMembers(); err != nil {
			return err
		}
	}

	if flags&Users != 0 {
		if err := ss.handleUsers(); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) handleUsers() error {
	users := make([]*storage.User, 0, len(ss.users))

	if err := ss.mgr.store.QueryAllUsers(ss.ctx, func(user *storage.User) error {
		if ghUser, _, err := ss.mgr.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Users.Get(ss.ctx, user.UserLogin)
		}); err == nil {
			users = append(users, gh.ConvertUser(ghUser.(*github.User)))
		} else {
			scope.Warnf("couldn't get information for user %s: %v", user.UserLogin, err)
		}

		return nil
	}); err != nil {
		return err
	}

	if ss.dryRun {
		scope.Infof("would have written %d users to storage", len(users))
		return nil
	}

	return ss.mgr.store.WriteUsers(ss.ctx, users)
}

func (ss *syncState) pushUsers() error {
	users := make([]*storage.User, 0, len(ss.users))
	for login := range ss.users {
		if u, err := ss.mgr.store.ReadUser(ss.ctx, login); err != nil {
			return fmt.Errorf("unable to read info for user %s from storage: %v", login, err)
		} else if u == nil || u.Name == "" {
			if ghUser, _, err := ss.mgr.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
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

	if ss.dryRun {
		scope.Infof("would have written %d users to storage", len(users))
		return nil
	}

	return ss.mgr.store.WriteUsers(ss.ctx, users)
}

func (ss *syncState) handleRepo(repo gh.RepoDesc) error {
	scope.Infof("Syncing repo %s", repo)

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
		if err := ss.handleEventsFromBigQuery(repo); err != nil {
			return err
		}

		if err := ss.handleEventsFromGitHub(repo); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) handleActivity(repo gh.RepoDesc, cb func(gh.RepoDesc, time.Time) error,
	getField func(*storage.BotActivity) *time.Time) error {

	start := time.Now().UTC()
	priorStart := time.Time{}

	if activity, _ := ss.mgr.store.ReadBotActivity(ss.ctx, repo.OrgLogin, repo.RepoName); activity != nil {
		priorStart = *getField(activity)
	}

	if err := cb(repo, priorStart); err != nil {
		return err
	}

	if err := ss.mgr.store.UpdateBotActivity(ss.ctx, repo.OrgLogin, repo.RepoName, func(act *storage.BotActivity) error {
		if *getField(act) == priorStart {
			*getField(act) = start
		}
		return nil
	}); err != nil {
		scope.Warnf("unable to update bot activity for repo %s: %v", repo, err)
	}

	return nil
}

func (ss *syncState) handleMembers() error {
	reposByOrg := make(map[string][]string)
	for _, repo := range ss.mgr.reg.Repos() {
		reposByOrg[repo.OrgLogin] = append(reposByOrg[repo.OrgLogin], repo.RepoName)
	}

	var storageMembers []*storage.Member
	for o := range reposByOrg {
		scope.Debugf("Getting members for org %s", o)

		if err := ss.mgr.gc.FetchMembers(ss.ctx, o, func(members []*github.User) error {
			for _, member := range members {
				if ss.mgr.robots[member.GetLogin()] {
					// we don't treat robots as full members...
					continue
				}

				ss.addUsers(member.GetLogin())
				storageMembers = append(storageMembers, &storage.Member{OrgLogin: o, UserLogin: member.GetLogin()})
			}

			return nil
		}); err != nil {
			return err
		}

		// now compute the contribution info for each member
		for _, member := range storageMembers {
			info, err := ss.mgr.store.QueryMemberActivity(ss.ctx, member, reposByOrg[o])
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

			scope.Debugf("Computed cached contribution info for member %s", member.UserLogin)
			member.CachedInfo = string(result)
		}
	}

	if ss.dryRun {
		scope.Infof("would have written %d members to storage", len(storageMembers))
		return nil
	}

	return ss.mgr.store.WriteAllMembers(ss.ctx, storageMembers)
}

func (ss *syncState) handleLabels(repo gh.RepoDesc) error {
	scope.Debugf("Getting labels from repo %s", repo)

	return ss.mgr.gc.FetchLabels(ss.ctx, repo.OrgLogin, repo.RepoName, func(labels []*github.Label) error {
		storageLabels := make([]*storage.Label, 0, len(labels))
		for _, label := range labels {
			storageLabels = append(storageLabels, gh.ConvertLabel(repo.OrgLogin, repo.RepoName, label))
		}

		if ss.dryRun {
			scope.Infof("would have written %d labels from repo %s to storage", len(storageLabels), repo)
			return nil
		}

		return ss.mgr.store.WriteLabels(ss.ctx, storageLabels)
	})
}

func (ss *syncState) handleEventsFromBigQuery(repo gh.RepoDesc) error {
	scope.Debugf("Getting events from repo %s", repo)

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

	q := ss.mgr.bq.Query(fmt.Sprintf(`
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
				if !ss.dryRun {
					if err := ss.mgr.store.WriteIssueEvents(ss.ctx, issueEvents); err != nil {
						return fmt.Errorf("unable to write issue events to storage: %v", err)
					}
				} else {
					scope.Infof("Would have written %d issue events for repo %s to storage", len(issueEvents), repo)
				}
				issueEvents = issueEvents[:0]
			}

			if len(issueCommentEvents) > 0 {
				if !ss.dryRun {
					if err := ss.mgr.store.WriteIssueCommentEvents(ss.ctx, issueCommentEvents); err != nil {
						return fmt.Errorf("unable to write issue comment events to storage: %v", err)
					}
				} else {
					scope.Infof("Would have written %d issue comment events for repo %s to storage", len(issueCommentEvents), repo)
				}
				issueCommentEvents = issueCommentEvents[:0]
			}

			if len(prEvents) > 0 {
				if !ss.dryRun {
					if err := ss.mgr.store.WritePullRequestEvents(ss.ctx, prEvents); err != nil {
						return fmt.Errorf("unable to write pull request events to storage: %v", err)
					}
				} else {
					scope.Infof("Would have written %d pr events for repo %s to storage", len(prEvents), repo)
				}
				prEvents = prEvents[:0]
			}

			if len(prCommentEvents) > 0 {
				if !ss.dryRun {
					if err := ss.mgr.store.WritePullRequestReviewCommentEvents(ss.ctx, prCommentEvents); err != nil {
						return fmt.Errorf("unable to write pull request review comment events to storage: %v", err)
					}
				} else {
					scope.Infof("Would have written %d pr comment events for repo %s to storage", len(prCommentEvents), repo)
				}
				prCommentEvents = prCommentEvents[:0]
			}

			if len(prReviewEvents) > 0 {
				if !ss.dryRun {
					if err := ss.mgr.store.WritePullRequestReviewEvents(ss.ctx, prReviewEvents); err != nil {
						return fmt.Errorf("unable to write pull request review events to storage: %v", err)
					}
				} else {
					scope.Infof("Would have written %d pr review events for repo %s to storage", len(prReviewEvents), repo)
				}
				prReviewEvents = prReviewEvents[:0]
			}

			if done {
				return nil
			}
		}
	}
}

func (ss *syncState) handleEventsFromGitHub(repo gh.RepoDesc) error {
	scope.Debugf("Getting events from repo %s", repo)

	total := 0
	err := ss.mgr.gc.FetchRepoEvents(ss.ctx, repo.OrgLogin, repo.RepoName, func(events []*github.Event) error {
		var issueEvents []*storage.IssueEvent
		var issueCommentEvents []*storage.IssueCommentEvent
		var prEvents []*storage.PullRequestEvent
		var prCommentEvents []*storage.PullRequestReviewCommentEvent
		var prReviewEvents []*storage.PullRequestReviewEvent

		total += len(events)
		scope.Infof("Received %d events", total)

		for _, event := range events {
			switch *event.Type {
			case "IssueEvent":
				payload, err := event.ParsePayload()
				if err != nil {
					scope.Errorf("unable to parse payload for issue event: %v", err)
					continue
				}

				p := payload.(*github.IssueEvent)
				issueEvents = append(issueEvents, &storage.IssueEvent{
					OrgLogin:    repo.OrgLogin,
					RepoName:    repo.RepoName,
					IssueNumber: int64(p.GetIssue().GetNumber()),
					CreatedAt:   event.GetCreatedAt(),
					Actor:       event.GetActor().GetLogin(),
					Action:      p.GetEvent(),
				})

			case "IssueCommentEvent":
				payload, err := event.ParsePayload()
				if err != nil {
					scope.Errorf("unable to parse payload for issue comment event: %v", err)
					continue
				}

				p := payload.(*github.IssueCommentEvent)

				issueCommentEvents = append(issueCommentEvents, &storage.IssueCommentEvent{
					OrgLogin:       repo.OrgLogin,
					RepoName:       repo.RepoName,
					IssueNumber:    int64(p.GetIssue().GetNumber()),
					IssueCommentID: p.GetComment().GetID(),
					CreatedAt:      event.GetCreatedAt(),
					Actor:          event.GetActor().GetLogin(),
					Action:         p.GetAction(),
				})

			case "PullRequestEvent":
				payload, err := event.ParsePayload()
				if err != nil {
					scope.Errorf("unable to parse payload for pull request event: %v", err)
					continue
				}

				p := payload.(*github.PullRequestEvent)
				prEvents = append(prEvents, &storage.PullRequestEvent{
					OrgLogin:          repo.OrgLogin,
					RepoName:          repo.RepoName,
					PullRequestNumber: int64(p.GetPullRequest().GetNumber()),
					CreatedAt:         event.GetCreatedAt(),
					Actor:             event.GetActor().GetLogin(),
					Action:            p.GetAction(),
				})

			case "PullRequestCommentEvent":
				payload, err := event.ParsePayload()
				if err != nil {
					scope.Errorf("unable to parse payload for pull request review comment event: %v", err)
					continue
				}

				p := payload.(*github.PullRequestReviewCommentEvent)
				prCommentEvents = append(prCommentEvents, &storage.PullRequestReviewCommentEvent{
					OrgLogin:                   repo.OrgLogin,
					RepoName:                   repo.RepoName,
					PullRequestNumber:          int64(p.GetPullRequest().GetNumber()),
					PullRequestReviewCommentID: p.GetComment().GetID(),
					CreatedAt:                  event.GetCreatedAt(),
					Actor:                      event.GetActor().GetLogin(),
					Action:                     p.GetAction(),
				})

			case "PullRequestReviewEvent":
				payload, err := event.ParsePayload()
				if err != nil {
					scope.Errorf("unable to parse payload for pull request review event: %v", err)
					continue
				}

				p := payload.(*github.PullRequestReviewEvent)
				prReviewEvents = append(prReviewEvents, &storage.PullRequestReviewEvent{
					OrgLogin:            repo.OrgLogin,
					RepoName:            repo.RepoName,
					PullRequestNumber:   int64(p.GetPullRequest().GetNumber()),
					PullRequestReviewID: p.GetReview().GetID(),
					CreatedAt:           event.GetCreatedAt(),
					Actor:               event.GetActor().GetLogin(),
					Action:              p.GetAction(),
				})
			}
		}

		if len(issueEvents) > 0 {
			if !ss.dryRun {
				if err := ss.mgr.store.WriteIssueEvents(ss.ctx, issueEvents); err != nil {
					return fmt.Errorf("unable to write issue events to storage: %v", err)
				}
			} else {
				scope.Infof("Would have written %d issue events for repo %s to storage", len(issueEvents), repo)
			}
		}

		if len(issueCommentEvents) > 0 {
			if !ss.dryRun {
				if err := ss.mgr.store.WriteIssueCommentEvents(ss.ctx, issueCommentEvents); err != nil {
					return fmt.Errorf("unable to write issue comment events to storage: %v", err)
				}
			} else {
				scope.Infof("Would have written %d issue comment events for repo %s to storage", len(issueCommentEvents), repo)
			}
		}

		if len(prEvents) > 0 {
			if !ss.dryRun {
				if err := ss.mgr.store.WritePullRequestEvents(ss.ctx, prEvents); err != nil {
					return fmt.Errorf("unable to write pull request events to storage: %v", err)
				}
			} else {
				scope.Infof("Would have written %d pr events for repo %s to storage", len(prEvents), repo)
			}
		}

		if len(prCommentEvents) > 0 {
			if !ss.dryRun {
				if err := ss.mgr.store.WritePullRequestReviewCommentEvents(ss.ctx, prCommentEvents); err != nil {
					return fmt.Errorf("unable to write pull request review comment events to storage: %v", err)
				}
			} else {
				scope.Infof("Would have written %d pr review comment events for repo %s to storage", len(prCommentEvents), repo)
			}
		}

		if len(prReviewEvents) > 0 {
			if !ss.dryRun {
				if err := ss.mgr.store.WritePullRequestReviewEvents(ss.ctx, prReviewEvents); err != nil {
					return fmt.Errorf("unable to write pull request review events to storage: %v", err)
				}
			} else {
				scope.Infof("Would have written %d pr review events for repo %s to storage", len(prReviewEvents), repo)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return ss.mgr.gc.FetchIssueEvents(ss.ctx, repo.OrgLogin, repo.RepoName, func(events []*github.IssueEvent) error {
		var issueEvents []*storage.IssueEvent

		total += len(events)
		scope.Infof("Received %d events", total)

		for _, event := range events {
			issueEvents = append(issueEvents, &storage.IssueEvent{
				OrgLogin:    repo.OrgLogin,
				RepoName:    repo.RepoName,
				IssueNumber: int64(event.GetIssue().GetNumber()),
				CreatedAt:   event.GetCreatedAt(),
				Actor:       event.GetActor().GetLogin(),
				Action:      event.GetEvent(),
			})
		}

		if len(issueEvents) > 0 {
			if !ss.dryRun {
				if err := ss.mgr.store.WriteIssueEvents(ss.ctx, issueEvents); err != nil {
					return fmt.Errorf("unable to write issue events to storage: %v", err)
				}
			} else {
				scope.Infof("Would have written %d issue events for repo %s to storage", len(issueEvents), repo)
			}
		}

		return nil
	})
}

func (ss *syncState) handleRepoComments(repo gh.RepoDesc) error {
	scope.Debugf("Getting comments for repo %s", repo)

	return ss.mgr.gc.FetchRepoComments(ss.ctx, repo.OrgLogin, repo.RepoName, func(comments []*github.RepositoryComment) error {
		storageComments := make([]*storage.RepoComment, 0, len(comments))
		for _, comment := range comments {
			t := gh.ConvertRepoComment(repo.OrgLogin, repo.RepoName, comment)
			storageComments = append(storageComments, t)
			ss.addUsers(t.Author)
		}

		if ss.dryRun {
			scope.Infof("Would have written %d repo comments for repo %s to storage", len(storageComments), repo)
			return nil
		}

		return ss.mgr.store.WriteRepoComments(ss.ctx, storageComments)
	})
}

func (ss *syncState) handleIssues(repo gh.RepoDesc, startTime time.Time) error {
	scope.Debugf("Getting issues from repo %s", repo)

	total := 0
	return ss.mgr.gc.FetchIssues(ss.ctx, repo.OrgLogin, repo.RepoName, startTime, func(issues []*github.Issue) error {
		var storageIssues []*storage.Issue

		total += len(issues)
		scope.Infof("Received %d issues", total)

		for _, issue := range issues {
			t := gh.ConvertIssue(repo.OrgLogin, repo.RepoName, issue)
			storageIssues = append(storageIssues, t)
			ss.addUsers(t.Author)
			ss.addUsers(t.Assignees...)
		}

		if ss.dryRun {
			scope.Infof("Would have written %d issues for repo %s to storage", len(storageIssues), repo)
			return nil
		}

		return ss.mgr.store.WriteIssues(ss.ctx, storageIssues)
	})
}

func (ss *syncState) handleIssueComments(repo gh.RepoDesc, startTime time.Time) error {
	scope.Debugf("Getting issue comments from repo %s", repo)

	total := 0
	return ss.mgr.gc.FetchIssueComments(ss.ctx, repo.OrgLogin, repo.RepoName, startTime, func(comments []*github.IssueComment) error {
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

		if ss.dryRun {
			scope.Infof("Would have written %d issue comments for repo %s to storage", len(storageIssueComments), repo)
			return nil
		}

		return ss.mgr.store.WriteIssueComments(ss.ctx, storageIssueComments)
	})
}

func (ss *syncState) handleZenHub(repo gh.RepoDesc) error {
	scope.Debugf("Getting ZenHub issue data for repo %s", repo)

	// get all the issues
	var issues []*storage.Issue
	if err := ss.mgr.store.QueryIssuesByRepo(ss.ctx, repo.OrgLogin, repo.RepoName, func(issue *storage.Issue) error {
		issues = append(issues, issue)
		return nil
	}); err != nil {
		return fmt.Errorf("unable to read issues from repo %s: %v", repo, err)
	}

	sr, err := ss.mgr.store.ReadRepo(ss.ctx, repo.OrgLogin, repo.RepoName)
	if err != nil {
		return fmt.Errorf("unable to read information about repo %s from storage", repo)
	}

	// now get the ZenHub data for all issues
	var pipelines []*storage.IssuePipeline
	for _, issue := range issues {
		issueData, err := ss.mgr.zc.ThrottledCall(func(client *zh.Client) (interface{}, error) {
			return client.GetIssueData(int(sr.RepoNumber), int(issue.IssueNumber))
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
			if err = ss.mgr.store.WriteIssuePipelines(ss.ctx, pipelines); err != nil {
				return err
			}
			pipelines = pipelines[:0]
		}
	}

	if ss.dryRun {
		scope.Infof("Would have written %d issue pipelines for repo %s to storage", len(pipelines), repo)
		return nil
	}

	return ss.mgr.store.WriteIssuePipelines(ss.ctx, pipelines)
}

func (ss *syncState) handlePullRequests(repo gh.RepoDesc) error {
	scope.Debugf("Getting pull requests from repo %s", repo)

	total := 0
	return ss.mgr.gc.FetchPullRequests(ss.ctx, repo.OrgLogin, repo.RepoName, func(prs []*github.PullRequest) error {
		var storagePRs []*storage.PullRequest
		var storagePRReviews []*storage.PullRequestReview

		total += len(prs)
		scope.Infof("Received %d pull requests", total)

		for _, pr := range prs {
			// if this pr is already known to us and is up to date, skip further processing
			if existing, _ := ss.mgr.store.ReadPullRequest(ss.ctx, repo.OrgLogin, repo.RepoName, pr.GetNumber()); existing != nil {
				if existing.UpdatedAt == pr.GetUpdatedAt() {
					continue
				}
			}

			if err := ss.mgr.gc.FetchReviews(ss.ctx, repo.OrgLogin, repo.RepoName, pr.GetNumber(), func(reviews []*github.PullRequestReview) error {
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
			if err := ss.mgr.gc.FetchFiles(ss.ctx, repo.OrgLogin, repo.RepoName, pr.GetNumber(), func(files []string) error {
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

		if ss.dryRun {
			scope.Infof("Would have written %d prs and %d pr reviews for repo %s to storage", len(storagePRs), len(storagePRReviews), repo)
			return nil
		}

		err := ss.mgr.store.WritePullRequests(ss.ctx, storagePRs)
		if err == nil {
			err = ss.mgr.store.WritePullRequestReviews(ss.ctx, storagePRReviews)
		}

		return err
	})
}

func (ss *syncState) handlePullRequestReviewComments(repo gh.RepoDesc, start time.Time) error {
	scope.Debugf("Getting pull requests review comments from repo %s", repo)

	total := 0
	return ss.mgr.gc.FetchPullRequestReviewComments(ss.ctx, repo.OrgLogin, repo.RepoName, start, func(comments []*github.PullRequestComment) error {
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

		if ss.dryRun {
			scope.Infof("Would have written %d pr review comments for repo %s to storage", len(storagePRComments), repo)
			return nil
		}

		return ss.mgr.store.WritePullRequestReviewComments(ss.ctx, storagePRComments)
	})
}

func (ss *syncState) handleTestResults() error {
	for _, repo := range ss.mgr.reg.Repos() {
		r, ok := ss.mgr.reg.SingleRecord(refresher.RecordType, repo.OrgAndRepo)
		if !ok {
			continue
		}

		tor := r.(*refresher.TestOutputRecord)
		g := resultgatherer.TestResultGatherer{
			Client:           ss.mgr.blobstore,
			BucketName:       tor.BucketName,
			PreSubmitPrefix:  tor.PreSubmitTestPath,
			PostSubmitPrefix: tor.PostSubmitTestPath,
		}

		scope.Debugf("Getting test results for org %s", repo.OrgLogin)
		prMin := env.RegisterIntVar("PR_MIN", 0, "The minimum PR to scan for test results").Get()
		prMax := env.RegisterIntVar("PR_MAX", -1, "The maximum PR to scan for test results").Get()

		var completedTests = make(map[string]bool)
		ctLock := sync.RWMutex{}
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			ctLock.Lock()
			defer ctLock.Unlock()
			err := ss.mgr.store.QueryTestResultByDone(ss.ctx, repo.OrgLogin, repo.RepoName,
				func(result *storage.TestResult) error {
					completedTests[result.RunPath] = true
					return nil
				})
			if err != nil {
				scope.Warnf("Unable to fetch previous tests: %s", err)
			}
			wg.Done()
		}()
		prPaths := g.GetAllPullRequestsChan(ss.ctx, repo.OrgLogin, repo.RepoName).WithBuffer(100)
		// I think a composition syntax would be better here...
		errorChan := prPaths.Transform(func(prPathi interface{}) (prNum interface{}, err error) {
			prPath := prPathi.(string)
			prParts := strings.Split(prPath, "/")
			if len(prParts) < 2 {
				err = errors.New("too few segments in pr path")
				return
			}
			prNum = prParts[len(prParts)-2]
			if prInt, ierr := strconv.Atoi(prParts[len(prParts)-2]); ierr == nil {
				// skip this PR if it's outside the min and max inclusive
				if prInt < prMin || (prMax > -1 && prInt > prMax) {
					err = pipeline.ErrSkip
				}
			}
			return
		}).WithContext(ss.ctx).OnError(func(e error) {
			// TODO: this should probably be reported out or something...
			scope.Warnf("error processing test: %s", e)
		}).WithParallelism(50).Transform(func(prNumi interface{}) (testRunPaths interface{}, err error) {
			prNum := prNumi.(string)
			tests, err := g.GetTestsForPR(ss.ctx, repo.OrgLogin, repo.RepoName, prNum)
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
			if strings.Contains(testRunPath, "00") {
				fmt.Printf("checking test %s\n", testRunPath)
			}
			return g.GetTestResult(ss.ctx, testName, testRunPath)
		}).Batch(50).To(func(input interface{}) error {
			var testResults []*storage.TestResult
			for _, i := range input.([]interface{}) {
				singleResult := i.(*storage.TestResult)
				singleResult.OrgLogin = repo.OrgLogin
				singleResult.RepoName = repo.RepoName
				testResults = append(testResults, singleResult)
			}
			fmt.Printf("saving TestResult batch of size %d\n", len(testResults))
			err := ss.mgr.store.WriteTestResults(ss.ctx, testResults)
			if err != nil {
				return err
			}
			return nil
		}).WithParallelism(1).Go()
		var result *multierror.Error
		for err := range errorChan {
			result = multierror.Append(err.Err())
		}
		if result != nil {
			return result
		}
		// Update cache table to reflect these results.
		rowCount, err := ss.mgr.store.UpdateFlakeCache(ss.ctx)
		if err != nil {
			return err
		}
		log.Infof("Updated flake cache with %d additional flakes", rowCount)
		// TODO: check Post Submit tests as well.
	}

	return nil
}

func (ss *syncState) handlePostSubmitTestResults() error {
	for _, repo := range ss.mgr.reg.Repos() {
		r, ok := ss.mgr.reg.SingleRecord(refresher.RecordType, repo.OrgAndRepo)
		if !ok {
			continue
		}

		tor := r.(*refresher.TestOutputRecord)
		g := resultgatherer.TestResultGatherer{
			Client:           ss.mgr.blobstore,
			BucketName:       tor.BucketName,
			PreSubmitPrefix:  tor.PreSubmitTestPath,
			PostSubmitPrefix: tor.PostSubmitTestPath,
		}

		scope.Debugf("Getting post submit test results for org %s", repo.OrgLogin)
		var completedTests = make(map[string]bool)
		ctLock := sync.RWMutex{}
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			ctLock.Lock()
			defer ctLock.Unlock()
			err := ss.mgr.store.QueryPostSubmitTestResultByDone(ss.ctx, repo.OrgLogin, repo.RepoName,
				func(result *storage.PostSubmitTestResult) error {
					completedTests[result.RunPath] = true
					return nil
				})
			if err != nil {
				scope.Warnf("Unable to fetch previous post submit tests: %s", err)
			}
			wg.Done()
		}()
		Paths := g.GetAllPostSubmitTestChan(ss.ctx).WithBuffer(100)
		// I think a composition syntax would be better here...
		errorChan := Paths.WithContext(ss.ctx).OnError(func(e error) {
			// TODO: this should probably be reported out or something...
			scope.Warnf("error processing post submit test: %s", e)
		}).WithParallelism(50).Transform(func(pathi interface{}) (testRunPaths interface{}, err error) {
			var result [][]string

			// Wait for a comprehensive list of completed tests
			wg.Wait()
			ctLock.RLock()
			defer ctLock.RUnlock()

			bucket := g.Client.Bucket(g.BucketName)
			testPref := pathi.(string)
			testPrefSplit := strings.Split(testPref, "/")
			testName := testPrefSplit[len(testPrefSplit)-2]
			runPaths, err := bucket.ListPrefixes(ss.ctx, testPref)
			if err != nil {
				return nil, err
			}
			for _, runPath := range runPaths {
				if _, ok := completedTests[runPath]; !ok {
					result = append(result, []string{testName, runPath})
				}
			}
			testRunPaths = result
			return
		}).Expand().Transform(func(testRunPathi interface{}) (i interface{}, err error) {
			inputArray := testRunPathi.([]string)
			testRunPath := inputArray[1]
			testName := inputArray[0]
			if strings.Contains(testRunPath, "00") {
				fmt.Printf("checking post submit test %s\n", testRunPath)
			}
			return g.GetPostSubmitTestResult(ss.ctx, testName, testRunPath)
		}).Batch(50).To(func(input interface{}) error {
			var testResults []*storage.PostSubmitTestResult
			for _, i := range input.([]interface{}) {
				singleResult := i.(*storage.PostSubmitTestResult)
				singleResult.OrgLogin = repo.OrgLogin
				singleResult.RepoName = repo.RepoName
				testResults = append(testResults, singleResult)
			}
			fmt.Printf("saving PostSubmitTestResult batch of size %d\n", len(testResults))
			err := ss.mgr.store.WritePostSumbitTestResults(ss.ctx, testResults)
			if err != nil {
				return err
			}
			return nil
		}).WithParallelism(1).Go()
		var result *multierror.Error
		for err := range errorChan {
			result = multierror.Append(err.Err())
		}
		if result != nil {
			return result
		}
		// Update cache table to reflect these results.
		rowCount, err := ss.mgr.store.UpdateFlakeCache(ss.ctx)
		if err != nil {
			return err
		}
		log.Infof("Updated flake cache with %d additional flakes", rowCount)
	}

	return nil
}

func (ss *syncState) handleMaintainers() error {
	maintainers := make(map[string]*storage.Maintainer)

	for _, repo := range ss.mgr.reg.Repos() {
		scope.Debugf("Getting maintainers for repo %s", repo)

		fc, _, _, err := ss.mgr.gc.ThrottledCallTwoResult(func(client *github.Client) (interface{}, interface{}, *github.Response, error) {
			return client.Repositories.GetContents(ss.ctx, repo.OrgLogin, repo.RepoName, "CODEOWNERS", nil)
		})

		if err == nil {
			err = ss.handleCODEOWNERS(repo, maintainers, fc.(*github.RepositoryContent))
		} else {
			err = ss.handleOWNERS(repo, maintainers)
		}

		if err != nil {
			scope.Warnf("Unable to establish maintainers for repo %s: %v", repo, err)
		}
	}

	// get the correct case for the maintainer login names, since they are case insensitive in the CODEOWNERS/OWNERS files
	storageMaintainers := make([]*storage.Maintainer, 0, len(maintainers))
	for _, maintainer := range maintainers {
		if u, err := ss.mgr.store.ReadUser(ss.ctx, maintainer.UserLogin); err != nil {
			return fmt.Errorf("unable to read info for maintainer %s from storage: %v", maintainer.UserLogin, err)
		} else if u == nil || u.Name == "" {
			if ghUser, _, err := ss.mgr.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
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
		info, err := ss.mgr.store.QueryMaintainerActivity(ss.ctx, maintainer)
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

	if ss.dryRun {
		scope.Infof("Would have written %d maintainers to storage", len(maintainers))
		return nil
	}

	return ss.mgr.store.WriteAllMaintainers(ss.ctx, storageMaintainers)
}

func (ss *syncState) handleCODEOWNERS(repo gh.RepoDesc, maintainers map[string]*storage.Maintainer, fc *github.RepositoryContent) error {
	content, err := fc.GetContent()
	if err != nil {
		return fmt.Errorf("unable to read CODEOWNERS body from repo %s: %v", repo, err)
	}

	lines := strings.Split(content, "\n")

	scope.Debugf("%d lines in CODEOWNERS file for repo %s", len(lines), repo)

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

			names, err := ss.expandTeam(repo.OrgLogin, login)
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

				scope.Debugf("User '%s' can review path '%s/%s'", name, repo, path)

				maintainer, err := ss.getMaintainer(repo.OrgLogin, maintainers, name)
				if maintainer == nil || err != nil {
					scope.Warnf("Couldn't get info on potential maintainer %s: %v", name, err)
					continue
				}

				maintainer.Paths = append(maintainer.Paths, repo.RepoName+"/"+path)
			}
		}
	}

	return nil
}

func (ss *syncState) expandTeam(orgLogin string, teamLogin string) ([]string, error) {
	index := strings.Index(teamLogin, "/")
	if index < 0 {
		return []string{teamLogin}, nil
	}

	team, _, err := ss.mgr.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Teams.GetTeamBySlug(ss.ctx, orgLogin, teamLogin[index+1:])
	})

	if err != nil {
		return nil, fmt.Errorf("unable to get information on team %s: %v", teamLogin, err)
	}

	id := team.(*github.Team).GetID()

	ghUsers, _, err := ss.mgr.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Teams.ListTeamMembers(ss.ctx, id, nil)
	})

	if err != nil {
		return nil, fmt.Errorf("unable to get members of team %s: %v", teamLogin, err)
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

func (ss *syncState) handleOWNERS(repo gh.RepoDesc, maintainers map[string]*storage.Maintainer) error {
	opt := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	// TODO: we need to get the SHA for the latest commit on the master branch, not just any branch
	rc, _, err := ss.mgr.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Repositories.ListCommits(ss.ctx, repo.OrgLogin, repo.RepoName, opt)
	})

	if err != nil {
		return fmt.Errorf("unable to get latest commit in repo %s: %v", repo, err)
	}

	tree, _, err := ss.mgr.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Git.GetTree(ss.ctx, repo.OrgLogin, repo.RepoName, rc.([]*github.RepositoryCommit)[0].GetSHA(), true)
	})

	if err != nil {
		return fmt.Errorf("unable to get tree in repo %s: %v", repo, err)
	}

	files := make(map[string]ownersFile)
	for _, entry := range tree.(*github.Tree).Entries {
		components := strings.Split(entry.GetPath(), "/")
		if components[len(components)-1] == "OWNERS" && components[0] != "vendor" { // HACK: skip Go's vendor directory

			url := "https://raw.githubusercontent.com/" + repo.OrgAndRepo + "/master/" + entry.GetPath()

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

	scope.Debugf("%d OWNERS files found in repo %s", len(files), repo)

	for path, file := range files {
		for _, user := range file.Approvers {
			maintainer, err := ss.getMaintainer(repo.OrgLogin, maintainers, user)
			if maintainer == nil || err != nil {
				scope.Warnf("Couldn't get info on potential maintainer %s: %v", user, err)
				continue
			}

			p := strings.TrimSuffix(path, "OWNERS")

			scope.Debugf("User '%s' can approve path %s/%s", user, repo, p)

			maintainer.Paths = append(maintainer.Paths, repo.RepoName+"/"+p)
		}
	}

	return nil
}

func (ss *syncState) addUsers(users ...string) {
	for _, user := range users {
		ss.users[user] = true
	}
}

func (ss *syncState) getMaintainer(orgLogin string, maintainers map[string]*storage.Maintainer, user string) (*storage.Maintainer, error) {
	maintainer, ok := maintainers[strings.ToUpper(user)]
	if ok {
		// already created a struct
		return maintainer, nil
	}

	ss.addUsers(user)

	maintainer, err := ss.mgr.store.ReadMaintainer(ss.ctx, orgLogin, user)
	if err != nil {
		return nil, err
	} else if maintainer == nil {
		// unknown maintainer, so create a record
		maintainer = &storage.Maintainer{
			OrgLogin:  orgLogin,
			UserLogin: user,
		}
	}

	maintainers[strings.ToUpper(user)] = maintainer

	return maintainer, nil
}
