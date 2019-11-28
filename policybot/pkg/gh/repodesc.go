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

package gh

import "strings"

type RepoDesc struct {
	OrgAndRepo string
	OrgLogin   string
	RepoName   string
	Branch     string
}

func NewRepoDesc(orgAndRepo string) RepoDesc {
	splits := strings.Split(orgAndRepo, "/")

	branch := ""
	if len(splits) > 2 {
		branch = splits[2]
	}

	return RepoDesc{
		OrgAndRepo: splits[0] + "/" + splits[1],
		OrgLogin:   splits[0],
		RepoName:   splits[1],
		Branch:     branch,
	}
}

func (rd RepoDesc) String() string {
	return rd.OrgAndRepo
}
