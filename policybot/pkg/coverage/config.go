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

package coverage

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"istio.io/pkg/cache"
)

type Config map[string]*Feature

type Feature struct {
	Stages map[string]*Stage
}

type Stage struct {
	Targets  map[string]int
	Packages []string
}

var cfgCache = cache.NewTTL(5*time.Minute, time.Minute)

func getConfig(org, repo string) (Config, error) {
	key := fmt.Sprintf("%s/%s/master", org, repo)
	if cfg, ok := cfgCache.Get(key); ok {
		return cfg.(Config), nil
	}
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/.istio-codecov.yaml", key)
	resp, err := http.Get(url)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return nil, err
	}
	for _, f := range cfg {
		for _, s := range f.Stages {
			normalized := make(map[string]int)
			for label, target := range s.Targets {
				normalized[normalizeLabel(label)] = target
			}
			s.Targets = normalized
		}
	}
	cfgCache.Set(key, &cfg)
	return cfg, nil
}

// normalizeLabel returns an equivalent label whose parts are sorted alphabetically.
func normalizeLabel(label string) string {
	if strings.Contains(label, "+") {
		parts := strings.Split(label, "+")
		sort.Sort(sort.StringSlice(parts))
		label = strings.Join(parts, "+")
	}
	return label
}

func getCustomLabels(cfg Config) []string {
	labelsMap := make(map[string]bool)
	for _, f := range cfg {
		for _, s := range f.Stages {
			for label := range s.Targets {
				if strings.Contains(label, "+") {
					labelsMap[label] = true
				}
			}
		}
	}
	var labels []string
	for label := range labelsMap {
		labels = append(labels, label)
	}
	return labels
}
