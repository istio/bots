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
	"github.com/google/go-github/v25/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/util"
	"istio.io/pkg/log"
)

// Syncer is responsible for synchronizing issues and pull request from GitHub to our local store
type Syncer struct {
	ctx   context.Context
	ghs   *gh.GitHubState
	ght   *util.GitHubThrottle
	store storage.Store
	orgs  []config.Org
}

type filterFlags int

// the things to sync
const (
	issues      filterFlags = 1 << 0
	prs                     = 1 << 1
	maintainers             = 1 << 2
	members                 = 1 << 3
	labels                  = 1 << 4
	zenhub                  = 1 << 5
)

// The state in Syncer is immutable once created. syncState on the other hand represents
// the mutable state used during a single sync operation.
type syncState struct {
	syncer *Syncer
	users  map[string]*storage.User
	flags  filterFlags
}

var scope = log.RegisterScope("syncer", "The GitHub data syncer", 0)

func NewHandler(ctx context.Context, ght *util.GitHubThrottle, ghs *gh.GitHubState, store storage.Store, orgs []config.Org) http.Handler {
	return &Syncer{
		ctx:   ctx,
		ght:   ght,
		ghs:   ghs,
		store: store,
		orgs:  orgs,
	}
}

func (s *Syncer) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	if err := s.Sync(r.URL.Query().Get("filter")); err != nil {
		scope.Errorf(err.Error())
	}
}

func convFilters(filter string) (filterFlags, error) {
	if filter == "" {
		// defaults to everything
		return issues | prs | maintainers | members | labels | zenhub, nil
	}

	var result filterFlags
	for _, f := range strings.Split(filter, ",") {
		switch f {
		case "issues":
			result |= issues
		case "prs":
			result |= prs
		case "maintainers":
			result |= maintainers
		case "members":
			result |= members
		case "labels":
			result |= labels
		case "zenhub":
			result |= zenhub
		default:
			return 0, fmt.Errorf("unknown filter value %s", f)
		}
	}

	return result, nil
}

