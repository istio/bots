// Code generated for package issues by go-bindata DO NOT EDIT. (@generated)
// sources:
// list.html
// summary.html
package issues

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

var _listHtml = []byte(`<aside class="callout warning">
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

{{ if .AreaCounts }}
    <table>
        <thead>
        <tr>
            <th>Area</th>
            <th>Count</th>
        </tr>
        </thead>
        <tbody>
            {{ range .AreaCounts }}
                <tr>
                    <td>{{ .Area }}</td>
                    <td>{{ .Count }}</td>
                </tr>
            {{ end }}
        </tbody>
    </table>
{{ end }}

<table>
  <caption>{{ .Title }}</caption>
  <thead>
  <tr>
      <th>Repository</th>
      <th>Number</th>
      <th>Title</th>
      <th>Created</th>
      <th>Last Updated</th>
      <th>Assigned To</th>
  </tr>
  </thead>
  <tbody>
      {{ range .Issues }}
          <tr>
              <td>{{ .RepoName }}</td>
              <td><a href="https://github.com/istio/{{ .RepoName }}/issues/{{ .IssueNumber }}">{{ .IssueNumber }}</a></td>
              <td>{{ .Title }}</td>
              <td>{{ .CreatedAt }}</td>
              <td>{{ .UpdatedAt }}</td>
              <td>{{ .Assignees }}</td>
          </tr>
      {{ end }}
  </tbody>
</table>
`)

func listHtmlBytes() ([]byte, error) {
	return _listHtml, nil
}

func listHtml() (*asset, error) {
	bytes, err := listHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "list.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _summaryHtml = []byte(`<aside class="callout warning">
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
  <caption>Issues Opened By Month</caption>
  <thead>
  <tr>
    <th>Repository</th>
    {{ range .Months }}
      <th>{{ . }}</th>
    {{ end}}
  </tr>
  </thead>
  <tbody>
    {{ range .Opened }}
      <tr>
        <td>{{ .RepoName }}</td>
        {{ range .Counts }}
          <td>{{ . }}</td>
        {{ end }}
      </tr>
    {{ end }}
  </tbody>
</table>
`)

func summaryHtmlBytes() ([]byte, error) {
	return _summaryHtml, nil
}

func summaryHtml() (*asset, error) {
	bytes, err := summaryHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "summary.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
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
	"list.html":    listHtml,
	"summary.html": summaryHtml,
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
	"list.html":    &bintree{listHtml, map[string]*bintree{}},
	"summary.html": &bintree{summaryHtml, map[string]*bintree{}},
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
