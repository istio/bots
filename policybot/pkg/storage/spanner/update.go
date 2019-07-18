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
