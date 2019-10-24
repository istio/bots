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

package blobstorage

import (
	"context"
	"io"

	"istio.io/bots/policybot/pkg/pipeline"
	pipelinetwo "istio.io/bots/policybot/pkg/pipeline2"
)

// Store defines how the bot interacts with a blob store
type Store interface {
	io.Closer

	Bucket(name string) Bucket
}

// Bucket represents a group of blobs.
type Bucket interface {
	Reader(ctx context.Context, path string) (io.ReadCloser, error)

	// ListPrefixes returns a slice of prefixes that begin with the input
	// prefix. This is roughly equivalent to a list of directories directly
	// under a given prefix, though in blob storage systems, directories
	// don't really exist.
	ListPrefixes(ctx context.Context, prefix string) ([]string, error)

	ListPrefixesProducer(ctx context.Context, prefix string) pipelinetwo.Pipeline
	ListItemsProducer(ctx context.Context, prefix string) chan pipeline.StringOutResult

	// ListItems returns a slice of GCS object names that begin with the input
	// prefix.
	ListItems(ctx context.Context, prefix string) ([]string, error)
}
