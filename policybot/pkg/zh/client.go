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
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	baseURL   = "https://api.zenhub.io"
	userAgent = "istio-policybot"
	timeout   = 30 * time.Second
)

type Client struct {
	authToken string
}

func NewClient(authToken string) *Client {
	return &Client{
		authToken: authToken,
	}
}

var ErrNotFound = errors.New("requested resource not found")

func (c *Client) sendRequest(method, urlPath string) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, baseURL+urlPath, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Authentication-Token", c.authToken)
	req.Header.Set("User-Agent", userAgent)
	if req.Method == "PUT" || req.Method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	return resp, nil
}
