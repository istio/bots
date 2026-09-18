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

package syncmgr

import (
	"strings"
	"testing"

	"istio.io/bots/policybot/pkg/storage"
)

func TestConvFilterFlags(t *testing.T) {
	tests := []struct {
		flag     string
		expected FilterFlags
	}{
		{
			"notafilter",
			0,
		},
		{
			"issues",
			Issues,
		},
		{
			"prs",
			Prs,
		},
		{
			"members",
			Members,
		},
		{
			"labels",
			Labels,
		},
		{
			"repocomments",
			RepoComments,
		},
		{
			"events",
			Events,
		},
		{
			"testresults",
			TestResults,
		},
		{
			"issues,prs,maintainers,members,labels,repocomments,events,testresults",
			Issues | Prs | Maintainers | Members | Labels | RepoComments | Events | TestResults,
		},
		{
			"Issues,PRs,mAiNtAinErS,MEMBERS,labeLs,RePoComMents,EventS,TestResults",
			Issues | Prs | Maintainers | Members | Labels | RepoComments | Events | TestResults,
		},
	}

	for _, test := range tests {
		actual, _ := ConvFilterFlags(test.flag)
		if actual != test.expected {
			t.Errorf("%s: converting to filter expected %d but returned %d",
				test.flag, test.expected, actual)
		}
	}
}

func TestReadEmeritusMaintainers(t *testing.T) {
	emeritus, err := readEmeritusMaintainers(strings.NewReader(`
emeritus:
- alice
- Bob
release-managers:
  Release Managers - 1.0:
    members:
    - carol
`))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := strings.Join(emeritus, ","), "alice,Bob"; got != want {
		t.Fatalf("unexpected emeritus maintainers: got %q want %q", got, want)
	}
}

func TestMarkEmeritusMaintainerPreservesActiveMaintainerPaths(t *testing.T) {
	maintainers := map[string]*storage.Maintainer{
		"ALICE": {
			OrgLogin:  "istio",
			UserLogin: "alice",
			Paths:     []string{"istio/pilot/"},
		},
	}

	maintainer := markEmeritusMaintainer("istio", maintainers, "Alice")

	if !maintainer.Emeritus {
		t.Fatal("maintainer was not marked emeritus")
	}
	if got, want := maintainer.UserLogin, "alice"; got != want {
		t.Fatalf("unexpected maintainer login: got %q want %q", got, want)
	}
	if got, want := strings.Join(maintainer.Paths, ","), "istio/pilot/"; got != want {
		t.Fatalf("unexpected maintainer paths: got %q want %q", got, want)
	}
	if len(maintainers) != 1 {
		t.Fatalf("expected existing maintainer entry to be reused, got %d entries", len(maintainers))
	}
}

func TestMarkEmeritusMaintainerAddsNewMaintainer(t *testing.T) {
	maintainers := map[string]*storage.Maintainer{}

	maintainer := markEmeritusMaintainer("istio", maintainers, "alice")

	if !maintainer.Emeritus {
		t.Fatal("maintainer was not marked emeritus")
	}
	if got, want := maintainer.OrgLogin, "istio"; got != want {
		t.Fatalf("unexpected org login: got %q want %q", got, want)
	}
	if got, want := maintainer.UserLogin, "alice"; got != want {
		t.Fatalf("unexpected user login: got %q want %q", got, want)
	}
	if _, ok := maintainers["ALICE"]; !ok {
		t.Fatal("maintainer was not keyed case-insensitively")
	}
}
