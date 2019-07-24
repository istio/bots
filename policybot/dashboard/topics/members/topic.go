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

	"istio.io/bots/policybot/dashboard/types"

	"istio.io/bots/policybot/pkg/util"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

// Members lets users view the set of project members.
type Members struct {
	store      storage.Store
	cache      *cache.Cache
	page       *template.Template
	defaultOrg string
}

type memberInfo struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	AvatarURL string `json:"avatar_url"`
}

// New creates a new Members instance.
func New(store storage.Store, cache *cache.Cache, defaultOrg string) *Members {
	return &Members{
		store:      store,
		cache:      cache,
		page:       template.Must(template.New("page").Parse(string(MustAsset("page.html")))),
		defaultOrg: defaultOrg,
	}
}

// Renders the HTML for this topic.
func (m *Members) RenderList(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = m.defaultOrg
	}

	mi, err := m.getMembers(req.Context(), orgLogin)
	if err != nil {
		return types.RenderInfo{}, err
	}

	var sb strings.Builder
	if err := m.page.Execute(&sb, mi); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (m *Members) getMembers(context context.Context, orgLogin string) ([]memberInfo, error) {
	org, err := m.cache.ReadOrg(context, orgLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", orgLogin, err)
	} else if org == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin)
	}

	var members []memberInfo
	if err = m.store.QueryMembersByOrg(context, org.OrgLogin, func(member *storage.Member) error {
		u, err := m.cache.ReadUser(context, member.UserLogin)
		if err != nil {
			return util.HTTPErrorf(http.StatusInternalServerError, "unable to read user information from storage: %v", err)
		}

		members = append(members, memberInfo{
			Login:     u.UserLogin,
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
