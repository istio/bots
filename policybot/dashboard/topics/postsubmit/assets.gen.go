// Code generated for package postsubmit by go-bindata DO NOT EDIT. (@generated)
// sources:
// analysis.html
// chooseBaseSha.html
// page.html
package postsubmit

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)
type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _analysisHtml = []byte(`<!DOCTYPE html>
<head>
  <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.5.1/jquery.min.js"></script>
</head>

<p>
    This will display information about Environment and Label for the BaseSha: {{ .Choosesha }} 
</p>

<div class="analysis">
  <table id="main">
    <thead>
    <tr>
        <th>LabelName</th>
        {{ range .AllEnvNanme }}
        <th>{{ . }}</th>
        {{ end }}
    </tr>
    </thead>
    <tbody>
      {{ $depth := 0 }}
      {{ range .LabelEnv }}
        <table>
          <thead>
            <tr>
              <td class="pivot">{{ .Label }}</td>
              {{ range .EnvCount }}
              <td>{{ . }} <button class="0" type="button">View All Test</button></td>
              {{ end }}
            </tr>
          </thead>
          {{template "innerlayer" (wrap .SubLabel $depth)}}      
        </table>
      {{ end }}
    </tbody>
  </table>
  <br>
  
  {{ if .ChooseEnv }}
  <p> View TestNames with Label: {{ .ChooseLabel }}, Env: {{ .ChooseEnv }}</p>
  <table id="testname">
    <thead>
      <tr>
        <th>TestName</th>
      </tr>
    </thead>
    <tbody>
      {{range $testname := .TestNameByEnvLabels}}
        <tr>
          <td>{{ $testname.TestOutcomeName }} 
            <a href="https://prow.istio.io/view/gcs/istio-prow/logs/{{ $testname.TestName }}/{{ $testname.RunNumber }}">Prow Link</a></td>
        </tr>
      {{ end }}
    </tbody>
  </table>
  {{ end }}
</div>

{{ define "innerlayer" }}
<tbody class="collapsed">
  <td class="subtd">
    {{$depth := .Depth}}
    {{ range .LabelEnv}}
      <table class="subtable">
        <thead>
          <tr>
            <td class="pivot">
              {{range slice $depth}}&nbsp;{{end}}{{ .Label }}
            </td>
            {{ range .EnvCount }}
            <td>{{ . }} <button class="{{$depth}}" type="button">View All Test</button></td>
            {{ end }}
          </tr>
        </thead>
        {{ if .SubLabel.LabelEnv }}
          {{template "innerlayer" (wrap .SubLabel $depth)}}
        {{ end }}
      </table>
    {{ end }}
  </td>       
</tbody>
{{ end }}

<script>
  $('.pivot').on('click', function(){
      $(this).closest("thead").next('tbody').toggleClass('collapsed');
  });
  $("button").click(function() {
    var label = getLabel($(this),parseInt(this.className));
    var env= getEnv($(this));
    postEnvLabel(env,label);
  });
  function postEnvLabel(env,label){
    $.ajax({
        url: "/selectEnvLabel",
        type: 'POST',
        data: {env:env, label:label},
    });
    window.location.assign("/postsubmit?option=analysis");
  };
  function getLabel($pos,num){
    var $label_pos = $pos.closest("tr").find(".pivot")
    var label= $label_pos.text().replace(/(\xA0|\r\n|\n|\r| )/gm,"")
    for (i = 0; i < num; i++) {
      $label_pos = $label_pos.closest("tbody").prev("thead").find("tr").find(".pivot")
      label = $label_pos.text().replace(/(\xA0|\r\n|\n|\r| )/gm,"") + "." + label;
    }
    return label;
  };
  function getEnv($pos){
    return $('#main thead th').eq($pos.closest("td").index()).text();
  };
</script>`)

func analysisHtmlBytes() ([]byte, error) {
	return _analysisHtml, nil
}

func analysisHtml() (*asset, error) {
	bytes, err := analysisHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "analysis.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _choosebaseshaHtml = []byte(`<!DOCTYPE html>
<head>
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.5.1/jquery.min.js"></script>
    <link rel="stylesheet" href="https://ajax.googleapis.com/ajax/libs/jqueryui/1.12.1/themes/smoothness/jquery-ui.css">
    <script src="https://ajax.googleapis.com/ajax/libs/jqueryui/1.12.1/jquery-ui.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/chosen/1.8.7/chosen.jquery.min.js"></script>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/chosen/1.8.7/chosen.min.css">
</head>

<body>
    Select BaseSha : <br/>
    <form id="myForm" name="myForm">
    <select name="basesha" class="chosen">
        {{ range .BaseSha }}
        <option>{{ . }}</option>
        {{ end }}
    </select>
    <input name="submit" type="submit" value="submit">
    </form>
</body>

<script type="text/javascript">
    $(document).ready(function() {
        $("#myForm").on("submit", function(e) {
            e.preventDefault();
            $.ajax({
                url: "/savebasesha",
                type: 'POST',
                data: $(this).serialize(),
            });
        window.location.assign("/postsubmit?option=analysis");
        });
    });
    $(".chosen").chosen();
</script>

</html>`)

func choosebaseshaHtmlBytes() ([]byte, error) {
	return _choosebaseshaHtml, nil
}

func choosebaseshaHtml() (*asset, error) {
	bytes, err := choosebaseshaHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "chooseBaseSha.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pageHtml = []byte(`<aside class="callout warning">
    <div class="type">
        <svg class="large-icon">
            <use xlink:href="/icons/icons.svg#callout-warning">
            </use>
        </svg>
    </div>
    <div class="content">
        This page is under construction
    </div>
</aside>

<table>
  <caption>Latest 100 BaseSha</caption>
  <thead>
  <tr>
    <th>BaseSha</th>
    <th>LastFinishTime</th>
    <th>NumberofTestDone</th>
  </tr>
  </thead>
  <tbody>
    {{ range .LatestBaseSha }}
    <tr>
        <td>{{ .BaseSha }}</td>
        <td>{{ .LastFinishTime }}</td>
        <td>{{ .NumberofTest }}</td>
    </tr>
    {{ end }}
  </tbody>
</table>`)

func pageHtmlBytes() ([]byte, error) {
	return _pageHtml, nil
}

func pageHtml() (*asset, error) {
	bytes, err := pageHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "page.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"analysis.html":      analysisHtml,
	"chooseBaseSha.html": choosebaseshaHtml,
	"page.html":          pageHtml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"analysis.html":      &bintree{analysisHtml, map[string]*bintree{}},
	"chooseBaseSha.html": &bintree{choosebaseshaHtml, map[string]*bintree{}},
	"page.html":          &bintree{pageHtml, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
