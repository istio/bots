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
	"sync"
)

// TODO: Differentiate between fatal and non-fatal errors

type StringLogTransformer struct {
	// TODO: probably should have reference to the input here
	ErrHandler  func(error)
	Parallelism int
	BufferSize  int
}

func (slt *StringLogTransformer) Transform(ctx context.Context, in chan OutResult, transformer func(interface{}) (interface{}, error)) chan InOutResult {
	return Transform(ctx, slt.Parallelism, slt.BufferSize, in, transformer, slt.ErrHandler)
}

// Transform consumes a channel of string or errors and produces a channel of string or errors.
// All incoming errors will be passed to the error handler, which returns nothing.
// All incoming errorless strings will be passed to the transform function, whose results will be written to the
// resulting channel *unless* the returned error is ErrSkip, in which case that element is skipped.
func Transform(ctx context.Context, parallelism int, bufferSize int, in chan OutResult,
	transformer func(interface{}) (interface{}, error), errhandler func(error),
) chan InOutResult {
	// TODO: can we have a channel factory to do this?
	outChan := make(chan InOutResult, bufferSize)
	var wg sync.WaitGroup
	if parallelism < 1 {
		parallelism = 1
	}
	wg.Add(parallelism)
	i := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				out := simpleInOut{
					simpleOut: simpleOut{err: ctx.Err()},
				}
				select {
				case outChan <- out:
				default:
					return
				}
			case sr, ok := <-in:
				// do stuff, write to out maybe
				if !ok {
					// channel is closed, time to exit
					return
				}
				if sr.Err() != nil {
					if errhandler != nil {
						errhandler(sr.Err())
					}
					continue
				}
				res, err := transformer(sr.Output())
				if err == ErrSkip {
					continue
				}
				out := simpleInOut{
					simpleOut: simpleOut{err: err, out: res},
					in:        sr.Output(),
				}
				// TODO: this section will never cancel if this write blocks.  Problem?
				outChan <- out
			}
		}
	}
	for x := 0; x < parallelism; x++ {
		go i()
	}
	go func() {
		wg.Wait()
		close(outChan)
	}()
	return outChan
}
