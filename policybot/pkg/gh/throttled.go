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

import (
	"time"

	"github.com/google/go-github/v26/github"

	"istio.io/pkg/log"
)

// ThrottledCall invokes the given callback and watches for error returns indicating a GitHub rate limit errors.
// If a rate limit error is detected, the call is tried again based on the reset time
// specified in the error.
func ThrottledCall(cb func() (interface{}, *github.Response, error)) (interface{}, *github.Response, error) {
	for {
		result, resp, err := cb()
		if err == nil {
			return result, resp, nil
		}

		_, ok := err.(*github.RateLimitError)
		if !ok {
			return result, resp, err
		}

		sleep(resp)
	}
}

// ThrottledCallNoResult invokes the given callback and watches for error returns indicating a GitHub rate limit errors.
// If a rate limit error is detected, the call is tried again based on the reset time
// specified in the error.
func ThrottledCallNoResult(cb func() (*github.Response, error)) (*github.Response, error) {
	for {
		resp, err := cb()
		if err == nil {
			return resp, nil
		}

		_, ok := err.(*github.RateLimitError)
		if !ok {
			return resp, err
		}

		sleep(resp)
	}
}

// ThrottledCallTwoResult invokes the given callback and watches for error returns indicating a GitHub rate limit errors.
// If a rate limit error is detected, the call is tried again based on the reset time
// specified in the error.
func ThrottledCallTwoResult(cb func() (interface{}, interface{}, *github.Response, error)) (interface{}, interface{}, *github.Response, error) {
	for {
		result1, result2, resp, err := cb()
		if err == nil {
			return result1, result2, resp, nil
		}

		_, ok := err.(*github.RateLimitError)
		if !ok {
			return result1, result2, resp, err
		}

		sleep(resp)
	}
}

func sleep(resp *github.Response) {
	// wait for the reset time
	// TODO: would be nice to wait in a cancellable way, per a context
	log.Debugf("Waiting for GitHub rate limit reset at %s", resp.Rate.Reset.String())
	time.Sleep(time.Until(resp.Rate.Reset.Time))
}
