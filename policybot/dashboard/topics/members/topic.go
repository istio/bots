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
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

type topic struct {
	store storage.Store
	cache *cache.Cache
	page  *template.Template
}

type memberInfo struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	AvatarURL string `json:"avatar_url"`
}

func NewTopic(store storage.Store, cache *cache.Cache) dashboard.Topic {
	return &topic{
		store: store,
		cache: cache,
		page:  template.Must(template.New("page").Parse(string(MustAsset("page.html")))),
	}
}

func (t *topic) Title() string {
	return "Org Members"
}

func (t *topic) Description() string {
	return "Learn about the folks that help develop and manage the Istio project"
}

func (t *topic) Name() string {
	return "members"
}

func (t *topic) Configure(htmlRouter *mux.Router, apiRouter *mux.Router, context dashboard.RenderContext) {
	htmlRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if m, err := t.getMembers(w, req); err == nil {
				sb := &strings.Builder{}
				if err := t.page.Execute(sb, m); err != nil {
					dashboard.RenderError(w, http.StatusInternalServerError, err)
					return
				}
				context.RenderHTML(w, sb.String())
			}
		})

	apiRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if m, err := t.getMembers(w, req); err == nil {
				context.RenderJSON(w, http.StatusOK, m)
			}
		})
}

func (t *topic) getMembers(w http.ResponseWriter, r *http.Request) ([]memberInfo, error) {
	o := r.URL.Query().Get("org")
	if o == "" {
		o = "istio" // TODO: remove istio dependency
	}

	org, err := t.cache.ReadOrgByLogin(r.Context(), o)
	if err != nil {
		err = fmt.Errorf("unable to get information on organization %s: %v", o, err)
		dashboard.RenderError(w, http.StatusInternalServerError, err)
		return nil, err
	} else if org == nil {
		err = fmt.Errorf("no information available on organization %s", o)
		dashboard.RenderError(w, http.StatusNotFound, err)
		return nil, err
	}

	var members []memberInfo
	if err = t.store.QueryMembersByOrg(r.Context(), org.OrgID, func(m *storage.Member) error {
		u, err := t.cache.ReadUser(r.Context(), m.UserID)
		if err != nil {
			return err
		}

		members = append(members, memberInfo{
			Login:     u.Login,
			Name:      u.Name,
			Company:   u.Company,
			AvatarURL: u.AvatarURL,
		})
		return nil
	}); err != nil {
		dashboard.RenderError(w, http.StatusInternalServerError, err)
		return nil, err
	}

	return members, nil
}
