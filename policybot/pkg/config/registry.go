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

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/google/go-github/v26/github"

	"istio.io/bots/policybot/pkg/gh"
)

type Record interface {
	GetRepos() []string
}

type RecordCardinality int

const (
	GlobalSingleton RecordCardinality = iota
	OnePerRepo
	MultiplePerRepo
)

type RecordFactory func() Record

type recordTypeInfo struct {
	factory RecordFactory
	card    RecordCardinality
}

var recordTypes = make(map[string]recordTypeInfo)

type recordSet map[string][]Record

type Registry struct {
	records       recordSet
	repos         map[string]recordSet
	globalRecords map[string]Record
	allRepos      []gh.RepoDesc
	core          *CoreRecord
	originRepo    gh.RepoDesc
	originPath    string
}

func RegisterType(recordType string, card RecordCardinality, factory RecordFactory) {
	recordTypes[recordType] = recordTypeInfo{
		factory: factory,
		card:    card,
	}
}

func LoadRegistryFromRepo(gc *gh.ThrottledClient, repo gh.RepoDesc, path string) (*Registry, error) {
	reg := &Registry{
		repos:         make(map[string]recordSet),
		records:       make(recordSet),
		globalRecords: make(map[string]Record),
		originRepo:    repo,
		originPath:    path,
	}

	t, _, err := gc.ThrottledCall(func(client *github.Client) (i interface{}, response *github.Response, e error) {
		return client.Git.GetTree(context.Background(), repo.OrgLogin, repo.RepoName, "master", true)
	})
	if err != nil {
		return nil, fmt.Errorf("unable to query GitHub for configuration state: %v", err)
	}

	tree := t.(*github.Tree)

	for _, entry := range tree.Entries {
		if strings.HasPrefix(entry.GetPath(), path) && strings.HasSuffix(entry.GetPath(), ".yaml") && entry.GetType() == "blob" {

			url := "https://raw.githubusercontent.com/" + repo.OrgAndRepo + "/" + repo.Branch + "/" + entry.GetPath()
			r, err := http.Get(url)
			if err != nil {
				return nil, fmt.Errorf("unable to fetch configuration file from %s: %v", url, err)
			}

			if r.StatusCode != 200 {
				return nil, fmt.Errorf("unable to fetch configuration file from %s: HTTP error %v", url, r.StatusCode)
			}

			var b []byte
			if b, err = ioutil.ReadAll(r.Body); err != nil {
				_ = r.Body.Close()
				return nil, fmt.Errorf("unable to read configuration file from %s: %v", url, err)
			}
			_ = r.Body.Close()

			if err := reg.processRecord(b); err != nil {
				return nil, fmt.Errorf("unable to parse configuration file %s in repo %s: %v", entry.GetPath(), repo, err)
			}
		}
	}

	if err = reg.postProcessAfterLoad(); err != nil {
		return nil, err
	}

	return reg, nil
}

func LoadRegistryFromDirectory(path string) (*Registry, error) {
	reg := &Registry{
		repos:         make(map[string]recordSet),
		records:       make(recordSet),
		globalRecords: make(map[string]Record),
		originRepo:    gh.RepoDesc{},
		originPath:    path,
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("unable to read configuration file %s: %v", path, err)
		}

		if err := reg.processRecord(b); err != nil {
			return fmt.Errorf("unable to parse configuration file %s: %v", path, err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if err = reg.postProcessAfterLoad(); err != nil {
		return nil, err
	}

	return reg, nil
}

func (reg *Registry) processRecord(b []byte) error {
	var r RecordBase
	if err := yaml.Unmarshal(b, &r); err != nil {
		return err
	}

	ri, ok := recordTypes[r.Type]
	if !ok {
		return fmt.Errorf("unsupported configuration type '%s'", r.Type)
	}

	if len(r.Repos) > 0 {
		for _, repo := range r.Repos {
			if !strings.Contains(repo, "/") {
				return fmt.Errorf("invalid repo name %s, needs to be in the form org/repo", repo)
			}
		}
	}

	o := ri.factory()
	if err := yaml.Unmarshal(b, o); err != nil {
		return err
	}

	if r.Type == recordType {
		reg.core = o.(*CoreRecord)
	}

	if ri.card == GlobalSingleton {
		if reg.globalRecords[r.Type] != nil {
			return fmt.Errorf("can't specify multiple configuration records of type '%s', it must be a global singleton", r.Type)
		}

		reg.globalRecords[r.Type] = o
	} else {
		reg.records[r.Type] = append(reg.records[r.Type], o)
	}

	return nil
}

func (reg *Registry) postProcessAfterLoad() error {
	if reg.core == nil {
		return errors.New("didn't find the required core configuration record")
	}

	// set the list of repos we care about
	for _, repo := range reg.core.Repos {
		if !strings.Contains(repo, "/") {
			return fmt.Errorf("invalid repo name %s, needs to be in the form org/repo", repo)
		}

		reg.allRepos = append(reg.allRepos, gh.NewRepoDesc(repo))
		reg.repos[repo] = make(recordSet)
	}

	for recType, recs := range reg.records {
		card := recordTypes[recType].card
		for _, rec := range recs {
			if len(rec.GetRepos()) == 0 {
				for _, repo := range reg.allRepos {
					recSet := reg.repos[repo.OrgAndRepo]
					if recSet == nil {
						continue
					}

					recSet[recType] = append(recSet[recType], rec)

					if card == OnePerRepo && len(recSet[recType]) > 1 {
						return fmt.Errorf("can't have multiple records of type %s matching the repo %s", recType, repo.OrgAndRepo)
					}
				}
			} else {
				for _, repo := range rec.GetRepos() {
					recSet := reg.repos[repo]
					if recSet == nil {
						continue
					}

					recSet[recType] = append(recSet[recType], rec)

					if card == OnePerRepo && len(recSet[recType]) > 1 {
						return fmt.Errorf("can't have multiple records of type %s matching the repo %s", recType, repo)
					}
				}
			}
		}
	}

	return nil
}

func (reg *Registry) Records(recordType string, orgAndRepo string) []Record {
	if orgAndRepo == "*" {
		// return the records for all the repos
		return reg.records[recordType]
	}

	recSet := reg.repos[orgAndRepo]
	if recSet == nil {
		return nil
	}

	return recSet[recordType]
}

func (reg *Registry) SingleRecord(recordType string, orgAndRepo string) (Record, bool) {
	recSet := reg.repos[orgAndRepo]
	if recSet == nil {
		return nil, false
	}

	if len(recSet[recordType]) == 0 {
		return nil, false
	}

	return recSet[recordType][0], true
}

func (reg *Registry) GlobalRecord(recordType string) (Record, bool) {
	result, ok := reg.globalRecords[recordType]
	return result, ok
}

func (reg *Registry) Repos() []gh.RepoDesc {
	return reg.allRepos
}

func (reg *Registry) Core() *CoreRecord {
	return reg.core
}

func (reg *Registry) OriginRepo() gh.RepoDesc {
	return reg.originRepo
}

func (reg *Registry) OriginPath() string {
	return reg.originPath
}
