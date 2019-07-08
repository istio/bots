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

package dashboard

import (
	"github.com/gorilla/mux"
)

// Topic represents a single major functional area within the dashboard
type Topic interface {
	// Title returns the title for the area, which will be used in the sidenav and window title.
	Title() string

	// Description returns a general deacription for the area
	Description() string

	// The name of this topic, used with URLs
	Name() string

	// Nested topics
	Subtopics() []Topic

	// Installs the routes
	Configure(htmlRouter *mux.Router, apiRouter *mux.Router, rc RenderContext, opt *Options)
}
