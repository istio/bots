---
title: Istio
description: Connect, secure, control, and observe services.
---
<main class="landing">

    <!-- Make sure to include Nelify's authentiation library -->
    <!-- Also available via npm as netlify-auth-providers -->
    <script src="https://unpkg.com/netlify-auth-providers"></script>

    <style>

    #login {
        display: inline-block;
        text-align: center;
        background-color: #28a745;
        background-image: linear-gradient(-180deg,#34d058,#28a745 90%);
        color: #fff;
        appearance: none;
        background-position: -1px -1px;
        background-repeat: repeat-x;
        background-size: 110% 110%;
        border: 1px solid rgba(27,31,35,.2);
        border-radius: .25em;
        cursor: pointer;
        font-size: 14px;
        font-weight: 600;
        line-height: 20px;
        padding: 6px 12px;
        position: relative;
        margin: 2em;
        user-select: none;
        vertical-align: middle;
        white-space: nowrap;
    }
    </style>

    <button id="login">Sign in with GitHub</button>

    <p id="name"></p>
    <p id="image"></p>
    <p id="policybot"></p>

    <script>
        fetchRepoData();

        let github_token = readCookie("github_token");
        if (github_token !== null) {
            document.getElementById("login").style.display = "none";
            fetchUserData();
            fetchRepoData();
        } else {
            document.getElementById("login").addEventListener("click", login);
        }

        function login(e) {
            e.preventDefault();
            var authenticator = new netlify.default ({});
            authenticator.authenticate({provider:"github", scope: "user"}, (err, data) => {
                if (err !== null) {
                    const name = document.getElementById("name");
                    name.innerText = err;
                    return;
                }

                github_token = data.token;
                createCookie("github_token", github_token);
                fetchUserData();
                fetchRepoData();
            })
        }

        function fetchRepoData() {
            fetch('https://policybot.istio.io/repos', {
            })
            .then(response => response.text())
            .then(data => {
                const policybot = document.getElementById("policybot");
                policybot.innerText = data;
            });
        }

        function fetchUserData() {
            fetch('https://api.github.com/user', {
                headers: {
                    "Authorization": "token " + github_token,
                }
            })
            .then(response => response.text())
            .then(data => {
                const json = JSON.parse(data);

                const name = document.getElementById("name");
                const image = document.getElementById("image");

                name.innerText = json.login;
                image.innerHTML = "<img style='width:30px' src='" + json.avatar_url + "'>";
            });
        }
  </script>
</main>
