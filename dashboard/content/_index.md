---
title: Istio Bots
description: Connect, secure, control, and observe services.
---
<main class="landing">

    <style>

    .auth-button {
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

    <button class="auth-button" id="login-button">Sign in with GitHub</button>
    <button class="auth-button" id="logout-button">Sign out from GitHub</button>

    <p id="name"></p>
    <p id="image"></p>
    <p id="policybot"></p>
</main>
