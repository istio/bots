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
These kind folks help Istio be what it is.
</p>

<table>
    <thead>
    <tr>
        <th>Avatar</th>
        <th>Login</th>
        <th>Name</th>
        <th>Company</th>
        <th>Emeritus</th>
    </tr>
    </thead>
    <tbody id="tbody">
    </tbody>
</table>

<script>
    "use strict";

    function fetchMaintainers() {
        let url = window.location.protocol + "//" + window.location.host + "/maintainersapi?org=istio";

        let ajax = new XMLHttpRequest();
        ajax.onload = onload;
        ajax.onerror = onerror;
        ajax.open("GET", url, true);
        ajax.send();

        function onload() {
            if (this.status === 200) { // request succeeded
                let maintainers = JSON.parse(this.responseText);
                const tbody = document.getElementById("tbody");

                let rows = ""
                for (let i = 0; i < maintainers.length; i++) {
                    const maintainer = maintainers[i];
                    const row = "<tr>"
                        + "<td><img style='width: 30px' src='" + maintainer.avatar_url + "'/></td>"
                        + "<td>" + maintainer.login + "</td>"
                        + "<td>" + maintainer.name + "</td>"
                        + "<td>" + maintainer.company + "</td>"
                        + "<td>" + maintainer.emeritus + "</td>"
                        + "</tr>";
                    rows += row;    
                }
                
                tbody.innerHTML = rows
            }
        }

        function onerror(e) {
            console.error(e);
        }
    }

    fetchMaintainers();
</script>
{{ end }}
`
