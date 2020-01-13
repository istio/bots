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
	"reflect"
	"sync"

	"github.com/eapache/channels"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/api/iterator"
)

type Impl struct {
	ctx         context.Context
	bufferSize  int
	parallelism int
	priorStep   Pipeline
	// exec acts like a receiver function, but is late bound
	exec         func(chan OutResult, *Impl) chan OutResult
	errorHandler func(error)
}

func (sp *Impl) Expand() Pipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *Impl) chan OutResult {
		outChan := make(chan OutResult, nx.bufferSize)
		var wg sync.WaitGroup

		if sp.parallelism < 1 {
			sp.parallelism = 1
		}
		wg.Add(nx.parallelism)
		i := func() {
			defer wg.Done()
			for {
				select {
				case <-nx.ctx.Done():
					out := simpleInOut{
						simpleOut: simpleOut{err: nx.ctx.Err()},
					}
					select {
					case outChan <- out:
					default:
						return
					}
				case sr, ok := <-in:
					//do stuff, write to out maybe
					if !ok {
						// channel is closed, time to exit
						return
					}
					if sr.Err() != nil {
						if nx.errorHandler != nil {
							nx.errorHandler(sr.Err())
						}
						continue
					}
					var priorInput interface{}
					if io, ok := sr.(InResult); ok {
						priorInput = io.Input()
					}
					// output here is an array stored in an interface{}, which is not rangeable
					switch reflect.TypeOf(sr.Output()).Kind() {
					case reflect.Slice:
						s := reflect.ValueOf(sr.Output())
						for i := 0; i < s.Len(); i++ {
							out := simpleInOut{
								simpleOut: simpleOut{
									out: s.Index(i).Interface(),
								},
								in: priorInput,
							}
							outChan <- out
						}
					default:
						out := simpleInOut{
							simpleOut: simpleOut{
								out: sr.Output(),
							},
							in: priorInput,
						}
						outChan <- out
					}
					// TODO: this section will never cancel if this write blocks.  Problem?
				}
			}
		}
		for x := 0; x < nx.parallelism; x++ {
			go i()
		}
		go func() {
			wg.Wait()
			close(outChan)
		}()
		return outChan
	}
	return &next
}

// TODO: the With and On functions need clarification around chaining
func (sp *Impl) WithContext(ctx context.Context) Pipeline {
	sp.ctx = ctx
	return sp
}

func (sp *Impl) WithBuffer(i int) Pipeline {
	sp.bufferSize = i
	return sp
}

func (sp *Impl) WithParallelism(i int) Pipeline {
	sp.parallelism = i
	return sp
}

type StringPipelineEnder struct {
	ctx          context.Context
	bufferSize   int
	parallelism  int
	priorStep    Pipeline
	exec         func(chan OutResult, *StringPipelineEnder) chan InResult
	errorHandler func(error)
}

func (spe *StringPipelineEnder) WithContext(ctx context.Context) End {
	spe.ctx = ctx
	return spe
}

func (spe *StringPipelineEnder) WithBuffer(i int) End {
	spe.bufferSize = i
	return spe
}

func (spe *StringPipelineEnder) WithParallelism(i int) End {
	spe.parallelism = i
	return spe
}

func (spe *StringPipelineEnder) OnError(f func(error)) End {
	spe.errorHandler = f
	return spe
}

func (spe *StringPipelineEnder) Go() chan InResult {
	// Ender's always have priors
	priorOut := spe.priorStep.Go() // TODO: handle errors here?
	return spe.exec(priorOut, spe)
}

func (sp *Impl) Go() chan OutResult {
	var priorOut chan OutResult
	if sp.priorStep != nil {
		priorOut = sp.priorStep.Go() // TODO: handle errors here?
	}
	return sp.exec(priorOut, sp)
}

func (sp *Impl) OnError(f func(error)) Pipeline {
	sp.errorHandler = f
	return sp
}

func (sp *Impl) makeChild() Impl {
	child := *sp
	child.priorStep = sp
	child.exec = nil
	return child
}

func (sp *Impl) Batch(size int) Pipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *Impl) (out chan OutResult) {
		out = make(chan OutResult, nx.bufferSize)
		wrapper := channels.Wrap(in)
		f := channels.NewBatchingChannel(channels.BufferCap(size))
		channels.Pipe(wrapper, f)
		go func() {
			defer close(out)
			for x := range f.Out() {
				iter := x.([]interface{})
				var outSlice []interface{}
				var errSlice error
				for _, itf := range iter {
					res := itf.(OutResult)
					if res.Err() == nil {
						outSlice = append(outSlice, res.Output())
					} else {
						errSlice = multierror.Append(errSlice, res.Err())
					}
				}
				batchOut := simpleOut{
					err: errSlice,
					out: outSlice,
				}
				out <- batchOut
			}
		}()
		return
	}
	return &next
}

func (sp *Impl) Transform(f func(result interface{}) (interface{}, error)) Pipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *Impl) chan OutResult {
		result := make(chan OutResult, sp.bufferSize)
		t := StringLogTransformer{
			ErrHandler:  nx.errorHandler,
			Parallelism: nx.parallelism,
			BufferSize:  nx.bufferSize,
		}
		input := t.Transform(nx.ctx, in, f)
		go func() {
			for i := range input {
				// this nonsense is necessary because channels don't support inheritance
				result <- i
			}
			close(result)
		}()
		return result
	}
	return &next
}

func (sp *Impl) To(f func(result interface{}) error) End {
	next := StringPipelineEnder{
		ctx:         sp.ctx,
		bufferSize:  sp.bufferSize,
		parallelism: sp.parallelism,
		priorStep:   sp,
	}
	t := StringLogTransformer{
		ErrHandler:  sp.errorHandler,
		Parallelism: sp.parallelism,
		BufferSize:  sp.bufferSize,
	}
	next.exec = func(in chan OutResult, nx *StringPipelineEnder) chan InResult {
		result := make(chan InResult, sp.bufferSize)
		g := func(result interface{}) (interface{}, error) { return "", f(result) }
		input := t.Transform(nx.ctx, in, g)
		go func() {
			for i := range input {
				// For enders, we don't report output unless there has been an error
				if i.Err() != nil {
					// this nonsense is necessary because channels don't support inheritance
					result <- i
				}
			}
			close(result)
		}()
		return result
	}
	return &next
}

func FromChan(in chan OutResult) Pipeline {
	x := IterProducer{
		Iterator: func() (s interface{}, e error) {
			res, ok := <-in
			if !ok {
				return "", iterator.Done
			}
			return res.Output(), res.Err()
		},
	}

	return &Impl{
		exec: func(_ chan OutResult, sp *Impl) chan OutResult {
			return x.Start(sp.ctx, sp.bufferSize)
		},
		ctx: context.Background(), // this is just the default
	}

}

func FromIter(x IterProducer) Pipeline {
	return &Impl{
		exec: func(_ chan OutResult, sp *Impl) chan OutResult {
			return x.Start(sp.ctx, sp.bufferSize)
		},
		ctx: context.Background(), // this is just the default
	}
}

func From(f func() (interface{}, error)) Pipeline {
	x := IterProducer{
		Iterator: f,
	}
	return FromIter(x)
}
