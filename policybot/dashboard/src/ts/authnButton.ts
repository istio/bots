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

const authnButton = getById("authn-button");
const authenticator = new Authenticator();

authenticator.LoggedIn.on(resetAuthnButton);
authenticator.LoggedOut.on(resetAuthnButton);

authenticator.LoginError.on(() => {
//    console.log("Login error:", error);
});

if (authnButton !== null) {
    listen(authnButton, click, e => {
        e.preventDefault();
        if (authenticator.IsLoggedIn) {
            authenticator.Logout();
        } else {
            authenticator.Login();
        }
    });
}

function resetAuthnButton(): void {
    if (authnButton) {
        if (authenticator.IsLoggedIn) {
            authnButton.innerHTML = "<img class='large-icon' src='" + authenticator.UserAvatarUrl + "'>";
            authnButton.title = authenticator.UserName + "\nSign out from GitHub";
        } else {
            authnButton.innerHTML = '<svg class="large-icon"><use xlink:href="/icons/icons.svg#github"/></svg>';
            authnButton.title = "Sign in to GitHub";
        }
    }
}

resetAuthnButton();
