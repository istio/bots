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

type RecordBase struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	Repos []string `json:"repos"`
}

func (rb RecordBase) GetRepos() []string {
	return rb.Repos
}

func (rb RecordBase) GetName() string {
	return rb.Name
}

func (rb RecordBase) GetType() string {
	return rb.Type
}