func (s *Syncer) Sync(filter string) error {
	flags, err := convFilters(filter)
	if err != nil {
		return err
	}

	ss := &syncState{
		syncer: s,
		users:  make(map[string]*storage.User),
		flags:  flags,
	}

	var orgs []*storage.Org
	var repos []*storage.Repo

	// get all the org & repo info
	if err := s.fetchOrgs(func(org *github.Organization) error {
		orgs = append(orgs, gh.OrgFromAPI(org))
		return s.fetchRepos(func(repo *github.Repository) error {
			repos = append(repos, gh.RepoFromAPI(repo))
			return nil
		})
	}); err != nil {
		return err
	}

	if err := s.store.WriteOrgs(orgs); err != nil {
		return err
	}

	if err := s.store.WriteRepos(repos); err != nil {
		return err
	}

	if flags&(members|labels|issues|prs) != 0 {
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

	if flags&maintainers != 0 {
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

	if err := ss.syncer.store.WriteUsers(users); err != nil {
		return err
	}

	return nil
}

func (ss *syncState) handleOrg(org *storage.Org, repos []*storage.Repo) error {
	scope.Infof("Syncing org %s", org.Login)

	if ss.flags&members != 0 {
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

	if ss.flags&labels != 0 {
		if err := ss.handleLabels(org, repo); err != nil {
			return err
		}
	}

	if ss.flags&issues != 0 {
		start := time.Now().UTC()
		priorStart := time.Time{}
		if activity, _ := ss.syncer.store.ReadBotActivityByID(org.OrgID, repo.RepoID); activity != nil {
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

		if err := ss.syncer.store.WriteBotActivities([]*storage.BotActivity{&activity}); err != nil {
			scope.Warnf("unable to update bot activity for repo %s/%s: %v", org.Login, repo.Name, err)
		}
	}

	if ss.flags&prs != 0 {
		if err := ss.handlePullRequests(org, repo); err != nil {
			return err
		}
	}

	return nil
}

func (ss *syncState) handleMembers(org *storage.Org) error {
	scope.Debugf("Getting members from org %s", org.Login)

	var storageMembers []*storage.Member
	if err := ss.syncer.fetchMembers(org, func(members []*github.User) error {
		for _, member := range members {
			ss.addUser(member)
			storageMembers = append(storageMembers, &storage.Member{OrgID: org.OrgID, UserID: member.GetNodeID()})
		}

		return nil
	}); err != nil {
		return err
	}

	return ss.syncer.store.WriteAllMembers(storageMembers)
}

func (ss *syncState) handleLabels(org *storage.Org, repo *storage.Repo) error {
	scope.Debugf("Getting labels from repo %s/%s", org.Login, repo.Name)

	return ss.syncer.fetchLabels(org, repo, func(labels []*github.Label) error {
		storageLabels := make([]*storage.Label, 0, len(labels))
		for _, label := range labels {
			storageLabels = append(storageLabels, gh.LabelFromAPI(org.OrgID, repo.RepoID, label))
		}

		return ss.syncer.store.WriteLabels(storageLabels)
	})
}

func (ss *syncState) handleIssues(org *storage.Org, repo *storage.Repo, startTime time.Time) error {
	scope.Debugf("Getting issues from repo %s/%s", org.Login, repo.Name)

	return ss.syncer.fetchIssues(org, repo, startTime, func(issues []*github.Issue) error {
		var storageIssues []*storage.Issue
		var storageIssueComments []*storage.IssueComment

		for _, issue := range issues {
			// if this issue is already known to us and is up to date, skip further processing
			if existing, _ := ss.syncer.ghs.ReadIssue(org.OrgID, repo.RepoID, issue.GetNodeID()); existing != nil {
				if existing.UpdatedAt == issue.GetUpdatedAt() {
					continue
				}
			}

			if err := ss.syncer.fetchComments(org, repo, issue.GetNumber(), startTime, func(comments []*github.IssueComment) error {
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

		err := ss.syncer.store.WriteIssues(storageIssues)
		if err == nil {
			err = ss.syncer.store.WriteIssueComments(storageIssueComments)
		}

		return err
	})
}

func (ss *syncState) handlePullRequests(org *storage.Org, repo *storage.Repo) error {
	scope.Debugf("Getting pull requests from repo %s/%s", org.Login, repo.Name)

	return ss.syncer.fetchPullRequests(org, repo, func(prs []*github.PullRequest) error {
		var storagePRs []*storage.PullRequest
		var storagePRReviews []*storage.PullRequestReview

		for _, pr := range prs {
			// if this pr is already known to us and is up to date, skip further processing
			if existing, _ := ss.syncer.ghs.ReadPullRequest(org.OrgID, repo.RepoID, pr.GetNodeID()); existing != nil {
				if existing.UpdatedAt == pr.GetUpdatedAt() {
					continue
				}
			}

			if err := ss.syncer.fetchReviews(org, repo, pr.GetNumber(), func(reviews []*github.PullRequestReview) error {
				for _, review := range reviews {
					ss.addUser(review.GetUser())
					storagePRReviews = append(storagePRReviews, gh.PullRequestReviewFromAPI(org.OrgID, repo.RepoID, pr.GetNodeID(), review))
				}

				return nil
			}); err != nil {
				return err
			}

			var prFiles []string
			if err := ss.syncer.fetchFiles(org, repo, pr.GetNumber(), func(files []string) error {
				for _, file := range files {
					prFiles = append(prFiles, org.Login+"/"+repo.Name+"/"+file)
				}

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

		err := ss.syncer.store.WritePullRequests(storagePRs)
		if err == nil {
			err = ss.syncer.store.WritePullRequestReviews(storagePRReviews)
		}

		return err
	})
}

func (ss *syncState) handleMaintainers(org *storage.Org, repos []*storage.Repo) error {
	scope.Debugf("Getting maintainers for org %s", org.Login)

	maintainers := make(map[string]*storage.Maintainer)

	for _, repo := range repos {
		fc, _, _, err := ss.syncer.ght.Get().Repositories.GetContents(ss.syncer.ctx, org.Login, repo.Name, "CODEOWNERS", nil)
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

	return ss.syncer.store.WriteAllMaintainers(storageMaintainers)
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

			scope.Debugf("User '%s' can review path '%s'", login, fields[0])

			maintainer, err := ss.getMaintainer(org, maintainers, login)
			if maintainer == nil || err != nil {
				scope.Warnf("Couldn't get info on potential maintainer %s: %v", login, err)
				continue
			}

			// add the path to this maintainer's list
			path := repo.Name
			if !strings.HasPrefix(path, "/") {
				path += "/"
			}
			path += fields[0]

			maintainer.Paths = append(maintainer.Paths, path)
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
	rc, _, err := ss.syncer.ght.Get().Repositories.ListCommits(ss.syncer.ctx, org.Login, repo.Name, opt)
	if err != nil {
		return fmt.Errorf("unable to get latest commit in repo %s/%s: %v", org.Login, repo.Name, err)
	}

	tree, _, err := ss.syncer.ght.Get().Git.GetTree(ss.syncer.ctx, org.Login, repo.Name, rc[0].GetSHA(), true)
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
			if p == "" {
				p = "*"
			}

			scope.Debugf("User %s can approve path %s", user, p)

			maintainer.Paths = append(maintainer.Paths, p)
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
		user, err = ss.syncer.ghs.ReadUserByLogin(login)
		if err != nil {
			return nil, fmt.Errorf("unable to read information from storage for user %s: %v", login, err)
		}
	}

	if user == nil {
		// couldn't find user info, ask GitHub directly
		u, _, err := ss.syncer.ght.Get().Users.Get(ss.syncer.ctx, login)
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
