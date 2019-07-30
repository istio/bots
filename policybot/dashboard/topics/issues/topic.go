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

package issues

import (
	"context"
	"net/http"
	"strings"
	"text/template"
	"time"

	"istio.io/bots/policybot/dashboard/types"

	"istio.io/bots/policybot/pkg/util"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

// Issues lets users view data about issues for all repositories.
type Issues struct {
	store      storage.Store
	cache      *cache.Cache
	page       *template.Template
	defaultOrg string
}

type issueInfo struct {
	RepoName    string    `json:"repo"`
	IssueNumber int64     `json:"number"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created"`
	UpdatedAt   time.Time `json:"updated"`
	ClosedAt    time.Time `json:"closed"`
	State       string    `json:"state"`
	Author      string    `json:"author"`
	Assignees   []string  `json:"assignees"`
}

// New creates a new Issues instance.
func New(store storage.Store, cache *cache.Cache, defaultOrg string) *Issues {
	return &Issues{
		store:      store,
		cache:      cache,
		page:       template.Must(template.New("page").Parse(string(MustAsset("page.html")))),
		defaultOrg: defaultOrg,
	}
}

// Renders the HTML for this topic.
func (i *Issues) RenderList(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = i.defaultOrg
	}

	mi, err := i.getIssues(req.Context(), orgLogin)
	if err != nil {
		return types.RenderInfo{}, err
	}

	var sb strings.Builder
	if err := i.page.Execute(&sb, mi); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (i *Issues) getIssues(context context.Context, orgLogin string) ([]issueInfo, error) {
	org, err := i.cache.ReadOrg(context, orgLogin)
	if err != nil {
		return nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", orgLogin, err)
	} else if org == nil {
		return nil, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin)
	}

	var issues []issueInfo
	if err = i.store.QueryOpenIssues(context, org.OrgLogin, func(issue *storage.Issue) error {
		issues = append(issues, issueInfo{
			RepoName:    issue.RepoName,
			IssueNumber: issue.IssueNumber,
			Title:       issue.Title,
			CreatedAt:   issue.CreatedAt,
			UpdatedAt:   issue.UpdatedAt,
			ClosedAt:    issue.ClosedAt,
			State:       issue.State,
			Author:      issue.Author,
			Assignees:   issue.Assignees,
		})

		return nil
	}); err != nil {
		return nil, err
	}

	return issues, nil
}
