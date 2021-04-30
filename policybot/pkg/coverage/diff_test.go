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
	"errors"
	"reflect"
	"testing"

	"istio.io/bots/policybot/pkg/storage"
)

var config = Config{
	"all": &Feature{
		Stages: map[string]*Stage{
			"stable": {
				Packages: []string{"istio.io/bots/policybot"},
				Targets: map[string]float64{
					"unit": 70,
				},
			},
		},
	},
}

func coverage(label string, covered int64) *storage.CoverageData {
	return &storage.CoverageData{
		PackageName:  "istio.io/bots/policybot/pkg/coverage",
		StmtsCovered: covered,
		StmtsTotal:   100,
		Type:         label,
	}
}

func TestComputeDiffResult(t *testing.T) {
	tests := []struct {
		name             string
		cfg              Config
		baseCov, headCov map[string][]*storage.CoverageData
		expected         *DiffResult
	}{
		{
			name: "base",
			cfg:  config,
			baseCov: map[string][]*storage.CoverageData{
				"istio.io/bots/policybot/pkg/coverage": {coverage("unit", 1)},
			},
			headCov: map[string][]*storage.CoverageData{
				"istio.io/bots/policybot/pkg/coverage": {coverage("unit", 70)},
			},
			expected: &DiffResult{},
		},
		{
			name: "failure",
			cfg:  config,
			baseCov: map[string][]*storage.CoverageData{
				"istio.io/bots/policybot/pkg/coverage": {coverage("unit", 1)},
			},
			headCov: map[string][]*storage.CoverageData{
				"istio.io/bots/policybot/pkg/coverage": {coverage("unit", 1)},
			},
			expected: &DiffResult{
				Entries: []*DiffResultEntry{
					{
						Feature: "all",
						Stage:   "stable",
						Label:   "unit",
						Target:  70,
						Actual:  1,
						Base:    1,
					},
				},
			},
		},
		{
			name: "ignoreLabel",
			cfg:  config,
			baseCov: map[string][]*storage.CoverageData{
				"istio.io/bots/policybot/pkg/coverage": {
					coverage("unit", 1), coverage("e2e", 100),
				},
			},
			headCov: map[string][]*storage.CoverageData{
				"istio.io/bots/policybot/pkg/coverage": {
					coverage("unit", 1), coverage("e2e", 100),
				},
			},
			expected: &DiffResult{
				Entries: []*DiffResultEntry{
					{
						Feature: "all",
						Stage:   "stable",
						Label:   "unit",
						Target:  70,
						Actual:  1,
						Base:    1,
					},
				},
			},
		},
	}

	for _, test := range tests {
		actual := computeDiffResult(test.cfg, test.baseCov, test.headCov)
		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("[%s]: expected %+v but got %+v", test.name, test.expected, actual)
			t.Errorf("Actual entries:")
			for _, e := range actual.Entries {
				t.Errorf("\t%+v", *e)
			}
			t.Errorf("Expected entries:")
			for _, e := range test.expected.Entries {
				t.Errorf("\t%+v", *e)
			}
		}
	}
}

func TestDiffResultGithubStatus(t *testing.T) {
	tests := []struct {
		diffResult DiffResult
		expected   string
	}{
		{
			DiffResult{},
			Success,
		},
		{
			DiffResult{
				err: errors.New(""),
			},
			Error,
		},
		{
			DiffResult{
				Entries: []*DiffResultEntry{
					{},
				},
			},
			Failure,
		},
	}

	for _, test := range tests {
		actual := test.diffResult.GetGithubStatus()
		if actual != test.expected {
			t.Errorf("[%+v] expected status %s but got %s", test.diffResult, test.expected, actual)
		}
	}
}
