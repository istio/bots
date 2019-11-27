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

package userdatamgr

import (
	"strings"
	"time"

	"istio.io/bots/policybot/pkg/config"
)

const recordType = "userdata"

type Time struct {
	time.Time
}

// A company a user is associated with
type affiliation struct {
	Organization string `json:"organization"`
	Start        Time   `json:"start"`
	End          Time   `json:"end"`
}

// Additional info about an Istio contributor
type userInfo struct {
	GitHubLogin    string        `json:"github_login"`
	Affiliations   []affiliation `json:"affiliations"`
	EmailAddresses []string      `json:"email_addresses,omitempty"`
}

type userdataRecord struct {
	config.RecordBase
	Users []*userInfo `json:"users"`
}

func init() {
	config.RegisterType(recordType, config.GlobalSingleton, func() config.Record {
		return new(userdataRecord)
	})
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
