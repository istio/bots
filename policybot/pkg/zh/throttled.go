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
	"time"

	"github.com/google/go-github/v26/github"

	"istio.io/pkg/log"
)

// ZenHubThrottle is used to throttle our use of the ZenHub API in order to
// prevent hitting rate limits or abuse limits.
type ThrottledClient struct {
	client *Client
}

func NewThrottledClient(zenhubToken string) *ThrottledClient {
	return &ThrottledClient{
		client: NewClient(zenhubToken),
	}
}

// ThrottledCall invokes the given callback and watches for error returns indicating a GitHub rate limit errors.
// If a rate limit error is detected, the call is tried again based on the reset time
// specified in the error.
func (tc *ThrottledClient) ThrottledCall(cb func(*Client) (interface{}, error)) (interface{}, error) {
	for {
		result, err := cb(tc.client)
		if err == nil {
			return result, nil
		}

		rle, ok := err.(*github.RateLimitError)
		if !ok {
			return result, err
		}

		sleep(rle)
	}
}

func sleep(rle *github.RateLimitError) {
	// wait for the reset time
	// TODO: would be nice to wait in a cancellable way, per a context
	log.Debugf("Waiting for ZenHub rate limit reset at %s", rle.Rate.Reset.UTC().String())
	time.Sleep(time.Until(rle.Rate.Reset.Time))
}
