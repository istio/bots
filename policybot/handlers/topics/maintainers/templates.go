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

package maintainers

var maintainerTemplate = `
{{ define "content" }}

<p>
These kind folks are responsible for specific areas of the Istio product, guiding its development
and maintaining its code base.
</p>

<table>
    <thead>
    <tr>
        <th>Avatar</th>
        <th>Login</th>
        <th>Name</th>
        <th>Company</th>
        <th>Emeritus</th>
		<th>Paths</th>
    </tr>
    </thead>
    <tbody id="tbody">
    </tbody>
</table>

<script>
    "use strict";

    function refreshMaintainers() {
        const url = window.location.protocol + "//" + window.location.host + "/maintainersapi/";

		fetch(url)
			.then(response => {
				if (response.status !== 200) {
					return "Unable to access " + url + ": " + response.statusText;
				}

				return response.text();
			})
			.catch(e => {
				return "Unable to access " + url + ": " + e;
			})
			.then(data => {
                const maintainers = JSON.parse(data);

				const tbody = document.getElementById("tbody");
				for (let i = 0; i < maintainers.length; i++) {
				    const row = document.createElement("tr");

				    const avatarCell = document.createElement("td");
				    avatarCell.innerText = maintainers[i].avatar_url;
				    row.appendChild(avatarCell);

				    const loginCell = document.createElement("td");
				    loginCell.innerText = maintainers[i].login;
				    row.appendChild(loginCell);

				    const nameCell = document.createElement("td");
				    nameCell.innerText = maintainers[i].name;
				    row.appendChild(nameCell);

				    const companyCell = document.createElement("td");
				    companyCell.innerText = maintainers[i].company;
				    row.appendChild(companyCell);

				    const emeritusCell = document.createElement("td");
				    emeritusCell.innerText = maintainers[i].emeritus;
				    row.appendChild(emeritusCell);

				    tbody.appendChild(row);
				}
			});
    }

    refreshMaintainers();
</script>
{{ end }}
`
