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

// Take in a pr number from blob storage and examines the pr
// for all tests that are run and their results. The results are then written to storage.

package coverage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v26/github"
	"golang.org/x/tools/cover"

	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/pkg/log"
)

var scope = log.RegisterScope("coverage", "Coverage client", 0)

var (
	pending    = "pending"
	statusName = "istio-testing/coverage"
)

type profiles map[string]*cover.Profile

const (
	e2eType   = "e2e"
	integType = "integ"
	unitType  = "unit"
)

// Client handles all aspects of gathering and reporting code coverage.
type Client struct {
	OrgLogin, Repo string
	Bucket         string
	BlobClient     blobstorage.Store
	StorageClient  storage.Store
	GithubClient   *gh.ThrottledClient
}

func logTime(sha string, start time.Time) {
	scope.Infof("coverage check for %s completed in %s", sha, time.Since(start))
}

// CheckCoverage checks to see if all pending tests have completed and compiles
// coverage information if so.
func (c *Client) CheckCoverage(ctx context.Context, pr *github.PullRequest, sha string) error {
	defer logTime(sha, time.Now())
	resp, _, err := c.GithubClient.ThrottledCall(
		func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Repositories.GetCombinedStatus(ctx, c.OrgLogin, c.Repo, sha, nil)
		})
	if err != nil {
		return fmt.Errorf("coverage: error looking up statuses for %s/%s commit %s: %v",
			c.OrgLogin, c.Repo, sha, err)
	}
	statuses := resp.(*github.CombinedStatus)
	hasCoverageStatus := false
	for _, status := range statuses.Statuses {
		if status.GetContext() == statusName {
			hasCoverageStatus = true
		} else if status.GetState() == "pending" {
			scope.Infof("skipping coverage check for %s/%s commit %s, which has pending statuses",
				c.OrgLogin, c.Repo, sha)
			return nil
		}
	}
	if !hasCoverageStatus {
		scope.Infof("skipping coverage check for %s/%s commit %s, which has no coverage status",
			c.OrgLogin, c.Repo, sha)
		return nil
	}

	// Get all test results for the commit from the database.
	var coverageResults []*storage.TestResult
	err = c.StorageClient.QueryTestResultsBySHA(
		ctx, c.OrgLogin, c.Repo, sha,
		func(result *storage.TestResult) error {
			if hasCoverage(result) {
				coverageResults = append(coverageResults, result)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("coverage: error fetching test results for %s/%s commit %s: %v",
			c.OrgLogin, c.Repo, sha, err)
	}

	// Download, parse, and synthesize all coverage files. Unfortunately, we
	// have to download to temp files to process them with Go's coverage
	// package.
	b := c.BlobClient.Bucket(c.Bucket)
	tmpDir, err := ioutil.TempDir("", "coverage")
	if err != nil {
		return fmt.Errorf("coverage: error creating temp dir for coverage files")
	}
	defer os.RemoveAll(tmpDir)
	covMap := make(map[*storage.TestResult]profiles)
	for _, result := range coverageResults {
		covMap[result], err = c.getProfilesForTestResult(ctx, result, b, tmpDir)
		if err != nil {
			return err
		}
	}

	covData, err := c.getCoverageDataFromProfiles(sha, covMap, pr)
	if err != nil {
		return fmt.Errorf("coverage: error generating coverage data from profiles: %v", err)
	}
	err = c.StorageClient.WriteCoverageData(ctx, covData)
	if err != nil {
		return fmt.Errorf("coverage: error writing coverage data to storage: %v", err)
	}
	return nil
}

func (c *Client) getProfilesForTestResult(
	ctx context.Context,
	r *storage.TestResult,
	b blobstorage.Bucket,
	tmpDir string,
) (profiles, error) {
	merged := make(profiles)
	for _, artifact := range r.Artifacts {
		if isCoverageArtifact(artifact) {
			f, err := ioutil.TempFile(tmpDir, "coverage-*.cov")
			if err != nil {
				return nil, fmt.Errorf("coverage: error creating coverage temp file: %v", err)
			}
			r, err := b.Reader(ctx, artifact)
			if err != nil {
				return nil, fmt.Errorf("coverage: error reading coverage file %s/%s from blob storage: %v",
					c.Bucket, artifact, err)
			}
			_, err = io.Copy(f, r)
			if err != nil {
				return nil, fmt.Errorf("coverage: error writing coverage file %s/%s: %v",
					c.Bucket, artifact, err)
			}
			profs, err := cover.ParseProfiles(f.Name())
			if err != nil {
				return nil, fmt.Errorf("coverage: error parsing coverage file for %s/%s: %v",
					c.Bucket, artifact, err)
			}
			err = mergeProfiles(merged, indexProfiles(profs))
			if err != nil {
				return nil, fmt.Errorf("coverage: error merging coverage profiles for %s/%s: %v",
					c.Bucket, artifact, err)
			}
		}
	}
	return merged, nil
}

type aggregate struct {
	p profiles
	c time.Time
}

func (c *Client) getCoverageDataFromProfiles(
	sha string,
	covMap map[*storage.TestResult]profiles,
	pr *github.PullRequest,
) ([]*storage.CoverageData, error) {
	var covs []*storage.CoverageData
	testTypeCov := map[string]*aggregate{
		e2eType:   {make(profiles), time.Time{}},
		integType: {make(profiles), time.Time{}},
		unitType:  {make(profiles), time.Time{}},
	}
	// In theory, we can probably just use GetRef instead of operating over GetLabel, but
	// there really is not documentation explaining the difference between the second part
	// of the label and the ref. The best I can tell is that perhaps the PR can be based
	// on an older ref in the branch, which would cause the ref to be a SHA instead of
	// a branch name.
	label := pr.GetBase().GetLabel()
	branch := label[strings.Index(label, ":")+1:]
	// Create a merged profile for each test type and generate CoverageData entries for each
	// individual test.
	for r, profs := range covMap {
		info := testTypeCov[getTestType(r.TestName)]
		if err := mergeProfiles(info.p, profs); err != nil {
			return nil, err
		}
		// Use the last finish time for a given test type as the aggregate finish time.
		if r.FinishTime.After(info.c) {
			info.c = r.FinishTime
		}
		covs = append(covs,
			getCoverageDataFromProfiles(c.OrgLogin, c.Repo, branch, sha, r.TestName, r.FinishTime, profs)...)
	}
	// Now create CoverageData entries representing each test type aggregation.
	for testType, info := range testTypeCov {
		covs = append(covs,
			getCoverageDataFromProfiles(c.OrgLogin, c.Repo, branch, sha, testType, info.c, info.p)...)
	}
	return covs, nil
}

func getCoverageDataFromProfiles(
	org, repo, branch, sha, test string,
	completedAt time.Time,
	profs profiles,
) []*storage.CoverageData {
	packageCov := make(map[string]*storage.CoverageData)
	for name, p := range profs {
		pkg := name[0:strings.LastIndex(name, "/")]
		cov, ok := packageCov[pkg]
		if !ok {
			cov = &storage.CoverageData{
				OrgLogin:    org,
				RepoName:    repo,
				BranchName:  branch,
				PackageName: pkg,
				Sha:         sha,
				TestName:    test,
				Type:        getTestType(test),
				CompletedAt: completedAt,
			}
			packageCov[pkg] = cov
		}
		for _, block := range p.Blocks {
			cov.StmtsTotal += int64(block.NumStmt)
			if block.Count > 0 {
				cov.StmtsCovered += int64(block.NumStmt)
			}
		}
	}
	covs := make([]*storage.CoverageData, 0, len(packageCov))
	for _, cov := range packageCov {
		covs = append(covs, cov)
	}
	return covs
}

// SetPendingCoverage sets a pending coverage status for a given commit.
func (c *Client) SetPendingCoverage(ctx context.Context, sha string) {
	_, _, err := c.GithubClient.ThrottledCall(
		func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Repositories.CreateStatus(ctx, c.OrgLogin, c.Repo, sha, &github.RepoStatus{
				State:   &pending,
				Context: &statusName,
			})
		})
	if err != nil {
		scope.Errorf("Failed to set pending coverage status on %s/%s for commit %s: %v",
			c.OrgLogin, c.Repo, sha, err)
	}
}

