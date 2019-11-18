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
	gc     *gh.ThrottledClient
	store  storage.Store
	cache  *cache.Cache
	orgs   []config.Org
	config *config.Lifecycle
	dryrun bool
}

type Stats struct {
	markedStale           int
	closed                int
	markedNeedsTriage     int
	markedNeedsEscalation int
}

var scope = log.RegisterScope("lifecyclemgr", "The issue and pull request lifecycle manager", 0)

const botSignature = "\n\n_Created by the issue and PR lifecycle manager_."

func New(gc *gh.ThrottledClient, store storage.Store, cache *cache.Cache, a *config.Args) *LifecycleMgr {
	return &LifecycleMgr{
		gc:     gc,
		store:  store,
		cache:  cache,
		orgs:   a.Orgs,
		config: &a.Lifecycle,
		dryrun: false,
	}
}

func (lm *LifecycleMgr) ManageAll(context context.Context) error {
	for _, o := range lm.orgs {
		for _, r := range o.Repos {
			var st Stats

			var issues []*storage.Issue
			if err := lm.store.QueryOpenIssuesByRepo(context, o.Name, r.Name, func(issue *storage.Issue) error {
				issues = append(issues, issue)
				return nil
			}); err != nil {
				return err
			}

			for _, issue := range issues {
				err := lm.manageIssue(context, issue, &st)
				if err != nil {
					scope.Errorf("%v", err)
				}
			}

			scope.Infof("STATS: repo %s, markedStale %d, closed %d, markedNeedsTriage %d, markedNeedsEscalation %d\n", r.Name,
				st.markedStale, st.closed, st.markedNeedsEscalation, st.markedNeedsTriage)
		}
	}

	return nil
}

func (lm *LifecycleMgr) ManageIssue(context context.Context, issue *storage.Issue) error {
	return lm.manageIssue(context, issue, &Stats{})
}

func (lm *LifecycleMgr) manageIssue(context context.Context, issue *storage.Issue, st *Stats) error {
	now := time.Now()

	if now.Sub(issue.CreatedAt) < time.Duration(lm.config.TriageDelay) {
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
		case lm.config.TriageLabel:
			hasTriageLabel = true
		case lm.config.EscalationLabel:
			hasEscalationLabel = true
		case lm.config.CantBeStaleLabel:
			hasStaleproofLabel = true
		case lm.config.StaleLabel:
			hasStaleLabel = true
		case lm.config.FeatureRequestLabel:
			hasEnhancementLabel = true
		case lm.config.CloseLabel:
			hasCloseLabel = true
		default:
			for _, il := range lm.config.IgnoreLabels {
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
			_ = lm.removeLabel(context, issue, lm.config.TriageLabel)
		}

		if hasEscalationLabel {
			_ = lm.removeLabel(context, issue, lm.config.EscalationLabel)
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
			oldCutoff := now.Add(-time.Duration(lm.config.RealOldDelay))

			if issue.CreatedAt.After(oldCutoff) {
				st.markedNeedsTriage++
				if !hasTriageLabel {
					if err := lm.addLabel(context, issue, lm.config.TriageLabel); err != nil {
						return err
					}
				}

				return nil
			}
		} else if hasTriageLabel {
			// the item has been triaged, so the label can be removed if present
			if err := lm.removeLabel(context, issue, lm.config.TriageLabel); err != nil {
				return err
			}
		}
	}

	// see if the item needs escalation
	if lm.needsEscalation(pipeline, latestMemberCommentDelta) {
		st.markedNeedsEscalation++
		if !hasEscalationLabel {
			if err := lm.addLabel(context, issue, lm.config.EscalationLabel); err != nil {
				return err
			}
		}
	} else if hasEscalationLabel {
		// remove the label if no need for escalation
		if err := lm.removeLabel(context, issue, lm.config.EscalationLabel); err != nil {
			return err
		}
	}

	if hasStaleproofLabel {
		// clean up any leftover stale label and staleness comment
		if hasStaleLabel {
			if err := lm.removeLabel(context, issue, lm.config.StaleLabel); err != nil {
				return err
			}
		}

		// remove staleness comment
		if err := lm.removeComment(context, issue); err != nil {
			return err
		}

		return nil
	}

	var staleDelay time.Duration
	var closeDelay time.Duration
	if pr {
		staleDelay = time.Duration(lm.config.PullRequestStaleDelay)
		closeDelay = time.Duration(lm.config.PullRequestCloseDelay)
	} else if hasEnhancementLabel {
		staleDelay = time.Duration(lm.config.FeatureRequestStaleDelay)
		closeDelay = time.Duration(lm.config.FeatureRequestCloseDelay)
	} else {
		staleDelay = time.Duration(lm.config.IssueStaleDelay)
		closeDelay = time.Duration(lm.config.IssueCloseDelay)
	}

	if latestMemberCommentDelta > closeDelay {
		st.closed++

		// close the issue
		if err := lm.closeIssue(context, issue); err != nil {
			return err
		}

		// add closing comment
		commentDate := latestMemberComment.Format("2006-01-02")
		if err := lm.addComment(context, issue, fmt.Sprintf(lm.config.CloseComment, commentDate), "closing"); err != nil {
			return err
		}

		// add closing label
		if !hasCloseLabel {
			if err := lm.addLabel(context, issue, lm.config.CloseLabel); err != nil {
				return err
			}
		}
	} else if latestMemberCommentDelta > staleDelay {
		commentDate := latestMemberComment.Format("2006-01-02")
		closeDate := latestMemberComment.Add(closeDelay).Format("2006-01-02")

		st.markedStale++

		// add staleness comment
		if err := lm.addComment(context, issue, fmt.Sprintf(lm.config.StaleComment, commentDate, closeDate), "staleness"); err != nil {
			return err
		}

		// add stale label
		if err := lm.addLabel(context, issue, lm.config.StaleLabel); err != nil {
			return err
		}
	} else {
		// remove stale label
		if hasStaleLabel {
			if err := lm.removeLabel(context, issue, lm.config.StaleLabel); err != nil {
				return err
			}
		}

		// remove staleness comment
		if err := lm.removeComment(context, issue); err != nil {
			return err
		}
	}

	return nil
}

