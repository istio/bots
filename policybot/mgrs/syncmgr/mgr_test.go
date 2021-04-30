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
	"testing"
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
