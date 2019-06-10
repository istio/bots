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
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"istio.io/bots/policybot/pkg/fw"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
)

type MaintainerQueries struct {
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

func NewMaintainerQueries(store storage.Store, ghs *gh.GitHubState) fw.Topic {
	return &MaintainerQueries{
		store: store,
		ghs:   ghs,
	}
}

func (mq *MaintainerQueries) Title() string {
	return "Maintainers"
}

func (mq *MaintainerQueries) Prefix() string {
	return "maintainers"
}

func (mq *MaintainerQueries) Activate(context fw.TopicContext) {
	tmpl := template.Must(context.Layout().Parse(maintainerTemplate))

	_ = context.HTMLRouter().StrictSlash(true).NewRoute().Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, _ := mq.getMaintainers(w, req)
		fw.RenderHTML(w, tmpl, m)
	})

	_ = context.JSONRouter().StrictSlash(true).NewRoute().Methods("GET").Path("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m, _ := mq.getMaintainers(w, req)
		fw.RenderJSON(w, http.StatusOK, m)
	})
}

func (mq *MaintainerQueries) getMaintainers(w http.ResponseWriter, r *http.Request) ([]Maintainer, error) {
	o := r.URL.Query().Get("org")
	if o == "" {
		fw.RenderError(w, http.StatusBadRequest, errors.New("no org query parameter specified"))
		return nil, nil
	}

	org, err := mq.ghs.ReadOrgByLogin(o)
	if err != nil {
		fw.RenderError(w, http.StatusNotFound, fmt.Errorf("no information available on organization %s", org))
		return nil, nil
	}

	var maintainers []Maintainer
	if err = mq.store.QueryMaintainersByOrg(org.OrgID, func(m *storage.Maintainer) error {
		u, err := mq.ghs.ReadUser(m.UserID)
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
