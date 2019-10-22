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
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"istio.io/bots/policybot/pkg/storage"
)

type Time struct {
	time.Time
}

// A company a user is associated with
type Affiliation struct {
	Organization string `json:"organization"`
	Start        Time   `json:"start"`
	End          Time   `json:"end"`
}

// Additional info about an Istio contributor
type UserInfo struct {
	GitHubLogin    string        `json:"github_login"`
	Affiliations   []Affiliation `json:"affiliations"`
	EmailAddresses []string      `json:"email_addresses,omitempty"`
}

type UserdataMgr struct {
	Users []*UserInfo `json:"users"`
}

func Load(file string) (UserdataMgr, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return UserdataMgr{}, fmt.Errorf("unable to read user data file %s: %v", file, err)
	}

	var um UserdataMgr
	if err = yaml.Unmarshal(b, &um); err != nil {
		return UserdataMgr{}, fmt.Errorf("unable to parse user data file %s: %v", file, err)
	}

	return um, nil
}

func (um UserdataMgr) Store(store storage.Store) error {
	var a []*storage.UserAffiliation

	for _, user := range um.Users {
		u, err := store.ReadUser(context.Background(), user.GitHubLogin)
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

	return store.WriteAllUserAffiliations(context.Background(), a)
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Time.Format("2006-01-02") + `"`), nil
}

func (t *Time) UnmarshalJSON(buf []byte) error {
	s := strings.Trim(string(buf), `"`)

	if len(s) == 0 || s == "null" {
		t.Time, _ = time.Parse("2006-01-02", "9999-01-01")
		return nil
	}

	tt, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}
