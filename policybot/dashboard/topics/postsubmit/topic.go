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
	"context"
	"net/http"
	"strings"
	"text/template"
	"time"
	"fmt"
	"istio.io/bots/policybot/dashboard/types"

	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
)

// PostSubmit lets users visualize critical information about the project's outstanding pull requests.
type PostSubmit struct {
	store         storage.Store
	cache         *cache.Cache
	latestbaseSha *template.Template
	baseSha       *template.Template
}

type BaseShaSummary struct {
	BaseSha        []string
	LastFinishTime []time.Time
	NumberofTest   []int64
}

type BaseShas struct {
	BaseSha        []string
}

type LabelEnvSummary struct {
	LabelEnv []LabelEnv
}

type LabelEnv struct {
	Label  string
	EnvCount    []EnvCount
}

type EnvCount struct {
	Env      string
	Counts   int
}


// New creates a new PostSubmit instance.
func New(store storage.Store, cache *cache.Cache) *PostSubmit {
	return &PostSubmit{
		store:         store,
		cache:         cache,
		latestbaseSha: template.Must(template.New("page").Parse(string(MustAsset("page.html")))),
		baseSha:       template.Must(template.New("analysis").Parse(string(MustAsset("analysis.html")))),
	}
}

func (ps *PostSubmit) RenderAllBaseSha(req *http.Request) (types.RenderInfo, error) {
	var analysis_page BaseShas
	allBaseShas, err := ps.store.QueryAllBaseSha(req.Context())
	analysis_page.BaseSha = allBaseShas
	if err != nil {
		return types.RenderInfo{}, err
	}
    myHandler(req)
	var sb strings.Builder
	if err := ps.baseSha.Execute(&sb, analysis_page); err != nil {
		return types.RenderInfo{}, err
	}
	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (ps *PostSubmit) RenderLatestBaseSha(req *http.Request) (types.RenderInfo, error) {
	baseShas, err := ps.getLatestBaseShas(req.Context())
	if err != nil {
		return types.RenderInfo{}, err
	}

	var sb strings.Builder
	if err := ps.latestbaseSha.Execute(&sb, baseShas); err != nil {
		return types.RenderInfo{}, err
	}

	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (ps *PostSubmit) getLatestBaseShas(context context.Context) (BaseShaSummary, error) {
	var summary BaseShaSummary
	var BaseShaList []string
	var LastFinishTimeList []time.Time
	var NumberofTestList []int64

	if err := ps.store.QueryLatestBaseSha(context, func(latestBaseSha *storage.LatestBaseSha) error {
		BaseShaList = append(BaseShaList, latestBaseSha.BaseSha)
		LastFinishTimeList = append(LastFinishTimeList, latestBaseSha.LastFinishTime)
		NumberofTestList = append(NumberofTestList, latestBaseSha.NumberofTest)
		return nil
	}); err != nil {
		return summary, err
	}
	summary.BaseSha = BaseShaList
	summary.LastFinishTime = LastFinishTimeList
	summary.NumberofTest = NumberofTestList
	return summary, nil
}

func (ps *PostSubmit) getLabelEnvTable(context context.Context, baseSha string) (LabelEnvSummary, error) {
	var summary LabelEnvSummary
	var Labels = make(map[string]map[string]int)
	//var LabelTree = make(map[string]map[string]int)

	if err := ps.store.QueryPostSubmitTestResult(context, baseSha, func(postSubmitTestResult *storage.PostSubmitTestResultDenormalized) error {
		_, ok := Labels[postSubmitTestResult.Label]
		if !ok {
			Labels[postSubmitTestResult.Label] = make(map[string]int)
		}
		Labels[postSubmitTestResult.Label][postSubmitTestResult.Environment]++
		return nil
	}); err != nil {
		return summary, err
	}
	/*
	for label, env := range Labels{
		splitlabel := strings.Split(label, ".") 

    }*/
	var labelEnvList []LabelEnv
	for label, envMap := range Labels {
		var labelEnv LabelEnv 
		var newEnvCount []EnvCount
		for env, count := range envMap {
			newEnvCount = append(newEnvCount, EnvCount{Env:env, Counts:count})
		}
		labelEnv.Label = label
		labelEnv.EnvCount = newEnvCount
		labelEnvList = append(labelEnvList, labelEnv)
	}
	summary.LabelEnv = labelEnvList

	return summary, nil
}

func myHandler(req *http.Request) {
    fmt.Println(req.FormValue("choseBaseSha"))
}