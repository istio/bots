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

const loginButton = getById("login-button");
const logoutButton = getById("logout-button");

const authenticator = new Authenticator();

authenticator.LoggedIn.on(() => {
    resetAuthButtons();
    syncUserData();
});

authenticator.LoggedOut.on(() => {
    resetAuthButtons();
    syncUserData();
});

authenticator.LoginError.on(error => {
    console.log("Login error:", error);
});

function syncUserData(): void {
    const policybot = document.getElementById("policybot") as HTMLElement;
    const name = document.getElementById("name") as HTMLElement;
    const image = document.getElementById("image") as HTMLElement;

    if (authenticator.IsLoggedIn) {
        fetch('https://policybot.istio.io/repos', {})
            .then(response => response.text())
            .then(data => {
                policybot.innerText = data;
            });

        name.innerText = authenticator.UserName as string;
        image.innerHTML = "<img style='width:40px' src='" + authenticator.UserAvatarUrl + "'>";
    } else {
        policybot.innerText = "Not logged in";
        name.innerText = "";
        image.innerHTML = "";
    }
}

if (loginButton !== null) {
    listen(loginButton, click, e => {
        e.preventDefault();
        authenticator.Login();
    });
}

if (logoutButton !== null) {
    listen(logoutButton, click, e => {
        e.preventDefault();
        authenticator.Logout();
    });
}

function resetAuthButtons(): void {
    if (authenticator.IsLoggedIn) {
        if (loginButton !== null) {
            loginButton.style.display = "none";
        }

        if (logoutButton !== null) {
            logoutButton.style.display = "inline-block";
        }
    } else {
        if (loginButton !== null) {
            loginButton.style.display = "inline-block";
        }

        if (logoutButton != null) {
            logoutButton.style.display = "none";
        }
    }
}

resetAuthButtons();
syncUserData();
