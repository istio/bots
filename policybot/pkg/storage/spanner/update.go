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

	"cloud.google.com/go/spanner"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/grpc/codes"

	"istio.io/bots/policybot/pkg/storage"
)

// running this in a Read/Write transaction causes timeouts, so we were advised to separate read and write
func (s store) UpdateFlakeCache(ctx context.Context) (sum int, multierr error) {
	errchan := s.QueryNewFlakes(ctx).OnError(func(err error) {
		scope.Warnf("Error parsing flake: %v", err)
	}).Batch(2800).To(func(input interface{}) (err error) {
		islice := input.([]interface{})
		mutations := make([]*spanner.Mutation, len(islice))
		for x, i := range islice {
			singleResult := i.(*storage.ConfirmedFlake)
			if mutations[x], err = insertOrUpdateStruct(confirmedFlakesTable, singleResult); err != nil {
				return
			}
		}

		if _, err = s.client.Apply(ctx, mutations); err == nil {
			sum += len(mutations)
		}
		return
	}).Go()
	var result *multierror.Error
	for err := range errchan {
		result = multierror.Append(result, err.Err())
	}
	if result != nil {
		return
	}
	return
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
