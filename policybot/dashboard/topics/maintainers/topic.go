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
	store         storage.Store
	cache         *cache.Cache
	combos        rawcache.ExpiringCache
	context       dashboard.RenderContext
	options       *dashboard.Options
	single        *template.Template
	user          *template.Template
	singleControl *template.Template
}

type combo struct {
	User           *storage.User
	Maintainer     *storage.Maintainer
	MaintainerInfo *storage.MaintainerInfo
	TimeZero       time.Time // hack to provide a zero-initialized timestamp to the Go templates
}

const activityWindow = time.Hour * 24 * 90

type filterFlags int

// what this page can display
const (
	recentlyActive   filterFlags = 1 << 0
	recentlyInactive             = 1 << 1
	emeritus                     = 1 << 2
)

func NewTopic(store storage.Store, cache *cache.Cache, cacheTTL time.Duration) dashboard.Topic {
	// purge the cache every 10 seconds
	evictionInterval := 10 * time.Second
	if cacheTTL < 20*time.Second {
		// if the TTL is very low, provide a faster eviction interval
		evictionInterval = cacheTTL / 2
	}

	return &topic{
		store:         store,
		cache:         cache,
		combos:        rawcache.NewTTL(cacheTTL, evictionInterval),
		single:        template.Must(template.New("single").Parse(string(MustAsset("single.html")))),
		user:          template.Must(template.New("user").Parse(string(MustAsset("user.html")))),
		singleControl: template.Must(template.New("singleControl").Parse(string(MustAsset("single_control.html")))),
	}
}

func (t *topic) Title() string {
	return "Org Maintainers"
}

func (t *topic) Description() string {
	return "Learn about folks that maintain Istio."
}

func (t *topic) URLSuffix() string {
	return "/maintainers"
}

func (t *topic) Subtopics() []dashboard.Topic {
	return []dashboard.Topic{
		filteredTopic{"Recently Active", "Maintainers who have been recently active on the project", "?filter=active"},
		filteredTopic{"Recently Inactive", "Maintainers who have been recently inactive on the project", "?filter=inactive"},
		filteredTopic{"Emeritus", "Maintainers who are no longer involved with the project", "?filter=emeritus"},
	}
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

	g, err := t.getSingleMaintainerInfo(r.Context(), orgLogin, userLogin)
	if err != nil {
		t.context.RenderHTMLError(w, err)
		return
	}

	content := &strings.Builder{}
	if err := t.single.Execute(content, g); err != nil {
		t.context.RenderHTMLError(w, err)
		return
	}

	control := &strings.Builder{}
	if err := t.singleControl.Execute(control, g); err != nil {
		t.context.RenderHTMLError(w, err)
		return
	}

	name := g.User.Name
	if name == "" {
		name = g.User.UserLogin
	}

	t.context.RenderHTML(w, name, content.String(), control.String())
}

func (t *topic) handleMaintainersListHTML(w http.ResponseWriter, r *http.Request) {
	orgLogin := r.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = t.options.DefaultOrg
	}

	filter, err := convFilterFlags(r.URL.Query().Get("filter"))
	if err != nil {
		t.context.RenderHTMLError(w, util.HTTPErrorf(http.StatusBadRequest, "invalid filter expression %s specified", r.URL.Query().Get("filter")))
		return
	}

	org, err := t.cache.ReadOrg(r.Context(), orgLogin)
	if err != nil {
		t.context.RenderHTMLError(w, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", orgLogin, err))
		return
	} else if org == nil {
		t.context.RenderHTMLError(w, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin))
		return
	}

	if filter != 0 && filter != recentlyActive|recentlyInactive {
		title := ""
		if bits.OnesCount(uint(filter)) > 1 {
			title = "Filtered Maintainers"
		} else if filter&recentlyActive != 0 {
			title = "Recently Active Maintainers"
		} else if filter&recentlyInactive != 0 {
			title = "Recently Inactive Maintainers"
		} else if filter&emeritus != 0 {
			title = "Emeritus Maintainers"
		}

		t.context.RenderHTML(w, title, string(MustAsset("list.html")), "")
	} else {
		t.context.RenderHTML(w, "", string(MustAsset("list.html")), "")
	}
}

func (t *topic) handleMaintainerListJSON(w http.ResponseWriter, r *http.Request) {
	orgLogin := r.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = t.options.DefaultOrg
	}

	filter, err := convFilterFlags(r.URL.Query().Get("filter"))
	if err != nil {
		util.RenderError(w, util.HTTPErrorf(http.StatusBadRequest, "%v", err))
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
		combo, err := t.getCombo(r.Context(), m, true)
		if err != nil {
			return err
		}

		if combo == nil {
			// no info found for this maintainer, skip it
			return nil
		}

		use := false
		cutoff := time.Now().Add(-activityWindow)
		if filter&recentlyActive != 0 {
			if combo.MaintainerInfo.LastMaintenanceActivity.After(cutoff) {
				use = true
			}
		}

		if filter&recentlyInactive != 0 {
			if combo.MaintainerInfo.LastMaintenanceActivity.Before(cutoff) {
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

		sb := &strings.Builder{}
		if err := t.user.Execute(sb, combo); err != nil {
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

func (t *topic) getSingleMaintainerInfo(context context.Context, orgLogin string, userLogin string) (*combo, error) {
	maintainer, err := t.cache.ReadMaintainer(context, orgLogin, userLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on maintainer %s: %v", userLogin, err)
	} else if maintainer == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on maintainer %s", userLogin)
	}

	combo, err := t.getCombo(context, maintainer, false)
	if err != nil {
		return nil, err
	}

	return combo, err
}

func (t *topic) getCombo(context context.Context, maintainer *storage.Maintainer, skipUnknowns bool) (*combo, error) {
	if result, ok := t.combos.Get(maintainer.OrgLogin + maintainer.UserLogin); ok {
		return result.(*combo), nil
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
	} else if user == nil {
		if skipUnknowns {
			return nil, nil
		}
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on maintainer %s", maintainer.UserLogin)
	}

	var info *storage.MaintainerInfo
	if maintainer.CachedInfo == "" {
		info, err = t.store.QueryMaintainerInfo(context, maintainer)
		if err != nil {
			return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information about maintainer %s: %v", maintainer.UserLogin, err)
		}
	} else {
		var o storage.MaintainerInfo
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

	t.combos.Set(org.OrgLogin+user.UserLogin, combo)
	return combo, nil
}
