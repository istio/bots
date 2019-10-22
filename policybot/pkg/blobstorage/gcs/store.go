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

	"istio.io/bots/policybot/pkg/pipeline"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"istio.io/bots/policybot/pkg/blobstorage"
)

type store struct {
	client *storage.Client
}

func NewStore(ctx context.Context, gcpCreds []byte) (blobstorage.Store, error) {
	var opts []option.ClientOption
	if gcpCreds != nil {
		opts = append(opts, option.WithCredentialsJSON(gcpCreds))
	}

	client, err := storage.NewClient(ctx, opts...)
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
	return pipeline.BuildSlice(resultChan)
}

func (b *bucket) ListPrefixesProducer(ctx context.Context, prefix string) (resultChan chan pipeline.StringReslt) {
	var query *storage.Query
	var it *storage.ObjectIterator
	lp := pipeline.StringProducer{
		Setup: func() error {
			query = &storage.Query{Prefix: prefix, Delimiter: "/"}
			it = b.bucket.Objects(ctx, query)
			return nil
		},
		Iterator: func() (res string, err error) {
			attrs, err := it.Next()
			if err == nil {
				if attrs.Prefix == "" {
					err = pipeline.Skip
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
	return pipeline.BuildSlice(resultChan)
}

func (b *bucket) ListItemsProducer(ctx context.Context, prefix string) chan pipeline.StringReslt {
	var query *storage.Query
	var it *storage.ObjectIterator
	lp := pipeline.StringProducer{
		Setup: func() error {
			query = &storage.Query{Prefix: prefix}
			it = b.bucket.Objects(ctx, query)
			return nil
		},
		Iterator: func() (res string, err error) {
			attrs, err := it.Next()
			if err == nil {
				if attrs.Name == "" {
					err = pipeline.Skip
				} else {
					res = attrs.Name
				}
			}
			return
		},
	}
	return lp.Start(ctx, 1)
}
