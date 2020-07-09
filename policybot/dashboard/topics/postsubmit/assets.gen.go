// Code generated by go-bindata.
// sources:
// analysis.html
// chooseBaseSha.html
// page.html
// DO NOT EDIT!

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

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _analysisHtml = []byte(`<p>
    This will display information about GitHub post submit text results
</p>

<ul id="myUL">
  {{ range .LabelEnv }}
  <li><span class="caret">{{ .Label }}</span>
    <ul class="nested">
      {{ range .EnvCount }}
      <li>{{ .Env }} : {{ .Counts }}</li>
      {{ end }}
      {{ range .SubLabel.LabelEnv}}
      <li><span class="caret">{{ .Label }}</span>
        <ul class="nested">
          {{ range .EnvCount }}
          <li>{{ .Env }} : {{ .Counts }}</li>
          {{ end }}
          {{ range .SubLabel.LabelEnv}}
          <li><span class="caret">{{ .Label }}</span>
            <ul class="nested">
              {{ range .EnvCount }}
              <li>{{ .Env }} : {{ .Counts }}</li>
              {{ end }}
            </ul>
          {{ end }}
          </li>
        </ul>
      </li>
      {{ end }}
    </ul>
  </li>
  {{ end }}
</ul>

<script>
  var toggler = document.getElementsByClassName("caret");
  var i;
  
  for (i = 0; i < toggler.length; i++) {
    toggler[i].addEventListener("click", function() {
      this.parentElement.querySelector(".nested").classList.toggle("active");
      this.classList.toggle("caret-down");
    });
  }
</script>

<style>
  ul, #myUL {
    list-style-type: none;
  }
  
  #myUL {
    margin: 0;
    padding: 0;
  }
  
  .caret {
    cursor: pointer;
    -webkit-user-select: none; /* Safari 3.1+ */
    -moz-user-select: none; /* Firefox 2+ */
    -ms-user-select: none; /* IE 10+ */
    user-select: none;
  }
  
  .caret::before {
    content: "\25B6";
    color: black;
    display: inline-block;
    margin-right: 6px;
  }
  
  .caret-down::before {
    -ms-transform: rotate(90deg); 
    -webkit-transform: rotate(90deg); 
    transform: rotate(90deg);  
  }
  
  .nested {
    display: none;
  }
  
  .active {
    display: block;
  }
  </style>`)

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
    <form action="/postsubmit?option=analysis" method="post">
    <select name="analysis" class="chosen">
        {{ range .BaseSha }}
        <option>{{ . }}</option>
        {{ end }}
    </select>
    <button type="submit">Submit</button>
    </form>
</body>

<script type="text/javascript">
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

<p>
    This will display information about GitHub post submit text results
</p>

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
	"analysis.html": analysisHtml,
	"chooseBaseSha.html": choosebaseshaHtml,
	"page.html": pageHtml,
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
	"analysis.html": &bintree{analysisHtml, map[string]*bintree{}},
	"chooseBaseSha.html": &bintree{choosebaseshaHtml, map[string]*bintree{}},
	"page.html": &bintree{pageHtml, map[string]*bintree{}},
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

