// Copyright Istio Authors
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

package cleaner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"
)

func TestBoilerplates(t *testing.T) {
	configPath := "../../../config/boilerplates"
	configDir, err := os.ReadDir(configPath)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		matchers []string
	}{
		{
			"issue-default",
			[]string{
				"area-selection",
				"valid-report-check",
			},
		},
		{
			"pr-default",
			[]string{
				"pr-area-selection",
				"pr-release-note",
			},
		},
	}
	boilerplates := map[string]*regexp.Regexp{}
	for _, bp := range configDir {
		f, err := os.ReadFile(filepath.Join(configPath, bp.Name()))
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(bp.Name(), ".yaml")
		var rec boilerplateRecord
		if err := yaml.Unmarshal(f, &rec); err != nil {
			t.Fatal(err)
		}
		rx, err := regexp.Compile("(?mis)" + rec.Regex)
		if err != nil {
			t.Fatal(err)
		}
		boilerplates[name] = rx
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join("testdata", tt.name+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			expected := map[string]bool{}
			for _, m := range tt.matchers {
				expected[m] = true
			}

			result := string(input)
			for bn, rx := range boilerplates {
				matched := rx.Match(input)
				if matched != expected[bn] {
					t.Errorf("expected match=%v, got match=%v for %v", expected[bn], matched, bn)
				}
				// Code allows custom replacement but its never used, so for now just test replace with empty
				result = rx.ReplaceAllString(result, "")
			}
			gp := filepath.Join("testdata", tt.name+".out.txt")
			if _, f := os.LookupEnv("REFRESH_GOLDEN"); f {
				if err := os.WriteFile(gp, []byte(result), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			expectedResult, err := os.ReadFile(gp)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(result, string(expectedResult)); diff != "" {
				t.Fatalf("did not get expected output. diff: %v", diff)
			}
		})
	}
	t.Run("coverage", func(t *testing.T) {
		for _, tt := range cases {
			for _, m := range tt.matchers {
				delete(boilerplates, m)
			}
		}
		if len(boilerplates) > 0 {
			for k := range boilerplates {
				t.Logf("Missing %v", k)
			}
			t.Fatalf("Not all boilerplates included in positive tests")
		}
	})
}
