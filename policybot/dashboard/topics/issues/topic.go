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
	"sort"
	"strings"
	"text/template"
	"time"

	"istio.io/bots/policybot/dashboard/types"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/bots/policybot/pkg/util"
)

// Issues lets users visualize critical information about outstanding issues.
type Issues struct {
	store      storage.Store
	cache      *cache.Cache
	list       *template.Template
	summary    *template.Template
	defaultOrg string
}

type issueInfo struct {
	RepoName    string `json:"repo"`
	IssueNumber int64  `json:"number"`
	Title       string `json:"title"`
	CreatedAt   string `json:"created"`
	UpdatedAt   string `json:"updated"`
	ClosedAt    string `json:"closed"`
	State       string `json:"state"`
	Author      string `json:"author"`
	Assignees   string `json:"assignees"`
}

type issuesSummary struct {
	Months []string
	Opened []issueCountByMonth
}

type issueCountByMonth struct {
	RepoName string
	Counts   []int
}

type listInfo struct {
	Title      string
	Issues     []issueInfo
	AreaCounts []areaCount
}

type areaCount struct {
	Area  string
	Count int
}

// New creates a new Issues instance.
func New(store storage.Store, cache *cache.Cache, defaultOrg string) *Issues {
	return &Issues{
		store:      store,
		cache:      cache,
		list:       template.Must(template.New("list").Parse(string(MustAsset("list.html")))),
		summary:    template.Must(template.New("summary").Parse(string(MustAsset("summary.html")))),
		defaultOrg: defaultOrg,
	}
}

func (i *Issues) RenderList(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = i.defaultOrg
	}

	mi, ac, err := i.getOpenIssues(req.Context(), orgLogin, "all")
	if err != nil {
		return types.RenderInfo{}, err
	}

	li := &listInfo{
		Title:      "All Open Issues",
		Issues:     mi,
		AreaCounts: ac,
	}

	var sb strings.Builder
	if err := i.list.Execute(&sb, li); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (i *Issues) RenderNeedsEscalation(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = i.defaultOrg
	}

	mi, ac, err := i.getOpenIssues(req.Context(), orgLogin, "escalation")
	if err != nil {
		return types.RenderInfo{}, err
	}

	li := &listInfo{
		Title:      "All Issues Needing Escalation",
		Issues:     mi,
		AreaCounts: ac,
	}

	var sb strings.Builder
	if err := i.list.Execute(&sb, li); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (i *Issues) RenderTriage(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = i.defaultOrg
	}

	mi, ac, err := i.getOpenIssues(req.Context(), orgLogin, "triage")
	if err != nil {
		return types.RenderInfo{}, err
	}

	li := &listInfo{
		Title:      "All Issues Needing Triage",
		Issues:     mi,
		AreaCounts: ac,
	}

	var sb strings.Builder
	if err := i.list.Execute(&sb, li); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (i *Issues) RenderStale(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = i.defaultOrg
	}

	mi, ac, err := i.getOpenIssues(req.Context(), orgLogin, "stale")
	if err != nil {
		return types.RenderInfo{}, err
	}

	li := &listInfo{
		Title:      "All Stale Issues",
		Issues:     mi,
		AreaCounts: ac,
	}

	var sb strings.Builder
	if err := i.list.Execute(&sb, li); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (i *Issues) RenderSummary(req *http.Request) (types.RenderInfo, error) {
	orgLogin := req.URL.Query().Get("org")
	if orgLogin == "" {
		orgLogin = i.defaultOrg
	}

	mi, err := i.getIssuesSummary(req.Context(), orgLogin)
	if err != nil {
		return types.RenderInfo{}, err
	}

	var sb strings.Builder
	if err := i.summary.Execute(&sb, mi); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (i *Issues) getOpenIssues(context context.Context, orgLogin string, kind string) ([]issueInfo, []areaCount, error) {
	org, err := i.cache.ReadOrg(context, orgLogin)
	if err != nil {
		return nil, nil, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", orgLogin, err)
	} else if org == nil {
		return nil, nil, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin)
	}

	areas := make(map[string]int)

	var issues []issueInfo
	if err = i.store.QueryOpenIssues(context, org.OrgLogin, func(issue *storage.Issue) error {
		keep := false
		switch kind {
		case "escalation":
			for _, lb := range issue.Labels {
				if lb == "lifecycle/needs-escalation" {
					keep = true
					break
				}
			}

		case "triage":
			for _, lb := range issue.Labels {
				if lb == "lifecycle/needs-triage" {
					keep = true
					break
				}
			}

		case "stale":
			for _, lb := range issue.Labels {
				if lb == "lifecycle/stale" {
					keep = true
					break
				}
			}
		}

		if !keep {
			return nil
		}

		foundArea := false
		for _, lb := range issue.Labels {
			if strings.HasPrefix(lb, "area/") {
				name := lb[5:]
				count := areas[name]
				areas[name] = count + 1
				foundArea = true
			}
		}

		if !foundArea {
			count := areas["unassigned"]
			areas["unassigned"] = count + 1
		}

		assignees := ""
		for _, assignee := range issue.Assignees {
			if assignees != "" {
				assignees += ",\n"
			}
			assignees += assignee
		}

		dateFormat := "02-Jan-2006"
		issues = append(issues, issueInfo{
			RepoName:    issue.RepoName,
			IssueNumber: issue.IssueNumber,
			Title:       issue.Title,
			CreatedAt:   issue.CreatedAt.Format(dateFormat),
			UpdatedAt:   issue.UpdatedAt.Format(dateFormat),
			ClosedAt:    issue.ClosedAt.Format(dateFormat),
			State:       issue.State,
			Author:      issue.Author,
			Assignees:   assignees,
		})

		return nil
	}); err != nil {
		return nil, nil, err
	}

	ac := make([]areaCount, 0, len(areas))
	for name, count := range areas {
		ac = append(ac, areaCount{Area: name, Count: count})
	}
	sort.Slice(ac, func(i, j int) bool {
		return strings.Compare(ac[i].Area, ac[j].Area) < 0
	})

	return issues, ac, nil
}

