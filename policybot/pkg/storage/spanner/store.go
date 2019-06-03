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

package spanner

import (
	"context"
	"fmt"

	"istio.io/pkg/log"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/option"

	"istio.io/bots/policybot/pkg/storage"
)

type store struct {
	client *spanner.Client
	ctx    context.Context
}

var scope = log.RegisterScope("spanner", "Spanner abstraction layer", 0)

func NewStore(ctx context.Context, database string, gcpCreds []byte) (storage.Store, error) {
	client, err := spanner.NewClient(ctx, database, option.WithCredentialsJSON(gcpCreds))
	if err != nil {
		return nil, fmt.Errorf("unable to create Spanner client: %v", err)
	}

	return &store{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *store) Close() error {
	s.client.Close()
	return nil
}
