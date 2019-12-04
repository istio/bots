// Code generated for package maintainers by go-bindata DO NOT EDIT. (@generated)
// sources:
// list.html
// single.html
// single_control.html
// user.html
package maintainers

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

var _listHtml = []byte(`{{ if eq .Mode "inactive" }}
    <p>
        These kind folks are responsible for specific areas of the Istio product, guiding its development
        and maintaining its code base, but they've been inactive in the project for the last {{ .ActivityDays }} days.
    </p>
{{ else if eq .Mode "emeritus" }}
    <p>
        These kind folks were previously responsible for specific areas of the Istio product, but have taken a leave of the project. We'd love
        to have 'em back.
    </p>
{{ else }}
<p>
    These kind folks are responsible for specific areas of the Istio product, guiding its development
    and maintaining its code base.
</p>
{{ end }}

<div id="gallery" class="user-gallery active inactive">
</div>

<script>
    "use strict";

    function getMaintainers() {
        let url = "ws://" + window.location.host + "/api/maintainers/" + window.location.search;
        if (window.location.protocol === "https:") {
            url = "wss://" + window.location.host + "/api/maintainers/" + window.location.search;
        }

        const ws = new WebSocket(url);
        const gallery = document.getElementById("gallery");

        // Attach a popper to the given anchor
        function attachPopper(anchor, element) {
            if (popper) {
                popper.destroy();
            }

            popper = new Popper(anchor, element, {
                modifiers: {
                    flip: {
                        enabled: true,
                    },
                    preventOverflow: {
                        enabled: true,
                    },
                    shift: {
                        enabled: true,
                    },
                },
                placement: "auto-start",
            });
        }

        function detachPopper() {
            if (popper) {
                popper.destroy();
            }
        }

        ws.onmessage = evt => {
            const el = document.createElement("html");
            el.innerHTML = evt.data;

            const user = el.querySelector(".user");
            const popover = el.querySelector(".popover");
            const fit = el.querySelector(".fit");

            convertUTCToLocalDate(el);
            fitty(fit, {
                minSize: 12,
                maxSize: 22,
            });

            listen(user, mouseenter, e => {
                e.cancelBubble = true;
                toggleOverlay(popover);
                attachPopper(user, popover);
            });

            listen(user, mouseleave, e => {
                e.cancelBubble = true;
                toggleOverlay(popover);
                detachPopper();
            });

            gallery.appendChild(user);

            if (popover !== null) {
                listen(popover, click, e => {
                    e.cancelBubble = true;
                });

                gallery.appendChild(popover);
            }
        };
    }

    getMaintainers();
</script>
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

var _singleHtml = []byte(`<div class="user-page">
    <div class="profile">
        <div class="avatar">
            <img src="{{ .User.AvatarURL }}" />
        </div>

        <div class="info">
            <div class="property-table">
                {{ if .User.Name }}
                    <div class="name">
                        Name
                    </div>
                    <div class="value">
                        {{ .User.Name }}
                    </div>
                {{ end  }}

                <div class="name">
                    GitHub Handle
                </div>
                <div class="value">
                    <a href="https://github.com/{{ .User.UserLogin }}">{{ .User.UserLogin }}</a>
                </div>

                {{ if .User.Company }}
                    <div class="name">
                        Affiliation
                    </div>
                    <div class="value">
                        {{ .User.Company }}
                    </div>
                {{ end }}

                {{ if .Maintainer.Emeritus }}
                    <div class="name">
                        Emeritus
                    </div>
                {{ end }}
            </div>
        </div>
    </div>

    {{ if not .Maintainer.Emeritus }}
        {{ $maintainer := .Maintainer }}

        {{ range $repoName, $repoInfo := .MaintainerInfo.Repos }}
            <div class="repo">
                <div class="title">
                    Maintenance activity on the <a href="https://github.com/{{ $maintainer.OrgLogin }}/{{ $repoName }}">{{ $repoName }}</a> repo
                </div>

                <div class="property-table">
                    <div class="name">
                        Last comment left on an issue or PR
                    </div>
                    <div class="value">
                        {{ if $repoInfo.LastIssueCommented.Number }}
                            #<a href="https://github.com/{{ $maintainer.OrgLogin }}/{{ $repoName }}/issues/{{ $repoInfo.LastIssueCommented.Number }}">
                                {{ $repoInfo.LastIssueCommented.Number }}</a>
                            on <span class="utc">{{ $repoInfo.LastIssueCommented.Time }}</span>
                        {{ else }}
                            &lt;never&gt;
                        {{ end }}
                    </div>

                    <div class="name">
                        Last issue or PR triaged
                    </div>
                    <div class="value">
                        {{ if $repoInfo.LastIssueTriaged.Number }}
                            #<a href="https://github.com/{{ $maintainer.OrgLogin }}/{{ $repoName }}/issues/{{ $repoInfo.LastIssueTriaged.Number }}">
                                {{ $repoInfo.LastIssueTriaged.Number }}</a>
                            on <span class="utc">{{ $repoInfo.LastIssueTriaged.Time }}</span>
                        {{ else }}
                            &lt;never&gt;
                        {{ end }}
                    </div>

                    <div class="name">
                        Last issue or PR closed
                    </div>
                    <div class="value">
                        {{ if $repoInfo.LastIssueClosed.Number }}
                            #<a href="https://github.com/{{ $maintainer.OrgLogin }}/{{ $repoName }}/issues/{{ $repoInfo.LastIssueClosed.Number }}">
                                {{ $repoInfo.LastIssueClosed.Number }}</a>
                            on <span class="utc">{{ $repoInfo.LastIssueClosed.Time }}</span>
                        {{ else }}
                            &lt;never&gt;
                        {{ end }}
                    </div>
                </div>

                <div class="paths" title="Shows the maintainer's activity in the paths the maintainer is responsible for within this repo">
                    <div class="property-table">
                        <div class="name header">
                            Paths Maintained
                        </div>
                        <div class="value header inner">
                            Last Submitted
                        </div>
                        <div class="value header">
                            Last Reviewed
                        </div>

                        {{ range $pathName, $pathInfo := .Paths }}
                            <div class="name">
                                <a href="https://github.com/{{ $maintainer.OrgLogin }}/{{ $repoName }}/tree/master/{{ $pathName }}">/{{ $pathName }}</a>
                            </div>
                            <div class="value inner">
                                {{ if $pathInfo.LastPullRequestSubmitted.Number }}
                                    #<a href="https://github.com/{{ $maintainer.OrgLogin }}/{{ $repoName }}/issues/{{ $pathInfo.LastPullRequestSubmitted.Number }}">
                                        {{ $pathInfo.LastPullRequestSubmitted.Number }}</a>
                                    on <span class="utc">{{ $pathInfo.LastPullRequestSubmitted.Time }}</span>
                                {{ else }}
                                    &lt;never&gt;
                                {{ end }}
                            </div>

                            <div class="value">
                                {{ if $pathInfo.LastPullRequestReviewed.Number }}
                                    #<a href="https://github.com/{{ $maintainer.OrgLogin }}/{{ $repoName }}/issues/{{ $pathInfo.LastPullRequestReviewed.Number }}">
                                        {{ $pathInfo.LastPullRequestReviewed.Number }}</a>
                                    on <span class="utc">{{ $pathInfo.LastPullRequestReviewed.Time }}</span>
                                {{ else }}
                                    &lt;never&gt;
                                {{ end }}
                            </div>
                        {{ end }}
                    </div>
                </div>
            </div>
        {{ end }}
    {{ end }}
</div>

<script>
    "use strict";
    document.addEventListener("DOMContentLoaded", () => {
        convertUTCToLocalDate(document);
    });
</script>
`)

func singleHtmlBytes() ([]byte, error) {
	return _singleHtml, nil
}

func singleHtml() (*asset, error) {
	bytes, err := singleHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "single.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _single_controlHtml = []byte(`<a href="/maintainers">
    <svg class="icon"><use xlink:href="/icons/icons.svg#left-arrow"/></svg>
    Maintainers
</a>
`)

func single_controlHtmlBytes() ([]byte, error) {
	return _single_controlHtml, nil
}

func single_controlHtml() (*asset, error) {
	bytes, err := single_controlHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "single_control.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _userHtml = []byte(`<div class="user">
    <a href="/maintainers/{{ .User.UserLogin }}">
        <div class="name">
            <div class="fit">
                {{ if .User.Name }}
                    {{ .User.Name }}
                {{ else }}
                    {{ .User.UserLogin }}
                {{ end }}
            </div>
        </div>

        <div class="avatar">
            <img src="{{ .User.AvatarURL }}">
        </div>

        {{ if not .Maintainer.Emeritus }}
            <div class="num-repos">
                {{ if eq (len .MaintainerInfo.Repos) 1 }}
                    Maintainer in 1 repo
                {{ else }}
                    Maintainer in {{ len .MaintainerInfo.Repos }} repos
                {{ end }}
            </div>

            <div class="last-seen">
                Last active on <span class="utc">{{ .MaintainerInfo.LastActivity }}</span>
            </div>
        {{ end }}

        {{ if .Maintainer.Emeritus }}
           <div class="emeritus">
               Emeritus
           </div>
        {{ end }}
    </a>
</div>

{{ if not .Maintainer.Emeritus }}
    <div class="popover">
        <div class="title">
            {{ if .User.Name }}
                {{ .User.Name }}
            {{ else }}
                {{ .User.UserLogin }}
            {{ end }}
        </div>

        <div class="body">
            {{ if ne .User.UserLogin .User.Name }}
                GitHub handle is {{ .User.UserLogin }}<br/>
            {{ end }}
            {{ if .User.Company }}
                Affiliated with {{ .User.Company }}<br/>
            {{ end }}

            {{ if .Maintainer.Emeritus }}
                Emeritus<br/>
            {{ else }}
                {{ $timeZero := .TimeZero }}
                {{ range $repoName, $repoInfo := .MaintainerInfo.Repos }}
                    <p>
                        Repo {{ $repoName }}<br/>

                        {{ $lastIssueActivity := $repoInfo.LastIssueCommented.Time }}
                        {{ if $lastIssueActivity.Before $repoInfo.LastIssueClosed.Time }}
                            {{ $lastIssueActivity = $repoInfo.LastIssueClosed.Time }}
                        {{ end }}
                        {{ if $lastIssueActivity.Before $repoInfo.LastIssueTriaged.Time }}
                            {{ $lastIssueActivity = $repoInfo.LastIssueTriaged.Time }}
                        {{ end }}

                        {{ $lastPRActivity := $timeZero }}
                        {{ range $repoInfo.Paths }}
                            {{ if $lastPRActivity.Before .LastPullRequestSubmitted.Time }}
                                {{ $lastPRActivity = .LastPullRequestSubmitted.Time }}
                            {{ end }}

                            {{ if $lastPRActivity.Before .LastPullRequestReviewed.Time }}
                                {{ $lastPRActivity = .LastPullRequestReviewed.Time }}
                            {{ end }}
                        {{ end }}

                        Latest issue activity: <span class="utc">{{ $lastIssueActivity }}</span><br/>
                        Latest PR activity: <span class="utc">{{ $lastPRActivity }}</span><br/>
                    </p>
                {{ end }}
            {{ end }}
        </div>

        <div class="arrow" x-arrow>
        </div>
    </div>
{{ end }}
`)

func userHtmlBytes() ([]byte, error) {
	return _userHtml, nil
}

func userHtml() (*asset, error) {
	bytes, err := userHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "user.html", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
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
	"list.html":           listHtml,
	"single.html":         singleHtml,
	"single_control.html": single_controlHtml,
	"user.html":           userHtml,
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
	"list.html":           &bintree{listHtml, map[string]*bintree{}},
	"single.html":         &bintree{singleHtml, map[string]*bintree{}},
	"single_control.html": &bintree{single_controlHtml, map[string]*bintree{}},
	"user.html":           &bintree{userHtml, map[string]*bintree{}},
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
