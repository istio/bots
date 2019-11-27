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

package flakemgr

import (
	"istio.io/bots/policybot/pkg/config"
)

const recordType = "flakenag"

type flakeNagRecord struct {
	config.RecordBase

	// InactiveDays represents the days that a flaky test issue hasn't been updated.
	InactiveDays int

	// CreatedDays determines the bot search range, only issues created within this days ago
	// are considered.
	CreatedDays int

	// Message is the message the bot comments on flaky test issues.
	Message string
}

func init() {
	config.RegisterType(recordType, config.OnePerRepo, func() config.Record {
		return new(flakeNagRecord)
	})
}
