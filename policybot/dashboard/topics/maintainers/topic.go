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
	"encoding/json"
	"fmt"
	"html/template"
	"math/bits"
	"net/http"
	"strings"
	"time"

	"istio.io/bots/policybot/dashboard/types"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/util"
	rawcache "istio.io/pkg/cache"
	"istio.io/pkg/log"
)

// Maintainers lets users view information about project maintainers.
type Maintainers struct {
	store          storage.Store
	cache          *cache.Cache
	combos         rawcache.ExpiringCache
	single         *template.Template
	user           *template.Template
	singleControl  *template.Template
	list           *template.Template
	activityWindow time.Duration
	defaultOrg     string
}

type combo struct {
	User           *storage.User
	Maintainer     *storage.Maintainer
	MaintainerInfo *storage.ActivityInfo
	TimeZero       time.Time // hack to provide a zero-initialized timestamp to the Go templates
}

type filterFlags int

// what this page can display
const (
	recentlyActive   filterFlags = 1 << 0
	recentlyInactive             = 1 << 1
	emeritus                     = 1 << 2
)

// New creates a new Maintainers instance
func New(store storage.Store, cache *cache.Cache, cacheTTL time.Duration, activityWindow time.Duration, defaultOrg string) *Maintainers {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if cacheTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = cacheTTL / 2
	}

	return &Maintainers{
		store:          store,
		cache:          cache,
		combos:         rawcache.NewTTL(cacheTTL, evictionInterval),
		single:         template.Must(template.New("single").Parse(string(MustAsset("single.html")))),
		user:           template.Must(template.New("user").Parse(string(MustAsset("user.html")))),
		singleControl:  template.Must(template.New("singleControl").Parse(string(MustAsset("single_control.html")))),
		list:           template.Must(template.New("list").Parse(string(MustAsset("list.html")))),
		activityWindow: activityWindow,
		defaultOrg:     defaultOrg,
	}
}

// Renders the HTML for a single maintainer.
func (m *Maintainers) RenderSingle(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = m.defaultOrg
	}

	userLogin := mux.Vars(req)["login"]

	g, err := m.getSingleMaintainerInfo(req.Context(), orgLogin, userLogin)
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

// Renders the HTML for the list of maintainers.
func (m *Maintainers) RenderList(req *http.Request) (types.RenderInfo, error) {
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
		if bits.OnesCount(uint(filter)) > 1 {
			title = "Filtered Maintainers"
		} else if filter&recentlyActive != 0 {
			title = "Recently Active Maintainers"
		} else if filter&recentlyInactive != 0 {
			title = "Recently Inactive Maintainers"
			info.Mode = "inactive"
		} else if filter&emeritus != 0 {
			title = "Emeritus Maintainers"
			info.Mode = "emeritus"
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

// Returns the list of maintainers via WebSocket.
func (m *Maintainers) GetList(w http.ResponseWriter, req *http.Request) {
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

	if err = m.store.QueryMaintainersByOrg(req.Context(), orgLogin, func(maintainer *storage.Maintainer) error {
		combo, err := m.getCombo(req.Context(), maintainer, true)
		if err != nil {
			return err
		}

		if combo == nil {
			// no info found for this maintainer, skip it
			return nil
		}

		use := false
		cutoff := time.Now().Add(-m.activityWindow)
		if filter&recentlyActive != 0 {
			if combo.MaintainerInfo.LastActivity.After(cutoff) {
				use = true
			}
		}

		if filter&recentlyInactive != 0 {
			if combo.MaintainerInfo.LastActivity.Before(cutoff) {
				use = true
			}
		}

		if filter&emeritus != 0 {
			if combo.Maintainer.Emeritus {
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
		// defaults to current maintainers
		return recentlyActive | recentlyInactive, nil
	}

	var result filterFlags
	for _, f := range strings.Split(filter, ",") {
		switch f {
		case "active":
			result |= recentlyActive
		case "inactive":
			result |= recentlyInactive
		case "emeritus":
			result |= emeritus
		default:
			return 0, fmt.Errorf("unknown filter flag %s", f)
		}
	}

	return result, nil
}

func (m *Maintainers) getSingleMaintainerInfo(context context.Context, orgLogin string, userLogin string) (*combo, error) {
	maintainer, err := m.cache.ReadMaintainer(context, orgLogin, userLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on maintainer %s: %v", userLogin, err)
	} else if maintainer == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on maintainer %s", userLogin)
	}

	combo, err := m.getCombo(context, maintainer, false)
	if err != nil {
		return nil, err
	}

	return combo, err
}

func (m *Maintainers) getCombo(context context.Context, maintainer *storage.Maintainer, skipUnknowns bool) (*combo, error) {
	if result, ok := m.combos.Get(maintainer.OrgLogin + maintainer.UserLogin); ok {
		return result.(*combo), nil
	}

	org, err := m.cache.ReadOrg(context, maintainer.OrgLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", maintainer.OrgLogin, err)
	} else if org == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", maintainer.OrgLogin)
	}

	user, err := m.cache.ReadUser(context, maintainer.UserLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to read from storage: %v", err)
	} else if user == nil {
		if skipUnknowns {
			return nil, nil
		}
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on maintainer %s", maintainer.UserLogin)
	}

	var info *storage.ActivityInfo
	if maintainer.CachedInfo == "" {
		info, err = m.store.QueryMaintainerActivity(context, maintainer)
		if err != nil {
			return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information about maintainer %s: %v", maintainer.UserLogin, err)
		}
	} else {
		var o storage.ActivityInfo
		err = json.Unmarshal([]byte(maintainer.CachedInfo), &o)
		if err != nil {
			return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to decode contribution info about maintainer %s: %v", maintainer.UserLogin, err)
		}
		info = &o
	}

	combo := &combo{
		User:           user,
		Maintainer:     maintainer,
		MaintainerInfo: info,
	}

	m.combos.Set(org.OrgLogin+user.UserLogin, combo)
	return combo, nil
}
