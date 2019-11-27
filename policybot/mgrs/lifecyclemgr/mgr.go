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

package lifecyclemgr

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/gh"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/bots/policybot/pkg/storage/cache"
	"istio.io/pkg/log"
)

// LifecycleMgr is responsible for managing the lifecycle of issues and pull requests.
type LifecycleMgr struct {
	gc    *gh.ThrottledClient
	store storage.Store
	cache *cache.Cache
	reg   *config.Registry
}

type stats struct {
	markedStale           int
	closed                int
	markedNeedsTriage     int
	markedNeedsEscalation int
}

var scope = log.RegisterScope("lifecyclemgr", "The issue and pull request lifecycle manager", 0)

const botSignature = "\n\n_Created by the issue and PR lifecycle manager_."

func New(gc *gh.ThrottledClient, store storage.Store, cache *cache.Cache, reg *config.Registry) *LifecycleMgr {
	return &LifecycleMgr{
		gc:    gc,
		store: store,
		cache: cache,
		reg:   reg,
	}
}

func (lm *LifecycleMgr) ManageAll(context context.Context, dryRun bool) error {
	for _, repo := range lm.reg.Repos() {
		r, ok := lm.reg.SingleRecord(RecordType, repo.OrgAndRepo)
		if !ok {
			continue
		}

		lr := r.(*lifecycleRecord)

		var st stats

		var issues []*storage.Issue
		if err := lm.store.QueryOpenIssuesByRepo(context, repo.OrgLogin, repo.OrgLogin, func(issue *storage.Issue) error {
			issues = append(issues, issue)
			return nil
		}); err != nil {
			return err
		}

		for _, issue := range issues {
			err := lm.manageIssue(context, issue, &st, lr, dryRun)
			if err != nil {
				scope.Errorf("%v", err)
			}
		}

		scope.Infof("STATS: repo %s, markedStale %d, closed %d, markedNeedsTriage %d, markedNeedsEscalation %d\n", repo,
			st.markedStale, st.closed, st.markedNeedsEscalation, st.markedNeedsTriage)
	}

	return nil
}

func (lm *LifecycleMgr) ManageIssue(context context.Context, issue *storage.Issue) error {
	r, ok := lm.reg.SingleRecord(RecordType, issue.OrgLogin+"/"+issue.RepoName)
	if !ok {
		return nil
	}

	lr := r.(*lifecycleRecord)
	return lm.manageIssue(context, issue, &stats{}, lr, false)
}

