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

	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"

	"istio.io/bots/policybot/dashboard/types"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// PostSubmit lets users visualize critical information about which features have been covered and which not after Github PR merged.
type PostSubmit struct {
	store         storage.Store
	cache         *cache.Cache
	router        *mux.Router
	latestbaseSha *template.Template
	baseSha       *template.Template
	analysis      *template.Template
	choosesha     string //basesha user choose to view
	chooseEnv     string //environment of selected basesha user wants to generate detailed test name
	chooseLabel   string //label of selected basesha user wants to generate detailed test name

}

//List of recent 100 commits with info
type LatestBaseShaSummary struct {
	LatestBaseSha []LatestBaseSha
}

//recent commits with BaseSha, last test finish time and number of test has done
type LatestBaseSha struct {
	BaseSha        string
	LastFinishTime time.Time
	NumberofTest   int64
}

//List of all BaseShas
type BaseShas struct {
	BaseSha []string
}

//Content for LabelEnv table and detailed tests
type LabelEnvSummary struct {
	Choosesha           string
	ChooseEnv           string
	ChooseLabel         string
	LabelEnv            []LabelEnv
	AllEnvNanme         []string
	TestNameByEnvLabels []*storage.TestNameByEnvLabel
}

//Label, corresponding count for different environement, and it's sublabel table
type LabelEnv struct {
	Label    string
	EnvCount []int
	SubLabel LabelEnvSummary
}

// New creates a new PostSubmit instance.
func New(store storage.Store, cache *cache.Cache, router *mux.Router) *PostSubmit {
	convertIntToMap := template.FuncMap{
		"slice": func(i int) []int { return make([]int, i) },
	}
	wrap := template.FuncMap{
		"wrap": func(summary LabelEnvSummary, depth int) map[string]interface{} {
			return map[string]interface{}{"LabelEnv": summary.LabelEnv, "Depth": depth + 1}
		},
	}
	ps := &PostSubmit{
		store:         store,
		cache:         cache,
		router:        router,
		latestbaseSha: template.Must(template.New("page").Parse(string(MustAsset("page.html")))),
		baseSha:       template.Must(template.New("chooseBaseSha").Parse(string(MustAsset("chooseBaseSha.html")))),
		analysis:      template.Must(template.New("analysis").Funcs(convertIntToMap).Funcs(wrap).Parse(string(MustAsset("analysis.html")))),
	}
	router.HandleFunc("/savebasesha", ps.chosenBaseSha)
	router.HandleFunc("/selectEnvLabel", ps.selectEnvLabel)
	go ps.cache.WriteLatestBaseShas()
	cron := cron.New()
	_, err := cron.AddFunc("@hourly", ps.cache.WriteLatestBaseShas)
	if err != nil {
		log.Errorf("add caching latest 100 BaseSha cron job: %s", err)
	}
	cron.Start()
	return ps
}

func (ps *PostSubmit) chosenBaseSha(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("reading post BaseSha: %s", err)
	}
	baseSha := r.FormValue("basesha")
	ps.choosesha = baseSha
	ps.chooseEnv = ""
	ps.chooseLabel = ""
}

func (ps *PostSubmit) selectEnvLabel(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("reading selected Env and Label: %s", err)
	}
	env := r.FormValue("env")
	label := r.FormValue("label")
	ps.chooseEnv = env
	ps.chooseLabel = label
}

