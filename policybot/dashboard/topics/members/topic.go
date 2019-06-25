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
	"net/http"
	"strings"
	"text/template"

	"istio.io/bots/policybot/pkg/util"

	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

type topic struct {
	store   storage.Store
	cache   *cache.Cache
	page    *template.Template
	context dashboard.RenderContext
	options *dashboard.Options
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

func (t *topic) Configure(htmlRouter *mux.Router, apiRouter *mux.Router, context dashboard.RenderContext, opt *dashboard.Options) {
	t.context = context
	t.options = opt

	htmlRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(t.handleListMembersHTML)

	apiRouter.StrictSlash(true).
		Path("/").
		Methods("GET").
		HandlerFunc(t.handleListMembersJSON)
}

func (t *topic) handleListMembersHTML(w http.ResponseWriter, r *http.Request) {
	orgLogin := r.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = t.options.DefaultOrg
	}

	m, err := t.getMembers(r.Context(), orgLogin)
	if err != nil {
		t.context.RenderHTMLError(w, err)
	}

	sb := &strings.Builder{}
	if err := t.page.Execute(sb, m); err != nil {
		t.context.RenderHTMLError(w, err)
		return
	}

	t.context.RenderHTML(w, sb.String())
}

func (t *topic) handleListMembersJSON(w http.ResponseWriter, r *http.Request) {
	orgLogin := r.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = "istio" // TODO: remove istio dependency
	}

	m, err := t.getMembers(r.Context(), orgLogin)
	if err != nil {
		t.context.RenderHTMLError(w, err)
		return
	}

	t.context.RenderJSON(w, http.StatusOK, m)
}

func (t *topic) getMembers(context context.Context, orgLogin string) ([]memberInfo, error) {
	org, err := t.cache.ReadOrgByLogin(context, orgLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", orgLogin, err)
	} else if org == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin)
	}

	var members []memberInfo
	if err = t.store.QueryMembersByOrg(context, org.OrgID, func(m *storage.Member) error {
		u, err := t.cache.ReadUser(context, m.UserID)
		if err != nil {
			return util.HTTPErrorf(http.StatusInternalServerError, "unable to read user information from storage: %v", err)
		}

		members = append(members, memberInfo{
			Login:     u.Login,
			Name:      u.Name,
			Company:   u.Company,
			AvatarURL: u.AvatarURL,
		})

		return nil
	}); err != nil {
		return nil, err
	}

	return members, nil
}
