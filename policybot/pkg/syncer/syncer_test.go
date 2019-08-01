package syncer

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
			"zenhub",
			ZenHub,
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
			"issues,prs,maintainers,members,labels,zenhub,repocomments,events,testresults",
			Issues | Prs | Maintainers | Members | Labels | ZenHub | RepoComments | Events | TestResults,
		},
		{
			"Issues,PRs,mAiNtAinErS,MEMBERS,labeLs,zEnhUb,RePoComMents,EventS,TestResults",
			Issues | Prs | Maintainers | Members | Labels | ZenHub | RepoComments | Events | TestResults,
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
