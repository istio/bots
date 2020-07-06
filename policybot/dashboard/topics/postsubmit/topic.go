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

package postsubmit

import (
	"time"
	"context"
	"net/http"
	"strings"
	"text/template"

	"istio.io/bots/policybot/dashboard/types"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

// PostSubmit lets users visualize critical information about the project's outstanding pull requests.
type PostSubmit struct {
	store    storage.Store
	cache    *cache.Cache
	baseSha  *template.Template
	page     string
}

type baseShaSummary struct {
	BaseSha			[]string
	LastFinishTime  []time.Time
	NumberofTest 	[]int64
}

// New creates a new PostSubmit instance.
func New(store storage.Store, cache *cache.Cache) *PostSubmit {
	return &PostSubmit{
		store: store,
		cache: cache,
		page:  string(MustAsset("page.html")),
	}
}

// Renders the HTML for this topic.
func (ps *PostSubmit) Render(req *http.Request) (types.RenderInfo, error) {
	return types.RenderInfo{
		Content: ps.page,
	}, nil
}

func (ps *PostSubmit) RenderLatestBaseSha(req *http.Request) (types.RenderInfo, error) {
	baseShas, err := ps.getLatestBaseShas(req.Context())
	if err != nil {
		return types.RenderInfo{}, err
	}

	var sb strings.Builder
	if err := ps.baseSha.Execute(&sb, baseShas); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (i *PostSubmit) getLatestBaseShas(context context.Context) (baseShaSummary, error) {
	var summary baseShaSummary
	var BaseShaList []string
	var LastFinishTimeList []time.Time
	var NumberofTestList []int64

	if err := i.store.QueryLatestBaseSha(context, func(latestBaseSha *storage.LatestBaseSha) error {
		BaseShaList = append(BaseShaList,latestBaseSha.BaseSha)
		LastFinishTimeList = append(LastFinishTimeList,latestBaseSha.LastFinishTime)
		NumberofTestList = append(NumberofTestList,latestBaseSha.NumberofTest)
		return nil
	}); err != nil {
		return summary, err
	}

	return summary, nil
}
