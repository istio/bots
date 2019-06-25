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
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/zh"
	"istio.io/pkg/log"
)

// Syncer is responsible for synchronizing state from GitHub and ZenHub into our local store
type Syncer struct {
	cache *cache.Cache
	ght   *gh.ThrottledClient
	zht   *zh.ThrottledClient
	store storage.Store
	bs    blobstorage.Store
	orgs  []config.Org
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
)

// The state in Syncer is immutable once created. syncState on the other hand represents
// the mutable state used during a single sync operation.
type syncState struct {
	syncer *Syncer
	users  map[string]*storage.User
	flags  FilterFlags
	ctx    context.Context
}

var scope = log.RegisterScope("syncer", "The GitHub data syncer", 0)

func New(ght *gh.ThrottledClient, cache *cache.Cache,
	zht *zh.ThrottledClient, store storage.Store, bs blobstorage.Store, orgs []config.Org) *Syncer {
	return &Syncer{
		ght:   ght,
		cache: cache,
		zht:   zht,
		store: store,
		orgs:  orgs,
		bs:    bs,
	}
}

func ConvFilterFlags(filter string) (FilterFlags, error) {
	if filter == "" {
		// defaults to everything
		return Issues | Prs | Maintainers | Members | Labels | ZenHub, nil
	}

	var result FilterFlags
	for _, f := range strings.Split(filter, ",") {
		switch f {
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
		default:
			return 0, fmt.Errorf("unknown filter flag %s", f)
		}
	}

	return result, nil
}

