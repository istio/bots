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
	"sort"
	"testing"
)

func TestNormalizeLabel(t *testing.T) {
	tests := []struct{ label, expected string }{
		{"unit", "unit"},
		{"unit+e2e", "e2e+unit"},
		{"unit+e2e+integ", "e2e+integ+unit"},
	}
	for _, test := range tests {
		actual := normalizeLabel(test.label)
		if actual != test.expected {
			t.Errorf("expected label '%s' to normalize to '%s', but got '%s'",
				test.label, test.expected, actual)
		}
	}
}

func TestGetCustomLabels(t *testing.T) {
	tests := []struct {
		cfg      Config
		expected []string
	}{
		{Config{
			"feature": &Feature{
				Stages: map[string]*Stage{
					"alpha": {
						Targets: map[string]int{
							"unit":      90,
							"e2e+integ": 90,
						},
					},
				},
			},
		}, []string{"e2e+integ"}},
	}
	for _, test := range tests {
		actual := getCustomLabels(test.cfg)
		sort.Strings(actual)
		sort.Strings(test.expected)
		if !eq(actual, test.expected) {
			t.Errorf("expected %v but got %v", test.expected, actual)
		}
	}
}

func eq(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, ae := range actual {
		if ae != expected[i] {
			return false
		}
	}
	return true
}
