package charts

var TimeseriesTemplate = `
{{ define "timeseries" }}
  <link href='https://fonts.googleapis.com/css?family=Open+Sans:400,300,700' rel='stylesheet' type='text/css'>
  <link href='https://fonts.googleapis.com/css?family=PT+Serif:400,700,400italic' rel='stylesheet' type='text/css'>
  <link href='https://netdna.bootstrapcdn.com/font-awesome/4.2.0/css/font-awesome.css' rel='stylesheet' type='text/css'>

  <link href='/charts/metricsgraphics.css' rel='stylesheet' type='text/css'>
  <script src='https://d3js.org/d3.v4.min.js' charset='utf-8'></script>
  <script type='text/javascript' src='/charts/metricsgraphics.min.js'></script>
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
`