func (lm *LifecycleMgr) closeIssue(context context.Context, issue *storage.Issue) error {
	if !lm.dryrun {
		if _, _, err := lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			closed := "closed"
			return client.Issues.Edit(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), &github.IssueRequest{
				State: &closed,
			})
		}); err != nil {
			return fmt.Errorf("unable to close issue/PR %d in repo %s/%s: %v", issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
		}

		scope.Infof("Closed issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	} else {
		scope.Infof("Would have closed issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	}

	return nil
}

func (lm *LifecycleMgr) addLabel(context context.Context, issue *storage.Issue, label string) error {
	if label == "" {
		return nil
	}

	if !lm.dryrun {
		if _, _, err := lm.gc.ThrottledCall(func(client *github.Client) (interface{}, *github.Response, error) {
			return client.Issues.AddLabelsToIssue(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), []string{label})
		}); err != nil {
			return fmt.Errorf("unable to set the `%s` label on issue/PR %d in repo %s/%s: %v", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
		}

		scope.Infof("Added the `%s` label to issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	} else {
		scope.Infof("Would have added the `%s` label to issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	}

	return nil
}

func (lm *LifecycleMgr) removeLabel(context context.Context, issue *storage.Issue, label string) error {
	if label == "" {
		return nil
	}

	if !lm.dryrun {
		if _, err := lm.gc.ThrottledCallNoResult(func(client *github.Client) (*github.Response, error) {
			return client.Issues.RemoveLabelForIssue(context, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), label)
		}); err != nil {
			return fmt.Errorf("unable to remove the `%s` label from issue/PR %d in repo %s/%s: %v", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName, err)
		}

		scope.Infof("Removed the `%s` label from issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	} else {
		scope.Infof("Would have removed the `%s` label from issue/PR %d in repo %s/%s", label, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	}

	return nil
}

func (lm *LifecycleMgr) addComment(context context.Context, issue *storage.Issue, comment string, kind string) error {
	if !lm.dryrun {
		var err error
		if comment != "" {
			err = gh.AddOrReplaceBotComment(context, lm.gc, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), comment, botSignature)
		} else {
			err = gh.RemoveBotComment(context, lm.gc, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), botSignature)
		}

		if err != nil {
			return err
		}

		scope.Infof("Added %s comment to issue/PR %d in repo %s/%s", kind, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	} else {
		scope.Infof("Would have added %s comment to issue/PR %d in repo %s/%s", kind, issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	}

	return nil
}

func (lm *LifecycleMgr) removeComment(context context.Context, issue *storage.Issue) error {
	if !lm.dryrun {
		err := gh.RemoveBotComment(context, lm.gc, issue.OrgLogin, issue.RepoName, int(issue.IssueNumber), botSignature)
		if err != nil {
			return err
		}

		scope.Infof("Removed comment from issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	} else {
		scope.Infof("Would have removed comment from issue/PR %d in repo %s/%s", issue.IssueNumber, issue.OrgLogin, issue.RepoName)
	}

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

func (lm *LifecycleMgr) needsEscalation(pipeline *storage.IssuePipeline, latestMemberCommentDelta time.Duration) bool {
	if highPriority(pipeline) {
		// needs escalation if no team member has commented on the item in the allowed escalation delay
		if latestMemberCommentDelta > time.Duration(lm.config.EscalationDelay) {
			return true
		}
	}

	return false
}

func highPriority(pipeline *storage.IssuePipeline) bool {
	return pipeline != nil && (pipeline.Pipeline == "P0" || pipeline.Pipeline == "Release Blocker")
}
