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

// Take in a pr number from blob storage and examines the pr
// for all tests that are run and their results. The results are then written to storage.
package pipeline

import (
	"context"
)

type Pipeline interface {
	Transform(func(interface{}) (interface{}, error)) Pipeline
	To(func(interface{}) error) PipelineEnd
	Batch(size int) Pipeline
	Go() chan OutResult
	Expand() Pipeline

	OnError(func(error)) Pipeline
	WithContext(ctx context.Context) Pipeline
	WithBuffer(int) Pipeline
	WithParallelism(int) Pipeline
}

type PipelineEnd interface {
	WithContext(ctx context.Context) PipelineEnd
	WithBuffer(int) PipelineEnd
	WithParallelism(int) PipelineEnd
	OnError(func(error)) PipelineEnd
	Go() chan InResult
}
