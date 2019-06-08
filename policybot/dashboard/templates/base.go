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

var BaseTemplate = `<!DOCTYPE html>
<html lang="en" itemscope itemtype="https://schema.org/WebPage">
    <head>
        <meta charset="utf-8">
        <meta http-equiv="X-UA-Compatible" content="IE=edge">
        <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
        <meta name="theme-color" content="#466BB0"/>

        {{ define "description" }}DESCRIPTION{{ end }}

        <meta name="title" content="TITLE">
        <meta name="description" content="{{ template "description" . }}">

		<title>TITLE</title>

        <!-- Google Analytics -->
        {{ $analytics_id := "ANALYTICSID" }}
        <script async src="https://www.googletagmanager.com/gtag/js?id={{ $analytics_id }}"></script>
        <script>
            window.dataLayer = window.dataLayer || [];
            function gtag(){dataLayer.push(arguments);}
            gtag('js', new Date());
            gtag('config', '{{ $analytics_id }}');
        </script>
        <!-- End Google Analytics -->

        <!-- Favicons: generated from img/istio-whitelogo-bluebackground-framed.svg by http://cthedot.de/icongen -->
        <link rel="shortcut icon" href="/favicons/favicon.ico" >
        <link rel="apple-touch-icon" href="/favicons/apple-touch-icon-180x180.png" sizes="180x180">
        <link rel="icon" type="image/png" href="/favicons/favicon-16x16.png" sizes="16x16">
        <link rel="icon" type="image/png" href="/favicons/favicon-32x32.png" sizes="32x32">
        <link rel="icon" type="image/png" href="/favicons/android-36x36.png" sizes="36x36">
        <link rel="icon" type="image/png" href="/favicons/android-48x48.png" sizes="48x48">
        <link rel="icon" type="image/png" href="/favicons/android-72x72.png" sizes="72x72">
        <link rel="icon" type="image/png" href="/favicons/android-96x96.png" sizes="96xW96">
        <link rel="icon" type="image/png" href="/favicons/android-144x144.png" sizes="144x144">
        <link rel="icon" type="image/png" href="/favicons/android-192x192.png" sizes="192x192">

        <!-- app manifests -->
        <link rel="manifest" href="/manifest.json">
        <meta name="apple-mobile-web-app-title" content="Istio">
        <meta name="application-name" content="Istio">

        <!-- style sheets -->
        <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Work+Sans:400|Chivo:400|Work+Sans:500,300,600,300italic,400italic` +
	`,500italic,600italic|Chivo:500,300,600,300italic,400italic,500italic,600italic">
        <link rel="stylesheet" href="/css/all.css">
    </head>

    <body>
        <!-- set the color theme as soon as possible -->
        <script src="/js/themes_init.min.js"></script>

        <!-- libraries we unconditionally pull in -->
        <script src="https://www.google.com/cse/brand?form=search-form" defer></script>

        <!-- our own stuff -->
        <script src="/js/all.min.js" data-manual defer></script>

        {{ template "header" . }}
        {{ template "main" . }}
    </body>
</html>
`
