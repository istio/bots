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
	"fmt"

	"cloud.google.com/go/spanner"

	"istio.io/bots/policybot/pkg/storage"
)

func (s *store) QueryMembersByOrg(orgID string, cb func(*storage.Member) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Members WHERE OrgID = '%s'", orgID)})
	err := iter.Do(func(row *spanner.Row) error {
		member := &storage.Member{}
		if err := row.ToStruct(member); err != nil {
			return err
		}

		if err := cb(member); err != nil {
			iter.Stop()
			return err
		}

		return nil
	})

	return err
}

func (s *store) QueryMaintainersByOrg(orgID string, cb func(*storage.Maintainer) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Maintainers WHERE OrgID = '%s'", orgID)})
	err := iter.Do(func(row *spanner.Row) error {
		maintainer := &storage.Maintainer{}
		if err := row.ToStruct(maintainer); err != nil {
			return err
		}

		if err := cb(maintainer); err != nil {
			iter.Stop()
			return err
		}

		return nil
	})

	return err
}

func (s *store) QueryIssuesByRepo(orgID string, repoID string, cb func(*storage.Issue) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Issues WHERE OrgID = '%s' AND RepoID = '%s';", orgID, repoID)})
	err := iter.Do(func(row *spanner.Row) error {
		issue := &storage.Issue{}
		if err := row.ToStruct(issue); err != nil {
			return err
		}

		if err := cb(issue); err != nil {
			iter.Stop()
			return err
		}

		return nil
	})

	return err
}

func (s *store) QueryTestFlakeByTestName(testName string, cb func(*storage.TestFlake) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM TestFlakes WHERE TestName = '%s'", testName)})
	err := iter.Do(func(row *spanner.Row) error {
		flake := &storage.TestFlake{}
		if err := row.ToStruct(flake); err != nil {
			return err
		}

		if err := cb(flake); err != nil {
			iter.Stop()
			return err
		}

		return nil
	})

	return err
}

func (s *store) QueryTestFlakeByPrNumber(prNum string, cb func(*storage.TestFlake) error) error {
	iter := s.client.Single().Query(s.ctx, spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM TestFlakes WHERE PrNum = '%s'", prNum)})
	err := iter.Do(func(row *spanner.Row) error {
		flake := &storage.TestFlake{}
		if err := row.ToStruct(flake); err != nil {
			return err
		}

		if err := cb(flake); err != nil {
			iter.Stop()
			return err
		}

		return nil
	})

	return err
}
