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
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/v26/github"
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

const (
	headerRateLimit     = "X-RateLimit-Limit"
	headerRateRemaining = "X-RateLimit-Remaining"
	headerRateReset     = "X-RateLimit-Reset"
)

func (c *Client) sendRequest(method, urlPath string) (*http.Response, error) {
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 == 2 {
		return resp, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, &github.RateLimitError{
			Rate:     parseRate(resp),
			Response: resp,
			Message:  "Limit reached",
		}
	}

	return resp, nil
}

// parseRate parses the rate related headers.
func parseRate(r *http.Response) github.Rate {
	var rate github.Rate
	if limit := r.Header.Get(headerRateLimit); limit != "" {
		rate.Limit, _ = strconv.Atoi(limit)
	}
	if remaining := r.Header.Get(headerRateRemaining); remaining != "" {
		rate.Remaining, _ = strconv.Atoi(remaining)
	}
	if reset := r.Header.Get(headerRateReset); reset != "" {
		if v, _ := strconv.ParseInt(reset, 10, 64); v != 0 {
			rate.Reset = github.Timestamp{time.Unix(v, 0)}
		}
	}
	return rate
}
