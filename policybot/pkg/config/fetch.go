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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ghodss/yaml"
)

// Given a partially initialize config arg, load a local file or GitHub-based file
// to fill in the rest. Any data not specified in the input file will be left intact.
func (a *Args) Fetch() error {
	if a.StartupOptions.ConfigFile == "" {
		return errors.New("no configuration file supplied")
	}

	var b []byte
	var err error

	if a.StartupOptions.ConfigRepo == "" {
		if b, err = ioutil.ReadFile(a.StartupOptions.ConfigFile); err != nil {
			return fmt.Errorf("unable to read configuration file %s: %v", a.StartupOptions.ConfigFile, err)
		}

		if err = yaml.Unmarshal(b, &a); err != nil {
			return fmt.Errorf("unable to parse configuration file %s: %v", a.StartupOptions.ConfigFile, err)
		}
	} else {
		url := "https://raw.githubusercontent.com/" + a.StartupOptions.ConfigRepo + "/" + a.StartupOptions.ConfigFile
		r, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("unable to fetch configuration file from %s: %v", url, err)
		}
		defer r.Body.Close()

		if r.StatusCode >= 400 {
			return fmt.Errorf("unable to fetch configuration file from %s: status code %d", url, r.StatusCode)
		}
		if b, err = ioutil.ReadAll(r.Body); err != nil {
			return fmt.Errorf("unable to read configuration file from %s: %v", url, err)
		}

		if err = yaml.Unmarshal(b, &a); err != nil {
			return fmt.Errorf("unable to parse configuration file from %s: %v", url, err)
		}
	}

	return nil
}