func (lm *LifecycleMgr) manageIssue(context context.Context, issue *storage.Issue, st *stats, lr *lifecycleRecord, dryRun bool) error {
	now := time.Now()

	if now.Sub(issue.CreatedAt) < time.Duration(lr.TriageDelay) {
		// stay quiet if the item is less than the triage delay
		scope.Infof("Issue/PR %d in repo %s/%s is too new, ignoring", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
		return nil
	}

	hasTriageLabel := false
	hasEscalationLabel := false
	hasStaleproofLabel := false
	hasStaleLabel := false
	hasEnhancementLabel := false
	hasCloseLabel := false
	for _, lb := range issue.Labels {
		switch lb {
		case lr.TriageLabel:
			hasTriageLabel = true
		case lr.EscalationLabel:
			hasEscalationLabel = true
		case lr.CantBeStaleLabel:
			hasStaleproofLabel = true
		case lr.StaleLabel:
			hasStaleLabel = true
		case lr.FeatureRequestLabel:
			hasEnhancementLabel = true
		case lr.CloseLabel:
			hasCloseLabel = true
		default:
			for _, il := range lr.IgnoreLabels {
				if il == lb {
					// matched an "ignore" label"
					scope.Infof("Issue/PR %d in repo %s/%s has the '%s' label so it will be ignored", issue.IssueNumber, issue.OrgLogin, issue.RepoName, il)
					return nil
				}
			}
		}
	}

	if issue.State == "closed" {
		scope.Infof("Issue/PR %d in repo %s/%s is closed, doing label cleanup and returning", issue.IssueNumber, issue.OrgLogin, issue.RepoName)

		if hasTriageLabel {
			_ = lm.removeLabel(context, issue, lr.TriageLabel, dryRun)
		}

		if hasEscalationLabel {
			_ = lm.removeLabel(context, issue, lr.EscalationLabel, dryRun)
		}

		return nil
	}

	latestMemberActivity, err := lm.store.GetLatestIssueMemberActivity(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber))
	if err != nil {
		return fmt.Errorf("could not get member activity for issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
	}

	latestMemberComment, err := lm.store.GetLatestIssueMemberComment(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber))
	if err != nil {
		return fmt.Errorf("could not get member comment for issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
	}

	latestMemberCommentDelta := now.Sub(latestMemberComment)
	if (latestMemberComment == time.Time{}) {
		latestMemberCommentDelta = now.Sub(issue.CreatedAt)
	}

	pipeline, err := lm.cache.ReadIssuePipeline(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber))
	if err != nil {
		return fmt.Errorf("could not get issue pipeline data for issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
	}

	pull, err := lm.cache.ReadPullRequest(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber))
	if err != nil {
		return fmt.Errorf("could not get pr info for issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
	}
	pr := pull != nil

	if !pr {
		// if an issue needs to be triaged, add the label
		if needsTriage(pipeline, latestMemberComment, latestMemberActivity, pr) {
			oldCutoff := now.Add(-time.Duration(lr.RealOldDelay))

			if issue.CreatedAt.After(oldCutoff) {
				st.markedNeedsTriage++
				if !hasTriageLabel {
					if err := lm.addLabel(context, issue, lr.TriageLabel, dryRun); err != nil {
						return err
					}
				}

				return nil
			}
		} else if hasTriageLabel {
			// the item has been triaged, so the label can be removed if present
			if err := lm.removeLabel(context, issue, lr.TriageLabel, dryRun); err != nil {
				return err
			}
		}
	}

	// see if the item needs escalation
	if lm.needsEscalation(pipeline, latestMemberCommentDelta, lr) {
		st.markedNeedsEscalation++
		if !hasEscalationLabel {
			if err := lm.addLabel(context, issue, lr.EscalationLabel, dryRun); err != nil {
				return err
			}
		}
	} else if hasEscalationLabel {
		// remove the label if no need for escalation
		if err := lm.removeLabel(context, issue, lr.EscalationLabel, dryRun); err != nil {
			return err
		}
	}

	if hasStaleproofLabel {
		// clean up any leftover stale label and staleness comment
		if hasStaleLabel {
			if err := lm.removeLabel(context, issue, lr.StaleLabel, dryRun); err != nil {
				return err
			}
		}

		// remove staleness comment
		if err := lm.removeComment(context, issue, dryRun); err != nil {
			return err
		}

		return nil
	}

	var staleDelay time.Duration
	var closeDelay time.Duration
	if pr {
		staleDelay = time.Duration(lr.PullRequestStaleDelay)
		closeDelay = time.Duration(lr.PullRequestCloseDelay)
	} else if hasEnhancementLabel {
		staleDelay = time.Duration(lr.FeatureRequestStaleDelay)
		closeDelay = time.Duration(lr.FeatureRequestCloseDelay)
	} else {
		staleDelay = time.Duration(lr.IssueStaleDelay)
		closeDelay = time.Duration(lr.IssueCloseDelay)
	}

	from := latestMemberComment
	if from == (time.Time{}) {
		from = issue.CreatedAt
	}

	if latestMemberCommentDelta > closeDelay {
		st.closed++

		// close the issue
		if err := lm.closeIssue(context, issue, dryRun); err != nil {
			return err
		}

		// add closing comment
		commentDate := from.Format("2006-01-02")
		if err := lm.addComment(context, issue, fmt.Sprintf(lr.CloseComment, commentDate), "closing", dryRun); err != nil {
			return err
		}

		// add closing label
		if !hasCloseLabel {
			if err := lm.addLabel(context, issue, lr.CloseLabel, dryRun); err != nil {
				return err
			}
		}
	} else if latestMemberCommentDelta > staleDelay {
		commentDate := from.Format("2006-01-02")
		closeDate := from.Add(closeDelay).Format("2006-01-02")

		st.markedStale++

		// add staleness comment
		if err := lm.addComment(context, issue, fmt.Sprintf(lr.StaleComment, commentDate, closeDate), "staleness", dryRun); err != nil {
			return err
		}

		// add stale label
		if err := lm.addLabel(context, issue, lr.StaleLabel, dryRun); err != nil {
			return err
		}
	} else {
		// remove stale label
		if hasStaleLabel {
			if err := lm.removeLabel(context, issue, lr.StaleLabel, dryRun); err != nil {
				return err
			}
		}

		// remove staleness comment
		if err := lm.removeComment(context, issue, dryRun); err != nil {
			return err
		}
	}

	return nil
}

func (lm *LifecycleMgr) closeIssue(context context.Context, issue *storage.Issue, dryRun bool) error {
	if dryRun {
		scope.Infof("Would have closed issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
		return nil
	}

	if _, _, err := lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		closed := "closed"
		return client.Issues.Edit(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), &github.IssueRequest{
			State: &closed,
		})
	}); err != nil {
		return fmt.Errorf("unable to close issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
	}

	scope.Infof("Closed issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	return nil
}

func (lm *LifecycleMgr) addLabel(context context.Context, issue *storage.Issue, label string, dryRun bool) error {
	if label == "" {
		return nil
	}

	if dryRun {
		scope.Infof("Would have added the `%s` label to issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
		return nil
	}

	if _, _, err := lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
		return client.Issues.AddLabelsToIssue(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), []string{label})
	}); err != nil {
		return fmt.Errorf("unable to add the `%s` label on issue/PR %d in repo %s/%s: %v", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
	}

	scope.Infof("Added the `%s` label to issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	return nil
}

func (lm *LifecycleMgr) removeLabel(context context.Context, issue *storage.Issue, label string, dryRun bool) error {
	if label == "" {
		return nil
	}

	if dryRun {
		scope.Infof("Would have added the `%s` label to issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
		return nil
	}

	if _, err := lm.gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
		return client.Issues.RemoveLabelForIssue(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), label)
	}); err != nil {
		return fmt.Errorf("unable to remove the `%s` label from issue/PR %d in repo %s/%s: %v", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
	}

	scope.Infof("Removed the `%s` label from issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	return nil
}

func (lm *LifecycleMgr) addComment(context context.Context, issue *storage.Issue, comment string, kind string, dryRun bool) error {
	if dryRun {
		scope.Infof("Would have added %s comment to issue/PR %d in repo %s/%s", kind, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
		return nil
	}

	var err error
	if comment != "" {
		err = lm.gc.AddOrReplaceBotComment(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), comment, botSignature)
	} else {
		err = lm.gc.RemoveBotComment(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), botSignature)
	}

	if err != nil {
		return err
	}

	scope.Infof("Added %s comment to issue/PR %d in repo %s/%s", kind, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	return nil
}

func (lm *LifecycleMgr) removeComment(context context.Context, issue *storage.Issue, dryRun bool) error {
	if dryRun {
		scope.Infof("Would have removed comment from issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
		return nil
	}

	err := lm.gc.RemoveBotComment(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), botSignature)
	if err != nil {
		return err
	}

	scope.Infof("Removed comment from issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	return nil
}

func needsTriage(pipeline *storage.IssuePipeline, latestMemberComment time.Time, latestMemberActivity time.Time, pr bool) bool {
	if !pr {
		// issues need to be prioritized
		if pipeline == nil || pipeline.Pipeline == "" || pipeline.Pipeline == "New Issues" {
			return true
		}
	}

	if (latestMemberComment == time.Time{}) && (latestMemberActivity == time.Time{}) {
		// no team member has left a comment or performed any other activity on the issue
		return true
	}

	return false
}

func (lm *LifecycleMgr) needsEscalation(pipeline *storage.IssuePipeline, latestMemberCommentDelta time.Duration, lr *lifecycleRecord) bool {
	if highPriority(pipeline) {
		// needs escalation if no team member has commented on the item in the allowed escalation delay
		if latestMemberCommentDelta > time.Duration(lr.EscalationDelay) {
			return true
		}
	}

	return false
}

func highPriority(pipeline *storage.IssuePipeline) bool {
	return pipeline != nil && (pipeline.Pipeline == "P0" || pipeline.Pipeline == "Release Blocker")
}
