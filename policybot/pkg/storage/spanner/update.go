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

package spanner

import (
	"context"

	"google.golang.org/grpc/codes"

	"cloud.google.com/go/spanner"

	"istio.io/bots/policybot/pkg/storage"
)

func (s store) UpdateFlakeCache(ctx context.Context) (int, error) {
	var sql = `INSERT INTO ConfirmedFlakes(PullRequestNumber, TestName, RunNumber, PassingRunNumber, OrgLogin, RepoName, Done)
select failed.PullRequestNumber, failed.TestName, failed.RunNumber, passed.RunNumber as passingRun, failed.OrgLogin, failed.RepoName, failed.Done
from TestResults as failed
JOIN TestResults as passed
ON passed.PullRequestNumber = failed.PullRequestNumber AND
                        passed.RunNumber != failed.RunNumber AND
                        passed.TestName = failed.TestName AND
                        passed.sha = failed.sha AND
                        passed.TestPassed AND 
NOT failed.TestPassed AND
failed.FinishTime > TIMESTAMP(DATE(2010,1,1)) AND
NOT failed.CloneFailed AND
failed.result!='ABORTED' AND
failed.HasArtifacts
LEFT JOIN ConfirmedFlakes ON failed.PullRequestNumber = ConfirmedFlakes.PullRequestNumber AND
failed.RunNumber = ConfirmedFlakes.RunNumber AND
failed.TestName = ConfirmedFlakes.TestName 
WHERE ConfirmedFlakes.PullRequestNumber is null
limit 2800`
	var rowCount int64
	var sum int
	var err, iErr error
	for {
		_, err = s.client.ReadWriteTransaction(ctx, func(ctx2 context.Context, txn *spanner.ReadWriteTransaction) error {
			rowCount, iErr = txn.Update(ctx2, spanner.Statement{SQL: sql})
			return nil
		})
		if iErr != nil {
			err = iErr
		}
		sum += int(rowCount)
		// spanner does not allow transactions of > 20,000 cells, so for large inserts,
		// we must divide 20,000 by the number of columns being inserted, and repeat
		// until we have reached the end of rows to insert.
		if err != nil || rowCount < 2800 {
			break
		}
	}
	return sum, err
}

func (s store) UpdateBotActivity(ctx1 context.Context, orgLogin string, repoName string, cb func(*storage.BotActivity) error) error {
	scope.Debugf("Updating bot activity for repo %s/%s", orgLogin, repoName)

	_, err := s.client.ReadWriteTransaction(ctx1, func(ctx2 context.Context, txn *spanner.ReadWriteTransaction) error {

		var result storage.BotActivity

		row, err := txn.ReadRow(ctx2, botActivityTable, botActivityKey(orgLogin, repoName), botActivityColumns)
		if spanner.ErrCode(err) == codes.NotFound {
			result.OrgLogin = orgLogin
			result.RepoName = repoName
		} else if err != nil {
			return err
		} else if err = rowToStruct(row, &result); err != nil {
			return err
		}

		err = cb(&result)
		if err != nil {
			return err
		}

		mutation, err := insertOrUpdateStruct(botActivityTable, &result)
		if err != nil {
			return err
		}

		// write all the update bot activity
		return txn.BufferWrite([]*spanner.Mutation{mutation})
	})

	return err
}
