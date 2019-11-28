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

package config

import "time"

const recordType = "core"

type CoreRecord struct {
	RecordBase

	ServerPort int `json:"server_port"`

	// The path to the Google Cloud Spanner database to use
	SpannerDatabase string `json:"spanner_db"`

	// Name to use as sender when sending emails
	EmailFrom string `json:"email_from"`

	// Email address to use as originating address when sending emails
	EmailOriginAddress string `json:"email_origin_address"`

	// The amount of time cache state is kept around before being discarded
	CacheTTL Duration `json:"cache_ttl"`

	// Name of GCP project that holds the GCS test buckets
	GCPProject string `json:"gcp_project"`

	// Time window within which a maintainer is considered active on the project
	MaintainerActivityWindow Duration `json:"maintainer_activity_window"`

	// Time window within which a member is considered active on the project
	MemberActivityWindow Duration `json:"member_activity_window"`

	// Users that are in fact robots
	Robots []string `json:"robots"`

	// Default GitHub org to use in the UI when none is specified
	DefaultOrg string `json:"default_org"`
}

func init() {
	RegisterType("core", GlobalSingleton, func() Record {
		return &CoreRecord{
			ServerPort:               8080,
			CacheTTL:                 Duration(15 * time.Minute),
			MaintainerActivityWindow: Duration(90 * 24 * time.Hour),
			MemberActivityWindow:     Duration(180 * 24 * time.Hour),
		}
	})
}
