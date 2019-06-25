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

package filters

import (
	"context"

	webhook "github.com/go-playground/webhooks/github"
)

// The interface to a GitHub webhook filter.
//
// Note that individual filters are invoked for any events incoming to the
// bot. The events specified by this interface therefore don't constrain
// what Handle is used for and are merely used to control which events will
// be accepted as a whole by the webhook.
type Filter interface {
	Events() []webhook.Event
	Handle(context context.Context, ghPayload interface{})
}
