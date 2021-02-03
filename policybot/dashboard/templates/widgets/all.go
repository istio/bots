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

//go:generate ../../../scripts/gen_topic.sh

package widgets

var (
	HeaderTemplate         = string(MustAsset("header.html"))
	SidebarTemplate        = string(MustAsset("sidebar.html"))
	SidebarLevelTemplate   = string(MustAsset("sidebar_level.html"))
	TimeSeriesInitTemplate = string(MustAsset("timeseries_init.html"))
	TimeSeriesTemplate     = string(MustAsset("timeseries.html"))
)
