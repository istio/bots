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

const gitHubTokenCookie = "gitHubToken";
declare const netlify: any;

// Provides authentication support for GitHub through the Netlify interface.
class Authenticator {
    private gitHubToken: string | null = readCookie(gitHubTokenCookie);
    private userLogin: string | null = null;
    private userName: string | null = null;
    private userAvatarUrl: string | null = null;

    // event handlers
    private readonly onLogin = new LiteEvent<string>();
    private readonly onLoginError = new LiteEvent<string>();
    private readonly onLogout = new LiteEvent<void>();

    constructor() {
        const urlParams = new URLSearchParams(window.location.search);
        const token = urlParams.get('access_token');

        if (token !== null) {
            this.gitHubToken = token;
            createCookie(gitHubTokenCookie, this.gitHubToken);
        }

        if (this.gitHubToken !== null) {
            this.fetchUserData(this.gitHubToken);
        }
    }

    public Login() {
        window.location.pathname = "login";
    }

    public Logout() {
        this.gitHubToken = null;
        this.userName = null;
        this.userAvatarUrl = null;
        deleteCookie(gitHubTokenCookie);
        this.onLogout.trigger();
    }

    public get LoggedIn() {
        return this.onLogin.expose();
    }

    public get LoginError() {
        return this.onLoginError.expose();
    }

    public get LoggedOut() {
        return this.onLogout.expose();
    }

    public get IsLoggedIn(): boolean {
        return this.gitHubToken !== null;
    }

    public get GitHubToken() {
        return this.gitHubToken;
    }

    public get UserLogin() {
        return this.userLogin;
    }

    public get UserName() {
        return this.userName;
    }

    public get UserAvatarUrl() {
        return this.userAvatarUrl;
    }

    private fetchUserData(gitHubToken: string): void {
        // TODO: Needs to trigger onLoginError event on failure
        fetch("https://api.github.com/user", {
            headers: {
                Authorization: "token " + gitHubToken,
            },
        })
            .then(response => response.text())
            .then(data => {
                const json = JSON.parse(data);
                this.gitHubToken = gitHubToken;
                this.userLogin = json.login;
                this.userName = json.name;
                this.userAvatarUrl = json.avatar_url;
                this.onLogin.trigger(this.userName as string);
            });
    }
}
