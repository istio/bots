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

package analyzer

import (
	"bytes"
	"net/http"

	"istio.io/bots/policybot/pkg/storage"
)

type Analyzer struct {
	store storage.Store
}

func NewAnalyzer(store storage.Store) *Analyzer {
	return &Analyzer{
		store: store,
	}
}

func (a *Analyzer) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)

	b := &bytes.Buffer{}
	org, _ := a.store.ReadOrgByLogin("istio")
	if org != nil {
		repo, _ := a.store.ReadRepoByName(org.OrgID, "istio")
		if repo != nil {
			b.WriteString("Hello World")
		}
	}

	_, _ = w.Write(b.Bytes())
}
