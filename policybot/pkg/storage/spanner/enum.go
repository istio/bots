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
	"cloud.google.com/go/spanner"

	"istio.io/bots/policybot/pkg/storage"
)

func (s *store) EnumUsers(cb func(*storage.User) bool) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: "SELECT * FROM Users;"})
	err := iter.Do(func(row *spanner.Row) error {
		user := &storage.User{}
		if err := row.ToStruct(user); err != nil {
			return err
		}

		if !cb(user) {
			iter.Stop()
		}

		return nil
	})

	return err
}
