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

package util

import (
	"context"

	"golang.org/x/time/rate"

	"istio.io/bots/policybot/pkg/zh"
)

const (
	maxZenHubRequestsPerMinute = 100.0                           // per-minute max, to stay under rate limit
	maxZenHubRequestsPerSecond = maxZenHubRequestsPerMinute / 60 // per-second max, to stay under abuse detection limit
	maxZenHubBurst             = 10                              // max burst size, to stay under abuse detection limit
)

// ZenHubThrottle is used to throttle our use of the ZenHub API in order to
// prevent hitting rate limits or abuse limits.
type ZenHubThrottle struct {
	ctx     context.Context
	client  *zh.Client
	limiter *rate.Limiter
}

func NewZenHubThrottle(ctx context.Context, zenhubToken string) *ZenHubThrottle {
	return &ZenHubThrottle{
		ctx:     ctx,
		client:  zh.NewClient(zenhubToken),
		limiter: rate.NewLimiter(maxZenHubRequestsPerSecond, maxZenHubBurst),
	}
}

// Get the ZenHub client in a throttled fashion, so we don't exceed ZenHub's usage limits. This will block
// until it is safe to make the call to GitHub.
func (zht *ZenHubThrottle) Get() *zh.Client {
	_ = zht.limiter.Wait(zht.ctx)
	return zht.client
}
