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

package templates

var PageNotFoundTemplate = `
{{ define "title" }}Page Not Found{{ end }}
{{ define "content" }}
<main class="notfound" role="main">
    <svg class="icon">
        <use xlink:href="/icons/icons.svg#exclamation-mark"/>
    </svg>

    <div class="error">
        We're sorry, the page you requested cannot be found
    </div>

    <div class="explanation">
        The URL may be misspelled or the page you're looking for is no longer available
    </div>
</main>
{{ end }}
`
