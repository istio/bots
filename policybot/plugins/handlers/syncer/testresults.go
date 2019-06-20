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

package syncer

import (
	"fmt"

	"k8s.io/test-infra/prow/config"

	"istio.io/bots/policybot/pkg/storage"
)

func (s *Syncer) fetchTestResults(org *storage.Org, repo *storage.Repo) error {
	cfg, err := config.Load("/Users/mtail/go/src/istio.io/test-infra/prow/config.yaml", "/Users/mtail/go/src/istio.io/test-infra/prow/cluster/jobs")
	if err != nil {
		fmt.Printf("Could not load prow config: %v\n", err)
	} else {
		ps := cfg.AllPostsubmits([]string{org.Login + "/" + repo.Name})
		for _, p := range ps {
			if p.DecorationConfig != nil {
				fmt.Printf("  PostSubmit: %s, %s, %+v\n", p.Name, p.Branches, p.DecorationConfig.GCSConfiguration)
			} else {
				fmt.Printf("  PostSubmit: %s, %s, no decor\n", p.Name, p.Branches)
			}
		}
	}

	return err
}
