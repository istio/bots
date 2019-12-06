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

package welcomer

import (
	"istio.io/bots/policybot/pkg/config"
)

const recordType = "welcome"

type welcomeRecord struct {
	config.RecordBase

	// The message to inject as a welcome message for new contributors to a repo.
	Message string

	// The message is posted if the user has never contributed to the repo, or if the last contribution
	// is older than the resend interval
	ResendDays int
}

func init() {
	config.RegisterType(recordType, config.OnePerRepo, func() config.Record {
		return new(welcomeRecord)
	})
}
