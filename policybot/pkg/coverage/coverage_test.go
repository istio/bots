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
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/google/go-github/v26/github"
	"golang.org/x/tools/cover"

	"istio.io/bots/policybot/pkg/storage"
)

type byTestName []*storage.CoverageData

func (b byTestName) Len() int {
	return len(b)
}

func (b byTestName) Less(i, j int) bool {
	return b[i].TestName < b[j].TestName
}

func (b byTestName) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func TestClientGetCoverageDataFromProfiles(t *testing.T) {
	unitProfs, err := cover.ParseProfiles(filepath.Join("testdata", "small.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	unitProfs2, err := cover.ParseProfiles(filepath.Join("testdata", "small_covered.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	e2eProfs, err := cover.ParseProfiles(filepath.Join("testdata", "small_covered.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	now := time.Now()
	covMap := map[*storage.TestResult]profiles{
		{TestName: "unittests", FinishTime: now}:  indexProfiles(unitProfs),
		{TestName: "unittests2", FinishTime: now}: indexProfiles(unitProfs2),
		{TestName: "e2e-test", FinishTime: now}:   indexProfiles(e2eProfs),
	}
	label := "bots:master"
	pr := &github.PullRequest{
		Base: &github.PullRequestBranch{
			Label: &label,
		},
	}
	c := Client{
		OrgLogin: "istio",
		Repo:     "bots",
	}
	data, err := c.getCoverageDataFromProfiles("sha", covMap, pr)
	if err != nil {
		t.Errorf("error generating coverage data: %v", err)
		return
	}
	sort.Sort(byTestName(data))
	expected := []*storage.CoverageData{
		{
			OrgLogin:     "istio",
			RepoName:     "bots",
			BranchName:   "master",
			PackageName:  "istio.io/bots/policybot/pkg/storage/spanner",
			Sha:          "sha",
			TestName:     "e2e",
			Type:         "e2e",
			CompletedAt:  now,
			StmtsCovered: 3,
			StmtsTotal:   3,
		},
		{
			OrgLogin:     "istio",
			RepoName:     "bots",
			BranchName:   "master",
			PackageName:  "istio.io/bots/policybot/pkg/storage/spanner",
			Sha:          "sha",
			TestName:     "e2e-test",
			Type:         "e2e",
			CompletedAt:  now,
			StmtsCovered: 3,
			StmtsTotal:   3,
		},
		{
			OrgLogin:     "istio",
			RepoName:     "bots",
			BranchName:   "master",
			PackageName:  "istio.io/bots/policybot/pkg/storage/spanner",
			Sha:          "sha",
			TestName:     "unit",
			Type:         "unit",
			CompletedAt:  now,
			StmtsCovered: 3,
			StmtsTotal:   5,
		},
		{
			OrgLogin:     "istio",
			RepoName:     "bots",
			BranchName:   "master",
			PackageName:  "istio.io/bots/policybot/pkg/storage/spanner",
			Sha:          "sha",
			TestName:     "unittests",
			Type:         "unit",
			CompletedAt:  now,
			StmtsCovered: 0,
			StmtsTotal:   5,
		},
		{
			OrgLogin:     "istio",
			RepoName:     "bots",
			BranchName:   "master",
			PackageName:  "istio.io/bots/policybot/pkg/storage/spanner",
			Sha:          "sha",
			TestName:     "unittests2",
			Type:         "unit",
			CompletedAt:  now,
			StmtsCovered: 3,
			StmtsTotal:   3,
		},
	}
	if !reflect.DeepEqual(expected, data) {
		t.Errorf("unexpected coverage output. got:")
		for _, c := range data {
			t.Errorf("%+v", *c)
		}
		t.Errorf("but expected:")
		for _, c := range expected {
			t.Errorf("%+v", *c)
		}
	}
}

func TestGetCoverageDataFromProfiles(t *testing.T) {
	profs, err := cover.ParseProfiles(filepath.Join("testdata", "test.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	now := time.Now()
	data := getCoverageDataFromProfiles("org", "repo", "master", "sha", "testname", now, indexProfiles(profs))
	expected := storage.CoverageData{
		OrgLogin:     "org",
		RepoName:     "repo",
		BranchName:   "master",
		PackageName:  "istio.io/bots/policybot/pkg/storage/spanner",
		Sha:          "sha",
		TestName:     "testname",
		Type:         unitType,
		CompletedAt:  now,
		StmtsCovered: 94,
		StmtsTotal:   754,
	}
	if expected != *data[0] {
		t.Errorf("expected %+v but got %v", expected, *data[0])
	}
}

const (
	write  = "istio.io/bots/policybot/pkg/storage/spanner/write.go"
	update = "istio.io/bots/policybot/pkg/storage/spanner/update.go"
)

func TestIndexProfiles(t *testing.T) {
	profs, err := cover.ParseProfiles(filepath.Join("testdata", "small.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	indexed := indexProfiles(profs)
	if len(indexed) != 2 {
		t.Errorf("expected 2 entries, but got %d", len(indexed))
	}
	if p, ok := indexed[write]; !ok || p.FileName != write {
		t.Errorf("missing index key %s", write)
	}
	if p, ok := indexed[update]; !ok || p.FileName != update {
		t.Errorf("missing index key %s", update)
	}
}

func TestMergeProfiles(t *testing.T) {
	small, err := cover.ParseProfiles(filepath.Join("testdata", "small.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	covered, err := cover.ParseProfiles(filepath.Join("testdata", "small_covered.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	smallIndex := indexProfiles(small)
	coveredIndex := indexProfiles(covered)
	mergeProfiles(smallIndex, coveredIndex)

	merged, err := cover.ParseProfiles(filepath.Join("testdata", "small_merged.cov"))
	if err != nil {
		t.Errorf("error loading test coverage data: %v", err)
		return
	}
	mergedIndex := indexProfiles(merged)
	if !reflect.DeepEqual(smallIndex, mergedIndex) {
		t.Errorf("expected %+v, but got %+v", mergedIndex, smallIndex)
	}
}
