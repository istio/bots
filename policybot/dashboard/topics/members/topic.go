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

package members

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"istio.io/bots/policybot/pkg/config"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"istio.io/bots/policybot/dashboard/types"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/util"
	rawcache "istio.io/pkg/cache"
	"istio.io/pkg/log"
)

// Members lets users view information about organization members.
type Members struct {
	store          storage.Store
	cache          *cache.Cache
	combos         rawcache.ExpiringCache
	single         *template.Template
	user           *template.Template
	singleControl  *template.Template
	list           *template.Template
	activityWindow time.Duration
	defaultOrg     string
	orgs           []config.Org
}

type combo struct {
	User       *storage.User
	Member     *storage.Member
	MemberInfo *storage.ActivityInfo
	TimeZero   time.Time // hack to provide a zero-initialized timestamp to the Go templates
}

type filterFlags int

// what this page can display
const (
	recentlyActive   filterFlags = 1 << 0
	recentlyInactive             = 1 << 1
)

// New creates a new Members instance
func New(store storage.Store, cache *cache.Cache, cacheTTL time.Duration, activityWindow time.Duration, defaultOrg string, orgs []config.Org) *Members {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if cacheTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = cacheTTL / 2
	}

	return &Members{
		store:          store,
		cache:          cache,
		combos:         rawcache.NewTTL(cacheTTL, evictionInterval),
		single:         template.Must(template.New("single").Parse(string(MustAsset("single.html")))),
		user:           template.Must(template.New("user").Parse(string(MustAsset("user.html")))),
		singleControl:  template.Must(template.New("singleControl").Parse(string(MustAsset("single_control.html")))),
		list:           template.Must(template.New("list").Parse(string(MustAsset("list.html")))),
		activityWindow: activityWindow,
		defaultOrg:     defaultOrg,
		orgs:           orgs,
	}
}

// Renders the HTML for a single member.
func (m *Members) RenderSingle(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = m.defaultOrg
	}

	userLogin := mux.Vars(req)["login"]

	g, err := m.getSingleMemberInfo(req.Context(), orgLogin, userLogin)
	if err != nil {
		return types.RenderInfo{}, err
	}

	var content strings.Builder
	if err := m.single.Execute(&content, g); err != nil {
		return types.RenderInfo{}, err
	}

	var control strings.Builder
	if err := m.singleControl.Execute(&control, g); err != nil {
		return types.RenderInfo{}, err
	}

	title := g.User.Name
	if title == "" {
		title = g.User.UserLogin
	}

	return types.RenderInfo{
		Title:   title,
		Content: content.String(),
		Control: control.String(),
	}, nil
}

// Renders the HTML for the list of members.
func (m *Members) RenderList(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = m.defaultOrg
	}

	filter, err := convFilterFlags(req.URL.Query().Get("filter"))
	if err != nil {
		return types.RenderInfo{}, util.HTTPErrorf(http.StatusBadRequest, "invalid filter expression %s specified", req.URL.Query().Get("filter"))
	}

	org, err := m.cache.ReadOrg(req.Context(), orgLogin)
	if err != nil {
		return types.RenderInfo{}, err
	} else if org == nil {
		return types.RenderInfo{}, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin)
	}

	info := struct {
		Mode         string
		ActivityDays int
	}{
		Mode:         "normal",
		ActivityDays: int(m.activityWindow / (time.Hour * 24)),
	}

	title := ""
	if filter != 0 && filter != recentlyActive|recentlyInactive {
		if filter&recentlyActive != 0 {
			title = "Recently Active Members"
		} else if filter&recentlyInactive != 0 {
			title = "Recently Inactive Members"
			info.Mode = "inactive"
		}
	}

	var sb strings.Builder
	if err := m.list.Execute(&sb, info); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Title:   title,
		Content: sb.String(),
	}, nil
}