func (s *Syncer) Sync(context context.Context, flags FilterFlags) error {
	ss := &syncState{
		syncer: s,
		users:  make(map[string]*storage.User),
		flags:  flags,
		ctx:    context,
	}

	var orgs []*storage.Org
	var repos []*storage.Repo

	// get all the org & repo info
	if err := s.fetchOrgs(ss.ctx, func(org *github.Organization) error {
		orgs = append(orgs, gh.OrgFromAPI(org))
		return s.fetchRepos(ss.ctx, func(repo *github.Repository) error {
			repos = append(repos, gh.RepoFromAPI(repo))
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

	if flags&(Members|Labels|Issues|Prs|ZenHub|RepoComments) != 0 {
		for _, org := range orgs {
			var orgRepos []*storage.Repo
			for _, repo := range repos {
				if repo.OrgID == org.OrgID {
					orgRepos = append(orgRepos, repo)
				}
			}

			if err := ss.handleOrg(org, orgRepos); err != nil {
				return err
			}
		}

		if err := ss.pushUsers(); err != nil {
			return err
		}
	}

	if flags&Maintainers != 0 {
		for _, org := range orgs {
			var orgRepos []*storage.Repo
			for _, repo := range repos {
				if repo.OrgID == org.OrgID {
					orgRepos = append(orgRepos, repo)
				}
			}

			if err := ss.handleMaintainers(org, orgRepos); err != nil {
				return err
			}
		}

		if err := ss.pushUsers(); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) pushUsers() error {
	users := make([]*storage.User, 0, len(ss.users))
	for _, user := range ss.users {
		users = append(users, user)
	}

	if err := ss.syncer.store.WriteUsers(ss.ctx, users); err != nil {
		return err
	}

	return nil
}

func (ss *syncState) handleOrg(org *storage.Org, repos []*storage.Repo) error {
	scope.Infof("Syncing org %s", org.Login)

	if ss.flags&Members != 0 {
		if err := ss.handleMembers(org); err != nil {
			return err
		}
	}

	for _, repo := range repos {
		if err := ss.handleRepo(org, repo); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) handleRepo(org *storage.Org, repo *storage.Repo) error {
	scope.Infof("Syncing repo %s/%s", org.Login, repo.Name)

	if ss.flags&Labels != 0 {
		if err := ss.handleLabels(org, repo); err != nil {
			return err
		}
	}

	if ss.flags&Issues != 0 {
		start := time.Now().UTC()
		priorStart := time.Time{}
		if activity, _ := ss.syncer.store.ReadBotActivityByID(ss.ctx, org.OrgID, repo.RepoID); activity != nil {
			priorStart = activity.LastIssueSyncStart
		}

		if err := ss.handleIssues(org, repo, priorStart); err != nil {
			return err
		}

		activity := storage.BotActivity{
			OrgID:              org.OrgID,
			RepoID:             repo.RepoID,
			LastIssueSyncStart: start,
		}

		if err := ss.syncer.store.WriteBotActivities(ss.ctx, []*storage.BotActivity{&activity}); err != nil {
			scope.Warnf("unable to update bot activity for repo %s/%s: %v", org.Login, repo.Name, err)
		}
	}

	if ss.flags&ZenHub != 0 {
		if err := ss.handleZenHub(org, repo); err != nil {
			return err
		}
	}

	if ss.flags&Prs != 0 {
		if err := ss.handlePullRequests(org, repo); err != nil {
			return err
		}
	}

	if ss.flags&RepoComments != 0 {
		if err := ss.handleRepoComments(org, repo); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) handleMembers(org *storage.Org) error {
	scope.Debugf("Getting members from org %s", org.Login)

	var storageMembers []*storage.Member
	if err := ss.syncer.fetchMembers(ss.ctx, org, func(members []*github.User) error {
		for _, member := range members {
			ss.addUser(member)
			storageMembers = append(storageMembers, &storage.Member{OrgID: org.OrgID, UserID: member.GetNodeID()})
		}

		return nil
	}); err != nil {
		return err
	}

	return ss.syncer.store.WriteAllMembers(ss.ctx, storageMembers)
}

func (ss *syncState) handleLabels(org *storage.Org, repo *storage.Repo) error {
	scope.Debugf("Getting labels from repo %s/%s", org.Login, repo.Name)

	return ss.syncer.fetchLabels(ss.ctx, org, repo, func(labels []*github.Label) error {
		storageLabels := make([]*storage.Label, 0, len(labels))
		for _, label := range labels {
			storageLabels = append(storageLabels, gh.LabelFromAPI(org.OrgID, repo.RepoID, label))
		}

		return ss.syncer.store.WriteLabels(ss.ctx, storageLabels)
	})
}

func (ss *syncState) handleRepoComments(org *storage.Org, repo *storage.Repo) error {
	scope.Debugf("Getting comments for repo %s/%s", org.Login, repo.Name)

	return ss.syncer.fetchRepoComments(ss.ctx, org, repo, func(comments []*github.RepositoryComment) error {
		storageComments := make([]*storage.RepoComment, 0, len(comments))
		for _, comment := range comments {
			storageComments = append(storageComments, gh.RepoCommentFromAPI(org.OrgID, repo.RepoID, comment))
		}

		return ss.syncer.store.WriteRepoComments(ss.ctx, storageComments)
	})
}

func (ss *syncState) handleIssues(org *storage.Org, repo *storage.Repo, startTime time.Time) error {
	scope.Debugf("Getting issues from repo %s/%s", org.Login, repo.Name)

	return ss.syncer.fetchIssues(ss.ctx, org, repo, startTime, func(issues []*github.Issue) error {
		var storageIssues []*storage.Issue
		var storageIssueComments []*storage.IssueComment

		for _, issue := range issues {
			// if this issue is already known to us and is up to date, skip further processing
			if existing, _ := ss.syncer.cache.ReadIssue(ss.ctx, org.OrgID, repo.RepoID, issue.GetNodeID()); existing != nil {
				if existing.UpdatedAt == issue.GetUpdatedAt() {
					continue
				}
			}

			if err := ss.syncer.fetchIssueComments(ss.ctx, org, repo, issue.GetNumber(), startTime, func(comments []*github.IssueComment) error {
				for _, comment := range comments {
					ss.addUser(comment.User)
					storageIssueComments = append(storageIssueComments, gh.IssueCommentFromAPI(org.OrgID, repo.RepoID, issue.GetNodeID(), comment))
				}

				return nil
			}); err != nil {
				return err
			}

			ss.addUser(issue.User)
			for _, assignee := range issue.Assignees {
				ss.addUser(assignee)
			}

			storageIssues = append(storageIssues, gh.IssueFromAPI(org.OrgID, repo.RepoID, issue))
		}

		err := ss.syncer.store.WriteIssues(ss.ctx, storageIssues)
		if err == nil {
			err = ss.syncer.store.WriteIssueComments(ss.ctx, storageIssueComments)
		}

		return err
	})
}

func (ss *syncState) handleZenHub(org *storage.Org, repo *storage.Repo) error {
	scope.Debugf("Getting ZenHub issue data for repo %s/%s", org.Login, repo.Name)

	// get all the issues
	var issues []*storage.Issue
	if err := ss.syncer.store.QueryIssuesByRepo(ss.ctx, org.OrgID, repo.RepoID, func(issue *storage.Issue) error {
		issues = append(issues, issue)
		return nil
	}); err != nil {
		return fmt.Errorf("unable to read issues from repo %s/%s: %v", org.Login, repo.Name, err)
	}

	// now get the ZenHub data for all issues
	var pipelines []*storage.IssuePipeline
	for _, issue := range issues {
		pipeline, err := ss.syncer.zht.Get(ss.ctx).GetIssueData(int(repo.RepoNumber), int(issue.Number))
		if err != nil {
			if err == zh.ErrNotFound {
				// not found, so nothing to do...
				return nil
			}

			return fmt.Errorf("unable to get issue data from zenhub for issue %d in repo %s/%s: %v", issue.Number, org.Login, repo.Name, err)
		}

		pipelines = append(pipelines, &storage.IssuePipeline{
			OrgID:       org.OrgID,
			RepoID:      repo.RepoID,
			IssueNumber: issue.Number,
			Pipeline:    pipeline.Pipeline.Name,
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

func (ss *syncState) handlePullRequests(org *storage.Org, repo *storage.Repo) error {
	scope.Debugf("Getting pull requests from repo %s/%s", org.Login, repo.Name)

	return ss.syncer.fetchPullRequests(ss.ctx, org, repo, func(prs []*github.PullRequest) error {
		var storagePRs []*storage.PullRequest
		var storagePRReviews []*storage.PullRequestReview
		var storagePRComments []*storage.PullRequestComment

		for _, pr := range prs {
			// if this pr is already known to us and is up to date, skip further processing
			if existing, _ := ss.syncer.cache.ReadPullRequest(ss.ctx, org.OrgID, repo.RepoID, pr.GetNodeID()); existing != nil {
				if existing.UpdatedAt == pr.GetUpdatedAt() {
					continue
				}
			}

			if err := ss.syncer.fetchIssueComments(ss.ctx, org, repo, pr.GetNumber(), time.Time{}, func(comments []*github.IssueComment) error {
				for _, comment := range comments {
					ss.addUser(comment.User)
					storagePRComments = append(storagePRComments, gh.PullRequestCommentFromAPI(org.OrgID, repo.RepoID, pr.GetNodeID(), comment))
				}

				return nil
			}); err != nil {
				return err
			}

			if err := ss.syncer.fetchReviews(ss.ctx, org, repo, pr.GetNumber(), func(reviews []*github.PullRequestReview) error {
				for _, review := range reviews {
					ss.addUser(review.GetUser())
					storagePRReviews = append(storagePRReviews, gh.PullRequestReviewFromAPI(org.OrgID, repo.RepoID, pr.GetNodeID(), review))
				}

				return nil
			}); err != nil {
				return err
			}

			var prFiles []string
			if err := ss.syncer.fetchFiles(ss.ctx, org, repo, pr.GetNumber(), func(files []string) error {
				prFiles = append(prFiles, files...)
				return nil
			}); err != nil {
				return err
			}

			ss.addUser(pr.GetUser())
			for _, reviewer := range pr.RequestedReviewers {
				ss.addUser(reviewer)
			}
			storagePRs = append(storagePRs, gh.PullRequestFromAPI(org.OrgID, repo.RepoID, pr, prFiles))
		}

		err := ss.syncer.store.WritePullRequests(ss.ctx, storagePRs)
		if err == nil {
			err = ss.syncer.store.WritePullRequestReviews(ss.ctx, storagePRReviews)
			if err == nil {
				err = ss.syncer.store.WritePullRequestComments(ss.ctx, storagePRComments)
			}
		}

		return err
	})
}

func (ss *syncState) handleMaintainers(org *storage.Org, repos []*storage.Repo) error {
	scope.Debugf("Getting maintainers for org %s", org.Login)

	maintainers := make(map[string]*storage.Maintainer)

	for _, repo := range repos {
		fc, _, _, err := ss.syncer.ght.Get(ss.ctx).Repositories.GetContents(ss.ctx, org.Login, repo.Name, "CODEOWNERS", nil)
		if err == nil {
			err = ss.handleCODEOWNERS(org, repo, maintainers, fc)
		} else {
			err = ss.handleOWNERS(org, repo, maintainers)
		}

		if err != nil {
			scope.Warnf("Unable to establish maintainers for repo %s/%s: %v", org.Login, repo.Name, err)
		}
	}

	storageMaintainers := make([]*storage.Maintainer, 0, len(maintainers))
	for _, maintainer := range maintainers {
		storageMaintainers = append(storageMaintainers, maintainer)
	}

	return ss.syncer.store.WriteAllMaintainers(ss.ctx, storageMaintainers)
}

func (ss *syncState) handleCODEOWNERS(org *storage.Org, repo *storage.Repo, maintainers map[string]*storage.Maintainer, fc *github.RepositoryContent) error {
	content, err := fc.GetContent()
	if err != nil {
		return fmt.Errorf("unable to read CODEOWNERS body from repo %s/%s: %v", org.Login, repo.Name, err)
	}

	lines := strings.Split(content, "\n")

	scope.Debugf("%d lines in CODEOWNERS file for repo %s/%s", len(lines), org.Login, repo.Name)

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

			// add the path to this maintainer's list
			path := strings.TrimPrefix(fields[0], "/")
			path = strings.TrimSuffix(path, "/*")
			if path == "*" {
				path = ""
			}

			scope.Debugf("User '%s' can review path '%s/%s/%s'", login, org.Login, repo.Name, path)

			maintainer, err := ss.getMaintainer(org, maintainers, login)
			if maintainer == nil || err != nil {
				scope.Warnf("Couldn't get info on potential maintainer %s: %v", login, err)
				continue
			}

			maintainer.Paths = append(maintainer.Paths, repo.RepoID+"/"+path)
		}
	}

	return nil
}

type ownersFile struct {
	Approvers []string `json:"approvers"`
	Reviewers []string `json:"reviewers"`
}

func (ss *syncState) handleOWNERS(org *storage.Org, repo *storage.Repo, maintainers map[string]*storage.Maintainer) error {
	opt := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	// TODO: we need to get the SHA for the latest commit on the master branch, not just any branch
	rc, _, err := ss.syncer.ght.Get(ss.ctx).Repositories.ListCommits(ss.ctx, org.Login, repo.Name, opt)
	if err != nil {
		return fmt.Errorf("unable to get latest commit in repo %s/%s: %v", org.Login, repo.Name, err)
	}

	tree, _, err := ss.syncer.ght.Get(ss.ctx).Git.GetTree(ss.ctx, org.Login, repo.Name, rc[0].GetSHA(), true)
	if err != nil {
		return fmt.Errorf("unable to get tree in repo %s/%s: %v", org.Login, repo.Name, err)
	}

	files := make(map[string]ownersFile)
	for _, entry := range tree.Entries {
		components := strings.Split(entry.GetPath(), "/")
		if components[len(components)-1] == "OWNERS" && components[0] != "vendor" { // HACK: skip Go's vendor directory

			url := "https://raw.githubusercontent.com/" + org.Login + "/" + repo.Name + "/master/" + entry.GetPath()

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

	scope.Debugf("%d OWNERS files found in repo %s/%s", len(files), org.Login, repo.Name)

	for path, file := range files {
		for _, user := range file.Approvers {
			maintainer, err := ss.getMaintainer(org, maintainers, user)
			if maintainer == nil || err != nil {
				scope.Warnf("Couldn't get info on potential maintainer %s: %v", user, err)
				continue
			}

			p := strings.TrimSuffix(path, "OWNERS")

			scope.Debugf("User '%s' can approve path %s/%s/%s", user, org.Login, repo.Name, p)

			maintainer.Paths = append(maintainer.Paths, repo.RepoID+"/"+p)
		}
	}

	return nil
}

func (ss *syncState) addUser(user *github.User) {
	ss.users[user.GetLogin()] = gh.UserFromAPI(user)
}

func (ss *syncState) getMaintainer(org *storage.Org, maintainers map[string]*storage.Maintainer, login string) (*storage.Maintainer, error) {
	user, ok := ss.users[login]
	if !ok {
		var err error
		user, err = ss.syncer.cache.ReadUserByLogin(ss.ctx, login)
		if err != nil {
			return nil, fmt.Errorf("unable to read information from storage for user %s: %v", login, err)
		}
	}

	if user == nil {
		// couldn't find user info, ask GitHub directly
		u, _, err := ss.syncer.ght.Get(ss.ctx).Users.Get(ss.ctx, login)
		if err != nil {
			return nil, fmt.Errorf("unable to read information from GitHub on user %s: %v", login, err)
		}

		user = gh.UserFromAPI(u)
		ss.users[user.Login] = user
	}

	maintainer, ok := maintainers[user.UserID]
	if !ok {
		// unknown maintainer, so create a record
		maintainer = &storage.Maintainer{
			OrgID:  org.OrgID,
			UserID: user.UserID,
		}
		maintainers[user.UserID] = maintainer
	}

	return maintainer, nil
}
