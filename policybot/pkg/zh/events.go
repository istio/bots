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

package zh

type IssueTransferEvent struct {
	Type             string `json:"type"`
	GitHubURL        string `json:"github_url"`
	Organization     string `json:"organization"`
	Repo             string `json:"repo"`
	UserName         string `json:"user_name"`
	IssueNumber      int    `json:"issue_number"`
	IssueTitle       string `json:"issue_title"`
	ToPipelineName   string `json:"to_pipeline_name"`
	FromPipelineName string `json:"from_pipeline_name"`
}

type EstimateSetEvent struct {
	Type         string `json:"type"`
	GitHubURL    string `json:"github_url"`
	Organization string `json:"organization"`
	Repo         string `json:"repo"`
	UserName     string `json:"user_name"`
	IssueNumber  int    `json:"issue_number"`
	IssueTitle   string `json:"issue_title"`
	Estimate     string `json:"estimate"`
}

type EstimateClearedEvent struct {
	Type         string `json:"type"`
	GitHubURL    string `json:"github_url"`
	Organization string `json:"organization"`
	Repo         string `json:"repo"`
	UserName     string `json:"user_name"`
	IssueNumber  int    `json:"issue_number"`
	IssueTitle   string `json:"issue_title"`
}

type IssueReprioritizedEvent struct {
	Type           string `json:"type"`
	GitHubURL      string `json:"github_url"`
	Organization   string `json:"organization"`
	Repo           string `json:"repo"`
	UserName       string `json:"user_name"`
	IssueNumber    int    `json:"issue_number"`
	IssueTitle     string `json:"issue_title"`
	ToPipelineName string `json:"to_pipeline_name"`
	FromPosition   int    `json:"from_position"`
	ToPosition     int    `json:"to_position"`
}
