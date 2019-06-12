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

package maintainers

import (
	"fmt"
	"html/template"
	"net/http"

	"istio.io/bots/policybot/pkg/fw"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
)

type Topic struct {
	store storage.Store
	ghs   *gh.GitHubState
}

type Maintainer struct {
	Login     string   `json:"login"`
	Name      string   `json:"name"`
	Company   string   `json:"company"`
	AvatarURL string   `json:"avatar_url"`
	Emeritus  bool     `json:"emeritus"`
	Paths     []string `json:"paths"`
}

func NewTopic(store storage.Store, ghs *gh.GitHubState) fw.Topic {
	return &Topic{
		store: store,
		ghs:   ghs,
	}
}

func (t *Topic) Title() string {
	return "Org Maintainers"
}

func (t *Topic) Description() string {
	return "Learn about folks that maintain Istio."
}

func (t *Topic) Prefix() string {
	return "maintainers"
}

func (t *Topic) Activate(context fw.TopicContext) {
	tmpl := template.Must(context.Layout().Parse(maintainerTemplate))

	_ = context.HTMLRouter().StrictSlash(true).NewRoute().Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if m, err := t.getMaintainers(w, req); err == nil {
			fw.RenderHTML(w, tmpl, m)
		}
	})

	_ = context.JSONRouter().StrictSlash(true).NewRoute().Methods("GET").Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if m, err := t.getMaintainers(w, req); err == nil {
			fw.RenderJSON(w, http.StatusOK, m)
		}
	})
}

func (t *Topic) getMaintainers(w http.ResponseWriter, r *http.Request) ([]Maintainer, error) {
	o := r.URL.Query().Get("org")
	if o == "" {
		o = "istio"
	}

	org, err := t.ghs.ReadOrgByLogin(o)
	if err != nil {
		err = fmt.Errorf("no information available on organization %s: %v", o, err)
		fw.RenderError(w, http.StatusNotFound, err)
		return nil, err
	} else if org == nil {
		err = fmt.Errorf("no information available on organization %s", o)
		fw.RenderError(w, http.StatusNotFound, err)
		return nil, err
	}

	var maintainers []Maintainer
	if err = t.store.QueryMaintainersByOrg(org.OrgID, func(m *storage.Maintainer) error {
		u, err := t.ghs.ReadUser(m.UserID)
		if err != nil {
			return err
		}

		maintainers = append(maintainers, Maintainer{
			Login:     u.Login,
			Name:      u.Name,
			Company:   u.Company,
			AvatarURL: u.AvatarURL,
			Emeritus:  m.Emeritus,
			Paths:     m.Paths,
		})
		return nil
	}); err != nil {
		fw.RenderError(w, http.StatusInternalServerError, err)
		return nil, err
	}

	return maintainers, nil
}