func hasCoverage(r *storage.TestResult) bool {
	for _, artifact := range r.Artifacts {
		if isCoverageArtifact(artifact) {
			return true
		}
	}
	return false
}

func isCoverageArtifact(artifact string) bool {
	return strings.HasSuffix(artifact, ".cov")
}

func getTestType(name string) string {
	if strings.HasPrefix(name, "e2e") {
		return e2eType
	} else if strings.HasPrefix(name, "integ") {
		return integType
	}
	return unitType
}

func indexProfiles(profs []*cover.Profile) profiles {
	m := make(profiles)
	for _, p := range profs {
		m[p.FileName] = p
	}
	return m
}

func mergeProfiles(a, b profiles) error {
	if len(b) == 0 {
		return nil
	}
	for _, p := range a {
		if q, ok := b[p.FileName]; ok {
			if err := mergeProfile(p, q); err != nil {
				return err
			}
		}
	}
	for _, p := range b {
		if _, ok := a[p.FileName]; !ok {
			a[p.FileName] = p
		}
	}
	return nil
}

func mergeProfile(p, q *cover.Profile) error {
	if len(p.Blocks) != len(q.Blocks) {
		return errors.New("profiles have different block lengths")
	}
	for i := 0; i < len(p.Blocks); i++ {
		bp := &p.Blocks[i]
		bq := q.Blocks[i]
		if bp.StartLine != bq.StartLine ||
			bp.StartCol != bq.StartCol ||
			bp.EndLine != bq.EndLine ||
			bp.EndCol != bq.EndCol {
			return fmt.Errorf("blocks have different boundaries: %+v %+v", bp, bq)
		}
		bp.Count += bq.Count
	}
	return nil
}
