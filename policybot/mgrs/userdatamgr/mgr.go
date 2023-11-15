// Copyright Istio Authors
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

package userdatamgr

import (
	"context"
	"fmt"
	"os"

	"istio.io/bots/policybot/pkg/config"
	"istio.io/bots/policybot/pkg/storage"
	"istio.io/istio/pkg/log"
)

// UserdataMgr populates Spanner with the set of known affiliated users
type UserdataMgr struct {
	store storage.Store
	reg   *config.Registry
}

var scope = log.RegisterScope("userdatamgr", "Populates the bot's store with the set of known affiliated users")

func New(store storage.Store, reg *config.Registry) *UserdataMgr {
	return &UserdataMgr{
		store: store,
		reg:   reg,
	}
}

func (um UserdataMgr) Store(dryRun bool) error {
	r, ok := um.reg.GlobalRecord(recordType)
	if !ok {
		scope.Infof("No user data configuration record found, exiting")
		return nil
	}

	ur := r.(*userdataRecord)

	var a []*storage.UserAffiliation
	for _, user := range ur.Users {
		u, err := um.store.ReadUser(context.Background(), user.GitHubLogin)
		if u == nil || err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "User %s is not known to the PolicyBot infrastructure, skipping\n", user.GitHubLogin)
			continue
		}

		for i, affiliation := range user.Affiliations {
			a = append(a, &storage.UserAffiliation{
				UserLogin:    user.GitHubLogin,
				Organization: affiliation.Organization,
				StartTime:    affiliation.Start.Time,
				EndTime:      affiliation.End.Time,
				Counter:      int64(i),
			})
		}
	}

	if dryRun {
		scope.Infof("Would have written %d affiliated users to storage", len(a))
		return nil
	}

	scope.Infof("Writing %d affiliated users to storage", len(a))

	return um.store.WriteAllUserAffiliations(context.Background(), a)
}
