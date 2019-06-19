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

package perf

var perfTemplate = `
{{ define "chart" }}
  <link href='https://fonts.googleapis.com/css?family=Open+Sans:400,300,700' rel='stylesheet' type='text/css'>
  <link href='https://fonts.googleapis.com/css?family=PT+Serif:400,700,400italic' rel='stylesheet' type='text/css'>
  <link href='https://netdna.bootstrapcdn.com/font-awesome/4.2.0/css/font-awesome.css' rel='stylesheet' type='text/css'>

  <link href='/libraries/metricsgraphics.css' rel='stylesheet' type='text/css'>
  <script src='https://d3js.org/d3.v4.min.js' charset='utf-8'></script>
  <script type='text/javascript' src='/libraries/metricsgraphics.min.js'></script>
	<div id={{ .Target }}></div>
	<script>
	  var data = MG.convert.date(JSON.parse({{ .TimeSeries }}), 'date');
		MG.data_graphic({
	    title: {{ .Name }},
	    description: "This graphic shows a time-series of downloads.",
	    data: data,
	    area: false,
      interpolate: d3.curveLinear,
	    show_tooltips: false,
	    width: 600,
	    height: 250,
	    target: '#' + {{ .Target }},
	    x_accessor: 'date',
	    y_accessor: 'value',
	  });
  </script>
{{ end }}

{{ define "content" }}

  <p>
  Istio Performance Test Results
  </p>

	{{ range . }}
		{{ template "chart" . }}
	{{ end }}

{{ end }}
`