func (ps *PostSubmit) RenderLabelEnv(req *http.Request) (types.RenderInfo, error) {
	var summary LabelEnvSummary
	summary, err := ps.getLabelEnvTable(req.Context(), ps.choosesha)
	if err != nil {
		return types.RenderInfo{}, err
	}
	if ps.choosesha != "" {
		testNameByEnvLabels, err := ps.store.QueryTestNameByEnvLabel(req.Context(), ps.choosesha, ps.chooseEnv, ps.chooseLabel)
		if err != nil {
			return types.RenderInfo{}, err
		}
		summary.TestNameByEnvLabels = testNameByEnvLabels
	}

	var sb strings.Builder
	if err := ps.analysis.Execute(&sb, summary); err != nil {
		return types.RenderInfo{}, err
	}
	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (ps *PostSubmit) RenderAllBaseSha(req *http.Request) (types.RenderInfo, error) {
	var chooseBaseShaPage BaseShas
	allBaseShas, err := ps.store.QueryAllBaseSha(req.Context())
	chooseBaseShaPage.BaseSha = allBaseShas
	if err != nil {
		return types.RenderInfo{}, err
	}
	var sb strings.Builder
	if err := ps.baseSha.Execute(&sb, chooseBaseShaPage); err != nil {
		return types.RenderInfo{}, err
	}
	return types.RenderInfo{
		Content: sb.String(),
	}, nil
}

func (ps *PostSubmit) RenderLatestBaseSha(req *http.Request) (types.RenderInfo, error) {
	baseShas, err := ps.cache.ReadLatestBaseShas()
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

func (ps *PostSubmit) getLabelEnvTable(context context.Context, baseSha string) (LabelEnvSummary, error) {
	var summary LabelEnvSummary
	var Labels = make(map[string]map[string]int)
	var allEnvNames = make(map[string]int)

	if err := ps.store.QueryPostSubmitTestEnvLabel(context, baseSha, func(postSubmitTestResult *storage.PostSubmitTestEnvLabel) error {
		_, ok := Labels[postSubmitTestResult.Label]
		if !ok {
			Labels[postSubmitTestResult.Label] = make(map[string]int)
		}
		Labels[postSubmitTestResult.Label][postSubmitTestResult.Environment]++
		if _, ok := allEnvNames[postSubmitTestResult.Environment]; !ok {
			allEnvNames[postSubmitTestResult.Environment] = len(allEnvNames)
		}
		return nil
	}); err != nil {
		return summary, err
	}

	summary = ps.getLabelTree(Labels, allEnvNames)

	var EnvNameList = make([]string, len(allEnvNames))
	for key, val := range allEnvNames {
		EnvNameList[val] = key
	}
	summary.AllEnvNanme = EnvNameList
	summary.Choosesha = ps.choosesha
	summary.ChooseEnv = ps.chooseEnv
	summary.ChooseLabel = ps.chooseLabel
	return summary, nil
}

func (ps *PostSubmit) getLabelTree(input map[string]map[string]int, envNames map[string]int) LabelEnvSummary {
	if len(input) < 1 {
		return LabelEnvSummary{}
	}
	var toplayer = make(map[string]map[string]int)
	var nextLayer = make(map[string]map[string]map[string]int)
	var nextLayerSummary = make(map[string]LabelEnvSummary)
	for label, envMap := range input {
		splitlabel := strings.Split(label, ".")
		_, ok := toplayer[splitlabel[0]]
		if !ok {
			toplayer[splitlabel[0]] = make(map[string]int)
		}
		for env, count := range envMap {
			toplayer[splitlabel[0]][env] += count
		}
		//add content after first dot to the map for the next layer
		if len(splitlabel) < 2 {
			continue
		}
		_, ok = nextLayer[splitlabel[0]]
		if !ok {
			nextLayer[splitlabel[0]] = make(map[string]map[string]int)
		}
		nextLayer[splitlabel[0]][strings.Join(splitlabel[1:], ".")] = envMap
	}

	for topLayerName, nextLayerMap := range nextLayer {
		nextLayerSummary[topLayerName] = ps.getLabelTree(nextLayerMap, envNames)
	}
	return ps.convertMapToSummary(toplayer, nextLayerSummary, envNames)
}

func (ps *PostSubmit) convertMapToSummary(input map[string]map[string]int, nextLayer map[string]LabelEnvSummary,
	envNames map[string]int) (summary LabelEnvSummary) {
	var labelEnvList []LabelEnv
	for label, envMap := range input {
		var labelEnv LabelEnv
		var envCount = make([]int, len(envNames))
		for env, count := range envMap {
			envCount[envNames[env]] = count
		}
		labelEnv.Label = label
		labelEnv.EnvCount = envCount
		labelEnv.SubLabel = nextLayer[label]
		labelEnvList = append(labelEnvList, labelEnv)
	}
	summary.LabelEnv = labelEnvList
	return
}