// Returns the list of members via WebSocket.
func (m *Members) GetList(w http.ResponseWriter, req *http.Request) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = m.defaultOrg
	}

	filter, err := convFilterFlags(req.URL.Query().Get("filter"))
	if err != nil {
		util.RenderError(w, util.HTTPErrorf(http.StatusBadRequest, "%v", err))
		return
	}

	// turn the connection into a web socket
	var upgrader websocket.Upgrader
	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		util.RenderError(w, util.HTTPErrorf(http.StatusInternalServerError, "%v", err))
		return
	}
	defer c.Close()

	if err = m.store.QueryMembersByOrg(req.Context(), orgLogin, func(member *storage.Member) error {
		combo, err := m.getCombo(req.Context(), member, true)
		if err != nil {
			return err
		}

		if combo == nil {
			// no info found for this member, skip it
			return nil
		}

		use := false
		cutoff := time.Now().Add(-m.activityWindow)
		if filter&recentlyActive != 0 {
			if combo.MemberInfo.LastActivity.After(cutoff) {
				use = true
			}
		}

		if filter&recentlyInactive != 0 {
			if combo.MemberInfo.LastActivity.Before(cutoff) {
				use = true
			}
		}

		if !use {
			return nil
		}

		var sb strings.Builder
		if err := m.user.Execute(&sb, combo); err != nil {
			return err
		}

		return c.WriteMessage(websocket.TextMessage, []byte(sb.String()))
	}); err != nil {
		log.Errorf("Returning error on websocket: %v", err)
		_ = c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%v", err)))
	}
}

func convFilterFlags(filter string) (filterFlags, error) {
	if filter == "" {
		// defaults to all members
		return recentlyActive | recentlyInactive, nil
	}

	var result filterFlags
	for _, f := range strings.Split(filter, ",") {
		switch f {
		case "active":
			result |= recentlyActive
		case "inactive":
			result |= recentlyInactive
		default:
			return 0, fmt.Errorf("unknown filter flag %s", f)
		}
	}

	return result, nil
}

func (m *Members) getSingleMemberInfo(context context.Context, orgLogin string, userLogin string) (*combo, error) {
	member, err := m.cache.ReadMember(context, orgLogin, userLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on member %s: %v", userLogin, err)
	} else if member == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on member %s", userLogin)
	}

	combo, err := m.getCombo(context, member, false)
	if err != nil {
		return nil, err
	}

	return combo, err
}

func (m *Members) getCombo(context context.Context, member *storage.Member, skipUnknowns bool) (*combo, error) {
	if result, ok := m.combos.Get(member.OrgLogin + member.UserLogin); ok {
		return result.(*combo), nil
	}

	org, err := m.cache.ReadOrg(context, member.OrgLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", member.OrgLogin, err)
	} else if org == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", member.OrgLogin)
	}

	user, err := m.cache.ReadUser(context, member.UserLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to read from storage: %v", err)
	} else if user == nil {
		if skipUnknowns {
			return nil, nil
		}
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on member %s", member.UserLogin)
	}

	var info *storage.ActivityInfo
	if member.CachedInfo == "" {
		var repoNames []string
		for _, configOrg := range m.orgs {
			if configOrg.Name == org.OrgLogin {
				for _, configRepo := range configOrg.Repos {
					repoNames = append(repoNames, configRepo.Name)
				}
			}
		}

		info, err = m.store.QueryMemberActivity(context, member, repoNames)
		if err != nil {
			return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information about member %s: %v", member.UserLogin, err)
		}
	} else {
		var o storage.ActivityInfo
		err = json.Unmarshal([]byte(member.CachedInfo), &o)
		if err != nil {
			return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to decode contribution info about member %s: %v", member.UserLogin, err)
		}
		info = &o
	}

	combo := &combo{
		User:       user,
		Member:     member,
		MemberInfo: info,
	}

	m.combos.Set(org.OrgLogin+user.UserLogin, combo)
	return combo, nil
}