func (i *Issues) getIssuesSummary(context context.Context, orgLogin string) (issuesSummary, error) {
	var summary issuesSummary
	opened := make(map[string]map[string]int)
	var months []string

	org, err := i.cache.ReadOrg(context, orgLogin)
	if err != nil {
		return summary, util.HTTPErrorf(http.StatusInternalServerError, "unable to get information on organization %s: %v", orgLogin, err)
	} else if org == nil {
		return summary, util.HTTPErrorf(http.StatusNotFound, "no information available on organization %s", orgLogin)
	}

	if err = i.store.QueryIssues(context, org.OrgLogin, func(issue *storage.Issue) error {
		var year float64 = 24 * 365
		create := issue.CreatedAt
		if time.Since(create).Hours() < year &&
			(create.Month() != time.Now().Month() || create.Year() == time.Now().Year()) {
			monthCreated := issue.CreatedAt.Month().String()
			repo := issue.RepoName
			_, ok := opened[repo]
			if !ok {
				opened[repo] = make(map[string]int)
			}
			opened[repo][monthCreated]++
		}
		return nil
	}); err != nil {
		return summary, err
	}

	// This is a hacky way to get the ordered months for the previous year, up to the current month
	var increment time.Month = 1
	var month time.Month
	currentMonth := time.Now().Month()
	for ; increment <= 12; increment++ {
		if currentMonth+increment > 12 {
			month = currentMonth + increment - 12
		} else {
			month = currentMonth + increment
		}
		months = append(months, month.String())
	}

	var openedCounts []issueCountByMonth
	for repo, monthlyCounts := range opened {
		var counts []int
		for _, month := range months {
			counts = append(counts, monthlyCounts[month])
		}
		openedCounts = append(openedCounts, issueCountByMonth{RepoName: repo, Counts: counts})
	}

	summary.Months = months
	summary.Opened = openedCounts

	return summary, nil
}
