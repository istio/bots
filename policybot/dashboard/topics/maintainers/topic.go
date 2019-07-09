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

//go:generate ../../../scripts/gen_topic.sh

package maintainers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"istio.io/bots/policybot/dashboard"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/util"
	rawcache "istio.io/pkg/cache"
	"istio.io/pkg/log"
)

type topic struct {
	store           storage.Store
	cache           *cache.Cache
	maintainerInfos rawcache.ExpiringCache
	context         dashboard.RenderContext
	options         *dashboard.Options
	single          *template.Template
}

type prAction struct {
	Path string    `json:"path"`
	When time.Time `json:"when"`
}

type repoInfo struct {
	Name                   string     `json:"name"`
	LastPullRequestActions []prAction `json:"last_pull_request_actions"`
	LastIssueCommented     time.Time  `json:"last_issue_commented"`
	LastIssueClosed        time.Time  `json:"last_issue_closed"`
	LastIssueTriaged       time.Time  `json:"last_issue_triaged"`
}

type maintainerInfo struct {
	Login     string     `json:"login"`
	Name      string     `json:"name"`
	Company   string     `json:"company"`
	AvatarURL string     `json:"avatar_url"`
	Emeritus  bool       `json:"emeritus"`
	RepoInfo  []repoInfo `json:"repo_info"`
	LastSeen  string     `json:"last_seen"`
}

func NewTopic(store storage.Store, cache *cache.Cache, cacheTTL time.Duration) dashboard.Topic {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if cacheTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = cacheTTL / 2
	}

	return &topic{
		store:           store,
		cache:           cache,
		maintainerInfos: rawcache.NewTTL(cacheTTL, evictionInterval),
		single:          template.Must(template.New("single").Parse(string(MustAsset("single.html")))),
	}
}

func (t *topic) Title() string {
	return "Org Maintainers"
}

func (t *topic) Description() string {
	return "Learn about folks that maintain Istio."
}

func (t *topic) Name() string {
	return "maintainers"
}

func (t *topic) Subtopics() []dashboard.Topic {
	return nil
}

func (t *topic) Configure(htmlRouter *mux.Router, apiRouter *mux.Router, context dashboard.RenderContext, opt *dashboard.Options) {
	t.context = context
	t.options = opt

	htmlRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(t.handleMaintainersListHTML)

	htmlRouter.StrictSlash(true).
		Path("/{login}").
		Methods("GET").
		HandlerFunc(t.handleSingleMaintainerHTML)

	apiRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(t.handleMaintainerListJSON)
}

func (t *topic) handleSingleMaintainerHTML(w http.ResponseWriter, r *http.Request) {
	orgLogin := r.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = t.options.DefaultOrg
	}

	userLogin := mux.Vars(r)["login"]

	info, err := t.getSingleMaintainerInfo(r.Context(), orgLogin, userLogin)
	if err != nil {
		t.context.RenderHTMLError(w, err)
		return
	}

	sb := &strings.Builder{}
	if err := t.single.Execute(sb, info); err != nil {
		t.context.RenderHTMLError(w, err)
		return
	}

	t.context.RenderHTML(w, sb.String())
}

func (t *topic) handleMaintainersListHTML(w http.ResponseWriter, r *http.Request) {
	orgLogin := r.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = t.options.DefaultOrg
	}

	org, err := t.cache.ReadOrg(r.Context(), orgLogin)
	if err != nil {
		t.context.RenderHTMLError(w, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", orgLogin, err))
		return
	} else if org == nil {
		t.context.RenderHTMLError(w, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin))
		return
	}

	t.context.RenderHTML(w, string(MustAsset("all.html")))
}

func (t *topic) handleMaintainerListJSON(w http.ResponseWriter, r *http.Request) {
	orgLogin := r.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = t.options.DefaultOrg
	}

	// turn the connection into a web socket
	var upgrader websocket.Upgrader
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		util.RenderError(w, util.HTTPErrorf(http.StatusInternalServerError, "%v", err))
		return
	}
	defer c.Close()

	if err = t.store.QueryMaintainersByOrg(r.Context(), orgLogin, func(m *storage.Maintainer) error {
		info, err := t.getMaintainerInfo(r.Context(), m)
		if err != nil {
			return err
		}

		return c.WriteJSON(info)
	}); err != nil {
		log.Errorf("Returning error on websocket: %v", err)
		_ = c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%v", err)))
	}
}

func (t *topic) getSingleMaintainerInfo(context context.Context, orgLogin string, userLogin string) (*maintainerInfo, error) {
	maintainer, err := t.cache.ReadMaintainer(context, orgLogin, userLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on maintainer %s: %v", userLogin, err)
	} else if maintainer == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on maintainer %s", userLogin)
	}

	info, err := t.getMaintainerInfo(context, maintainer)
	if err != nil {
		return nil, err
	}

	return info, err
}

func (t *topic) getMaintainerInfo(context context.Context, maintainer *storage.Maintainer) (*maintainerInfo, error) {
	if result, ok := t.maintainerInfos.Get(maintainer.OrgLogin + maintainer.UserLogin); ok {
		return result.(*maintainerInfo), nil
	}

	org, err := t.cache.ReadOrg(context, maintainer.OrgLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", maintainer.OrgLogin, err)
	} else if org == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", maintainer.OrgLogin)
	}

	user, err := t.cache.ReadUser(context, maintainer.UserLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to read from storage: %v", err)
	}

	info, err := t.store.QueryMaintainerInfo(context, maintainer)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information about maintainer %s: %v", maintainer.UserLogin, err)
	}

	var ri []repoInfo
	for _, repo := range info.Repos {
		r, err := t.cache.ReadRepo(context, org.OrgLogin, repo.RepoName)
		if err != nil {
			return nil, fmt.Errorf("unable to get repo information from storage: %v", err)
		}

		var pra []prAction
		for path, entry := range repo.LastPullRequestCommittedByPath {
			pra = append(pra, prAction{
				Path: path,
				When: entry.Time,
			})
		}

		ri = append(ri, repoInfo{
			Name:                   r.RepoName,
			LastIssueCommented:     repo.LastIssueCommented.Time,
			LastPullRequestActions: pra,
		})
	}

	name := user.Name
	if name == "" {
		name = user.UserLogin
	}

	mi := &maintainerInfo{
		Login:     user.UserLogin,
		Name:      name,
		Company:   user.Company,
		AvatarURL: user.AvatarURL,
		Emeritus:  false, // TODO: get real value
		RepoInfo:  ri,
		LastSeen:  "03/12/2018", // TODO: get real value
	}

	t.maintainerInfos.Set(org.OrgLogin+user.UserLogin, mi)
	return mi, nil
}
