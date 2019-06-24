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

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"istio.io/bots/policybot/pkg/blobstorage"
)

type store struct {
	client *storage.Client
}

func NewStore(ctx context.Context, gcpCreds []byte) (blobstorage.Store, error) {
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON(gcpCreds))
	if err != nil {
		return nil, fmt.Errorf("unable to create GCS client: %v", err)
	}

	return &store{
		client: client,
	}, nil
}

func (s *store) Close() error {
	return s.client.Close()
}

func (s *store) ReadBlob(path string) ([]byte, error) {
	return nil, nil
}
