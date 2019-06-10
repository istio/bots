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

var HeaderTemplate = `
{{ define "header" }}
<header>
    <nav>
        <a id="brand" href="/">
            <span class="logo">
				<svg viewBox="0 0 300 300">
				<circle cx="150" cy="150" r="146" stroke-width="2" />
				<polygon points="65,240 225,240 125,270"/>
				<polygon points="65,230 125,220 125,110"/>
				<polygon points="135,220 225,230 135,30"/>
				</svg>
			</span>
            <span class="name">Istio Eng Dashboard</span>
        </a>

        <div id="hamburger">
			<svg class="icon"><use xlink:href="/icons/icons.svg#hamburger"/></svg>
		</div>

        <div id="header-links">
			<button id="authn-button" title="Sign in to GitHub" aria-label="Sign in to GitHub">
				<svg class="icon"><use xlink:href="/icons/icons.svg#github"/></svg>
			</button>

            <div class="menu">
                <button id="gearDropdownButton" class="menu-trigger" title='Options and Settings" }}'
                        aria-label="Options and Settings" aria-controls="gearDropdownContent">
					<svg class="icon"><use xlink:href="/icons/icons.svg#gear"/></svg>
                </button>

                <div id="gearDropdownContent" class="menu-content" aria-labelledby="gearDropdownButton" role="menu">
                    <a tabindex="-1" role="menuitem" class="active" id="light-theme-item">Light Theme</a>
                    <a tabindex="-1" role="menuitem" id="dark-theme-item">Dark Theme</a>
                </div>
            </div>
        </div>
    </nav>
</header>
{{ end }}
`
