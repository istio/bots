// Copyright 2020 Istio Authors
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

package releasequalification

import (
	"context"
	"html/template"
	"net/http"
	"strings"
	"time"

	"istio.io/bots/policybot/dashboard/types"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// ReleaseQualification shows status of release qualification tests.
type ReleaseQualification struct {
	store         storage.Store
	cache         *cache.Cache
	monitorReport *template.Template
}

// SingleMonitorStatus represents the status of one single monitor
type SingleMonitorStatus struct {
	Name        string
	Status      string
	TestID      string
	ProjectID   string
	ClusterName string
	FiredTimes  int64
	UpdatedTime time.Time
}

// AggregatedMonitorStatus is aggregation of monitor statuses, key is the monitor name.
type AggregatedMonitorStatus map[string]*SingleMonitorStatus

// MonitorReport represents the data used for rendering the HTML page.
type MonitorReport struct {
	Branches       []string
	StatusByBranch map[string]AggregatedMonitorStatus
}

// New creates a new ReleaseQualification instance.
func New(store storage.Store, cache *cache.Cache) *ReleaseQualification {
	return &ReleaseQualification{
		store:         store,
		cache:         cache,
		monitorReport: template.Must(template.New("ReleaseQualification").Parse(string(MustAsset("page.html")))),
	}
}

// Render renders the HTML for this topic.
func (r *ReleaseQualification) Render(req *http.Request) (types.RenderInfo, error) {
	ms, err := r.getMonitorStatus(req.Context())
	if err != nil {
		return types.RenderInfo{}, err
	}

	var sb strings.Builder
	if err := r.monitorReport.Execute(&sb, ms); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (r *ReleaseQualification) getMonitorStatus(context context.Context) (MonitorReport, error) {
	var mr MonitorReport
	// StatusByBranch is mapping from branch name to aggregatedMonitorStatus
	mr.StatusByBranch = make(map[string]AggregatedMonitorStatus)

	if err := r.store.QueryMonitorStatus(context, func(monitor *storage.Monitor) error {
		branch := monitor.Branch
		if branch == "" {
			log.Warn("monitor branch is empty")
			return nil
		}
		if _, ok := mr.StatusByBranch[branch]; !ok {
			mr.Branches = append(mr.Branches, branch)
			mr.StatusByBranch[branch] = make(AggregatedMonitorStatus)
		}
		ms := mr.StatusByBranch[branch]
		monitorName := monitor.MonitorName
		if _, ok := ms[monitorName]; !ok {
			ms[monitorName] = &SingleMonitorStatus{}
		}
		sms := ms[monitorName]
		if sms.UpdatedTime.IsZero() || sms.UpdatedTime.Before(monitor.UpdatedTime) {
			sms.Name = monitorName
			sms.Status = monitor.Status
			sms.UpdatedTime = monitor.UpdatedTime
			sms.ClusterName = monitor.ClusterName
			sms.ProjectID = monitor.ProjectID
			sms.TestID = monitor.TestID
			sms.FiredTimes = monitor.FiredTimes
		}
		return nil
	}); err != nil {
		return mr, err
	}
	for branch, ags := range mr.StatusByBranch {
		log.Infof("monitor status for branch: %v\n", branch)
		for _, sgs := range ags {
			log.Infof("single status: %v\n", sgs)
		}
	}
	return mr, nil
}
