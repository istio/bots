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

var SidebarTemplate = `
{{ define "sidebar" }}
<nav id="sidebar" aria-label="Section Navigation">
    <div class="directory">
        <div class="card">
            <div id="header0" class="header">
	            Istio Eng Dashboard
        	</div>

            <div class="body default" aria-labelledby="header0">
				<ul role="tree" aria-expanded="true">
					{{ range getTopics }}
						<li role="none">
							<a role="treeitem" href="{{.URL}}">{{.Name}}</a>
						</li>
					{{ end }}
				</ul>
			</div>
        </div>
    </div>
</nav>
{{ end }}
`
