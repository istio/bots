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

package members

import (
	"fmt"
	"html/template"
	"net/http"

	"istio.io/bots/policybot/pkg/fw"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

type Topic struct {
	store storage.Store
	cache *cache.Cache
}

type Member struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	AvatarURL string `json:"avatar_url"`
}

func NewTopic(store storage.Store, cache *cache.Cache) fw.Topic {
	return &Topic{
		store: store,
		cache: cache,
	}
}

func (t *Topic) Title() string {
	return "Org Members"
}

func (t *Topic) Description() string {
	return "Learn about the folks that help develop and manage the Istio project"
}

func (t *Topic) Prefix() string {
	return "members"
}

func (t *Topic) Activate(context fw.TopicContext) {
	tmpl := template.Must(context.Layout().Parse(membersTemplate))

	_ = context.HTMLRouter().StrictSlash(true).NewRoute().Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if m, err := t.getMembers(w, req); err == nil {
			fw.RenderHTML(w, tmpl, m)
		}
	})

	_ = context.JSONRouter().StrictSlash(true).NewRoute().Methods("GET").Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if m, err := t.getMembers(w, req); err == nil {
			fw.RenderJSON(w, http.StatusOK, m)
		}
	})
}

func (t *Topic) getMembers(w http.ResponseWriter, r *http.Request) ([]Member, error) {
	o := r.URL.Query().Get("org")
	if o == "" {
		o = "istio"
	}

	org, err := t.cache.ReadOrgByLogin(o)
	if err != nil {
		err = fmt.Errorf("no information available on organization %s: %v", o, err)
		fw.RenderError(w, http.StatusNotFound, err)
		return nil, err
	} else if org == nil {
		err = fmt.Errorf("no information available on organization %s", o)
		fw.RenderError(w, http.StatusNotFound, err)
		return nil, err
	}

	var members []Member
	if err = t.store.QueryMembersByOrg(org.OrgID, func(m *storage.Member) error {
		u, err := t.cache.ReadUser(m.UserID)
		if err != nil {
			return err
		}

		members = append(members, Member{
			Login:     u.Login,
			Name:      u.Name,
			Company:   u.Company,
			AvatarURL: u.AvatarURL,
		})
		return nil
	}); err != nil {
		fw.RenderError(w, http.StatusInternalServerError, err)
		return nil, err
	}

	return members, nil
}
