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
	"errors"
	"fmt"
	"sync/atomic"

	"google.golang.org/api/iterator"
)

var Skip = errors.New("This iteration should be skipped")

type IterProducer struct {
	Setup    func() error
	Iterator func() (interface{}, error)
}

type InResult interface {
	Input() interface{}
	Err() error
}

type OutResult interface {
	Err() error
	Output() interface{}
}

type InOutResult interface {
	Input() interface{}
	Err() error
	Output() interface{}
}

type simpleOut struct {
	err error
	out interface{}
}

type simpleInOut struct {
	simpleOut
	in interface{}
}

func (s simpleInOut) Input() interface{} {
	return s.in
}

func (s simpleOut) Err() error {
	return s.err
}

func (s simpleOut) Output() interface{} {
	return s.out
}

func NewOut(out interface{}, err error) OutResult {
	return simpleOut{
		err: err,
		out: out,
	}
}

func (sp *IterProducer) Start(ctx context.Context, bufferSize int) (resultChan chan OutResult) {
	resultChan = make(chan OutResult, bufferSize)
	go func() {
		defer close(resultChan)
		var err error
		if sp.Setup != nil {
			err = sp.Setup()
		}
		if err != nil {
			resultChan <- simpleOut{err, ""}
			return
		}
		for {
			select {
			case <-ctx.Done():
				// attempt a non-blocking write of ctx.Error()
				select {
				case resultChan <- simpleOut{ctx.Err(), ""}:
				default:
					return
				}
			default:
				result, err := sp.Iterator()
				if err != nil {
					if err == Skip {
						continue
					}
					if err == iterator.Done {
						return
					}
					resultChan <- simpleOut{err, ""}
				} else {
					resultChan <- simpleOut{nil, result}
				}
			}
		}
		fmt.Print("closing IterProducer")
	}()
	return
}

func BuildSlice(resultChan chan OutResult) ([]interface{}, error) {
	var items = make([]interface{}, 0)
	for result := range resultChan {
		if result.Err() != nil {
			return nil, result.Err()
		}
		items = append(items, result.Output())
	}
	return items, nil
}

func BuildProducer(ctx context.Context, input []interface{}) chan OutResult {
	var count int32
	sp := &IterProducer{
		Setup: func() error {
			return nil
		},
		Iterator: func() (interface{}, error) {
			if count > int32(len(input)-1) {
				fmt.Print("sending done")
				return "", iterator.Done
			}
			atomic.AddInt32(&count, 1)
			return input[count-1], nil
		},
	}
	return sp.Start(ctx, 1)
}
