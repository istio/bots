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

package perf

import (
	"html/template"
	"net/http"
	"strings"

	"istio.io/bots/policybot/dashboard/templates/widgets"
	"istio.io/bots/policybot/dashboard/types"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

// Perf lets users visualize stats about project performance.
type Perf struct {
	store storage.Store
	cache *cache.Cache
	page  *template.Template
}

// New creates a new Perf instance.
func New(store storage.Store, cache *cache.Cache) *Perf {
	page := template.Must(template.New("page").Parse(string(MustAsset("page.html"))))
	_ = template.Must(page.Parse(widgets.TimeSeriesInitTemplate))
	_ = template.Must(page.Parse(widgets.TimeSeriesTemplate))

	return &Perf{
		store: store,
		cache: cache,
		page:  page,
	}
}

// Renders the HTML for this topic.
func (p *Perf) Render(req *http.Request) (types.RenderInfo, error) {
	var sb strings.Builder
	if err := p.page.Execute(&sb, p.getPerformanceResults()); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

type Result struct {
	Name       string
	TimeSeries string
	Target     string
}

func (p *Perf) getPerformanceResults() []Result {
	results := []Result{
		{Name: "Perf Test 1", Target: "perftest1", TimeSeries: p.getTimeSeries1()},
		{Name: "Perf Test 2", Target: "perftest2", TimeSeries: p.getTimeSeries2()},
	}
	return results
}

func (p *Perf) getTimeSeries1() string {
	return `[
		{"date": "2014-01-01", "value": 109865},
		{"date": "2014-01-02", "value": 34579},
		{"date": "2014-01-03", "value": 34908},
		{"date": "2014-01-04", "value": 85250},
		{"date": "2014-01-05", "value": 91904},
		{"date": "2014-01-06", "value": 76838},
		{"date": "2014-01-07", "value": 13854},
		{"date": "2014-01-08", "value": 77211},
		{"date": "2014-01-09", "value": 2210},
		{"date": "2014-01-10", "value": 81072},
		{"date": "2014-01-11", "value": 52310},
		{"date": "2014-01-12", "value": 31790},
		{"date": "2014-01-13", "value": 48881},
		{"date": "2014-01-14", "value": 64037},
		{"date": "2014-01-15", "value": 20685},
		{"date": "2014-01-16", "value": 6418},
		{"date": "2014-01-17", "value": 22924},
		{"date": "2014-01-18", "value": 37480},
		{"date": "2014-01-19", "value": 58882},
		{"date": "2014-01-20", "value": 29538},
		{"date": "2014-01-21", "value": 6897},
		{"date": "2014-01-22", "value": 99711},
		{"date": "2014-01-23", "value": 59017},
		{"date": "2014-01-24", "value": 6183},
		{"date": "2014-01-25", "value": 7346},
		{"date": "2014-01-26", "value": 59852},
		{"date": "2014-01-27", "value": 70783},
		{"date": "2014-01-28", "value": 67768},
		{"date": "2014-01-29", "value": 632803},
		{"date": "2014-01-30", "value": 25316},
		{"date": "2014-01-31", "value": 26177}]`
}

func (p *Perf) getTimeSeries2() string {
	return `[
		{"date": "2014-01-01", "value": 26547},
		{"date": "2014-01-02", "value": 978098},
		{"date": "2014-01-03", "value": 345},
		{"date": "2014-01-04", "value": 45632},
		{"date": "2014-01-05", "value": 784637},
		{"date": "2014-01-06", "value": 4564},
		{"date": "2014-01-07", "value": 736478},
		{"date": "2014-01-08", "value": 34566},
		{"date": "2014-01-09", "value": 36578},
		{"date": "2014-01-10", "value": 59477},
		{"date": "2014-01-11", "value": 78042},
		{"date": "2014-01-12", "value": 75438},
		{"date": "2014-01-13", "value": 243588},
		{"date": "2014-01-14", "value": 23457},
		{"date": "2014-01-15", "value": 7980},
		{"date": "2014-01-16", "value": 6418},
		{"date": "2014-01-17", "value": 22924},
		{"date": "2014-01-18", "value": 37480},
		{"date": "2014-01-19", "value": 58882},
		{"date": "2014-01-20", "value": 29538},
		{"date": "2014-01-21", "value": 3599},
		{"date": "2014-01-22", "value": 99711},
		{"date": "2014-01-23", "value": 45689},
		{"date": "2014-01-24", "value": 6183},
		{"date": "2014-01-25", "value": 3546},
		{"date": "2014-01-26", "value": 59852},
		{"date": "2014-01-27", "value": 23007},
		{"date": "2014-01-28", "value": 67768},
		{"date": "2014-01-29", "value": 98756},
		{"date": "2014-01-30", "value": 25316},
		{"date": "2014-01-31", "value": 26177}]`
}
