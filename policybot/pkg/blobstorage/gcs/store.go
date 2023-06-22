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

package gcs

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"

	"istio.io/bots/policybot/pkg/blobstorage"
	"istio.io/bots/policybot/pkg/pipeline"
)

type store struct {
	client *storage.Client
}

func NewStore(ctx context.Context) (blobstorage.Store, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create GCS client: %v", err)
	}

	return &store{
		client: client,
	}, nil
}

func (s *store) Bucket(name string) blobstorage.Bucket {
	return &bucket{bucket: s.client.Bucket(name)}
}

func (s *store) Close() error {
	return s.client.Close()
}

type bucket struct {
	bucket *storage.BucketHandle
}

func (b *bucket) Reader(ctx context.Context, path string) (io.ReadCloser, error) {
	return b.bucket.Object(path).NewReader(ctx)
}

func (b *bucket) ListPrefixes(ctx context.Context, prefix string) ([]string, error) {
	resultChan := b.ListPrefixesProducer(ctx, prefix)
	out, err := pipeline.BuildSlice(resultChan.Go())
	// cast to slice of string
	var result []string
	for _, o := range out {
		result = append(result, o.(string))
	}
	return result, err
}

func (b *bucket) ListPrefixesProducer(ctx context.Context, prefix string) pipeline.Pipeline {
	var query *storage.Query
	var it *storage.ObjectIterator
	lp := pipeline.IterProducer{
		Setup: func() error {
			query = &storage.Query{Prefix: prefix, Delimiter: "/"}
			it = b.bucket.Objects(ctx, query)
			return nil
		},
		Iterator: func() (res interface{}, err error) {
			attrs, err := it.Next()
			if err == nil {
				if attrs.Prefix == "" {
					err = pipeline.ErrSkip
				} else {
					res = attrs.Prefix
				}
			}
			return
		},
	}
	return pipeline.FromIter(lp)
}

func (b *bucket) ListPrefixesProducers(ctx context.Context, prefix string) (resultChan chan pipeline.OutResult) {
	var query *storage.Query
	var it *storage.ObjectIterator
	lp := pipeline.IterProducer{
		Setup: func() error {
			query = &storage.Query{Prefix: prefix, Delimiter: "/"}
			it = b.bucket.Objects(ctx, query)
			return nil
		},
		Iterator: func() (res interface{}, err error) {
			attrs, err := it.Next()
			if err == nil {
				if attrs.Prefix == "" {
					err = pipeline.ErrSkip
				} else {
					res = attrs.Prefix
				}
			}
			return
		},
	}
	return lp.Start(ctx, 1)
}

func (b *bucket) ListItems(ctx context.Context, prefix string) ([]string, error) {
	resultChan := b.ListItemsProducer(ctx, prefix)
	out, err := pipeline.BuildSlice(resultChan)
	// cast to slice of string
	var result []string
	for _, o := range out {
		result = append(result, o.(string))
	}
	return result, err
}

func (b *bucket) ListItemsProducer(ctx context.Context, prefix string) chan pipeline.OutResult {
	var query *storage.Query
	var it *storage.ObjectIterator
	lp := pipeline.IterProducer{
		Setup: func() error {
			query = &storage.Query{Prefix: prefix}
			it = b.bucket.Objects(ctx, query)
			return nil
		},
		Iterator: func() (res interface{}, err error) {
			attrs, err := it.Next()
			if err == nil {
				if attrs.Name == "" {
					err = pipeline.ErrSkip
				} else {
					res = attrs.Name
				}
			}
			return
		},
	}
	return lp.Start(ctx, 1)
}
