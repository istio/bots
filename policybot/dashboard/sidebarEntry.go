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

package dashboard

import (
	"github.com/gorilla/mux"

	"istio.io/bots/policybot/dashboard/types"
)

// sidebarEntry represents an entry in the left-hand navigation area.
type sidebarEntry struct {
	Title       string
	Description string
	URL         string
	Entries     []*sidebarEntry
	Routes      []*mux.Route
	Parent      *sidebarEntry
	Dashboard   *Dashboard
}

func (se *sidebarEntry) addEntry(title string, description string) *sidebarEntry {
	newEntry := &sidebarEntry{
		Title:       title,
		Description: description,
		Dashboard:   se.Dashboard,
		Parent:      se,
	}

	newEntry.Parent.Entries = append(newEntry.Parent.Entries, newEntry)
	newEntry.Dashboard.currentEntry = newEntry

	return newEntry
}

func (se *sidebarEntry) endEntry() *sidebarEntry {
	se.Dashboard.currentEntry = se.Parent
	return se.Parent
}

func (se *sidebarEntry) addPage(path string, render types.RenderFunc) *sidebarEntry {
	if len(se.Routes) == 0 {
		// use the first page associated with this entry as the entry's target URL
		se.URL = path
	}

	route := se.Dashboard.registerUIPage(path, render)
	se.Dashboard.entryMap[route] = se

	se.Routes = append(se.Routes, route)
	return se
}

// nolint: unparam
func (se *sidebarEntry) addPageWithQuery(path string, queryName string, queryValue string, render types.RenderFunc) *sidebarEntry {
	if len(se.Routes) == 0 {
		// use the first page associated with this entry as the entry's target URL
		se.URL = path + "?" + queryName + "=" + queryValue
	}

	route := se.Dashboard.registerUIPage(path, render).Queries(queryName, queryValue)
	se.Dashboard.entryMap[route] = se

	se.Routes = append(se.Routes, route)
	return se
}

func (se *sidebarEntry) IsSame(other *sidebarEntry) bool {
	return se == other
}

func (se *sidebarEntry) IsAncestor(other *sidebarEntry) bool {
	for {
		if other == nil {
			return false
		}

		if other == se {
			return true
		}

		other = other.Parent
	}
}
