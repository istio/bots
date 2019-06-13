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

package zh

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

type Estimate struct {
	Value int `json:"value"`
}

type PlusOne struct {
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Pipeline struct {
	Name string `json:"name"`
}

type IssueData struct {
	Estimate Estimate  `json:"estimate"`
	PlusOnes []PlusOne `json:"plus_ones"`
	Pipeline Pipeline  `json:"pipeline"`
	IsEpic   bool      `json:"is_epic"`
}

// Query ZenHub
func (c *Client) GetIssueData(repo, issue int) (*IssueData, error) {
	resp, err := c.sendRequest("GET", fmt.Sprintf("/p1/repositories/%d/issues/%d", repo, issue))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := &IssueData{}
	if err = json.Unmarshal(body, data); err != nil {
		return nil, err
	}

	return data, nil
}
