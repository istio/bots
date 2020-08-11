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
package notification

import (
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

//get Istio.io document owners information from github md file
func ReadDocsOwner() (docsOwnerMap map[string]string, err error) {
	file, err := os.Open("https://github.com/istio/istio.io/blob/master/DOC_OWNERS.md")
	if err != nil {
		return
	}
	defer file.Close()
	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		return
	}

	docsOwnerMap = make(map[string]string)
	doc.Find("html body h2").Each(func(index int, h2 *goquery.Selection) {
		owner := strings.Split(h2.Text(), ":")[0]
		nextEle := h2.Next()

		nextEle.Find("li").Each(func(index int, page *goquery.Selection) {
			pagelink := strings.TrimSuffix(page.Text(), "/index.md")
			docsOwnerMap[pagelink] = owner
		})

	})
	return
}
