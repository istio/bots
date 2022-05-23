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
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/storage"
)

type cov struct {
	covered, total int64
}

type DiffResultEntry struct {
	Feature, Stage, Label string
	Target, Actual, Base  float64
}

type DiffResult struct {
	err     error
	Entries []*DiffResultEntry
}

// GetGithubStatus returns the Github status string for the DiffResult.
func (d DiffResult) GetGithubStatus() string {
	if d.err != nil {
		return Error
	}
	if len(d.Entries) > 0 {
		return Failure
	}
	return Success
}

// GetDescription returns the one line description string for the DiffResult.
func (d DiffResult) GetDescription() string {
	s := d.GetGithubStatus()
	if s == Success {
		return "All coverage checks passed."
	}
	if s == Failure {
		return "Some coverage targets were not met."
	}
	return "Error while computing code coverage."
}

// GetComment returns a comment for the bot to post on a PR thread, if any.
// A blank string means no comment should be posted.
func (d DiffResult) GetComment() string {
	s := d.GetGithubStatus()
	if s == Success {
		return ""
	}
	if s == Failure {
		b := strings.Builder{}
		b.WriteString("Coverage checks failed:\n\n")
		for _, entry := range d.Entries {
			b.WriteString(
				fmt.Sprintf("* [%s.%s.%s]: Coverage for this PR is %f%%, which does not meet the coverage target of %f%% (base PR has %f%% coverage)\n",
					entry.Feature, entry.Stage, entry.Label, entry.Actual, entry.Target, entry.Base))
		}
		return b.String()
	}
	return "An internal error occurred while computing coverage. Please file an issue for investigation."
}

func (c *Client) checkCoverageDiff(
	ctx context.Context,
	pr *github.PullRequest,
	sha string,
) *DiffResult {
	cfg, err := GetConfig(c.OrgLogin, c.Repo)
	if err != nil {
		return &DiffResult{err: err}
	}
	if len(cfg) == 0 {
		return &DiffResult{}
	}
	baseSHA := pr.GetBase().GetSHA()
	baseCoverage, err := c.getCoverageData(ctx, baseSHA)
	if err != nil {
		return &DiffResult{err: err}
	}
	coverage, err := c.getCoverageData(ctx, sha)
	if err != nil {
		return &DiffResult{err: err}
	}

	return computeDiffResult(cfg, baseCoverage, coverage)
}

func (c *Client) getCoverageData(ctx context.Context, sha string) (map[string][]*storage.CoverageData, error) {
	var data map[string][]*storage.CoverageData
	err := c.StorageClient.QueryCoverageDataBySHA(
		ctx, c.OrgLogin, c.Repo, sha,
		func(result *storage.CoverageData) error {
			data[result.PackageName] = append(data[result.PackageName], result)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func updateCov(covs map[string]*cov, profs []*storage.CoverageData) {
	for _, prof := range profs {
		if _, ok := covs[prof.Type]; !ok {
			covs[prof.Type] = &cov{}
		}
		covs[prof.Type].covered += prof.StmtsCovered
		covs[prof.Type].total += prof.StmtsTotal
	}
}

func computeDiffResult(
	cfg Config,
	baseCov, headCov map[string][]*storage.CoverageData,
) *DiffResult {
	result := &DiffResult{}
	for featureName, feature := range cfg {
		for stageName, stage := range feature.Stages {
			base := make(map[string]*cov)
			curr := make(map[string]*cov)
			for _, stagePkg := range stage.Packages {
				// Perhaps we can do better than this.
				for pkg, covs := range baseCov {
					if strings.HasPrefix(pkg, stagePkg) {
						updateCov(base, covs)
					}
				}
				for pkg, covs := range headCov {
					if strings.HasPrefix(pkg, stagePkg) {
						updateCov(curr, covs)
					}
				}
			}
			for label, target := range stage.Targets {
				pct := 0.0
				if cov, ok := curr[label]; ok {
					pct = float64(cov.covered) * 100 / float64(cov.total)
				}
				basePct := 0.0
				if baseCov, ok := base[label]; ok {
					basePct = float64(baseCov.covered) * 100 / float64(baseCov.total)
				}
				if pct < target {
					result.Entries = append(result.Entries, &DiffResultEntry{
						Feature: featureName,
						Stage:   stageName,
						Label:   label,
						Target:  target,
						Actual:  pct,
						Base:    basePct,
					})
				}
			}
		}
	}
	return result
}
