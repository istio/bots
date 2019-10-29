package pipeline

import (
	"context"
	"reflect"
	"sync"

	"github.com/eapache/channels"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/api/iterator"
)

type PipelineImpl struct {
	ctx         context.Context
	bufferSize  int
	parallelism int
	priorStep   Pipeline
	// exec acts like a receiver function, but is late bound
	exec         func(chan OutResult, *PipelineImpl) chan OutResult
	errorHandler func(error)
}

func (sp *PipelineImpl) Expand() Pipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *PipelineImpl) chan OutResult {
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
func (sp *PipelineImpl) WithContext(ctx context.Context) Pipeline {
	sp.ctx = ctx
	return sp
}

func (sp *PipelineImpl) WithBuffer(i int) Pipeline {
	sp.bufferSize = i
	return sp
}

func (sp *PipelineImpl) WithParallelism(i int) Pipeline {
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

func (spe *StringPipelineEnder) WithContext(ctx context.Context) PipelineEnd {
	spe.ctx = ctx
	return spe
}

func (spe *StringPipelineEnder) WithBuffer(i int) PipelineEnd {
	spe.bufferSize = i
	return spe
}

func (spe *StringPipelineEnder) WithParallelism(i int) PipelineEnd {
	spe.parallelism = i
	return spe
}

func (spe *StringPipelineEnder) OnError(f func(error)) PipelineEnd {
	spe.errorHandler = f
	return spe
}

func (spe *StringPipelineEnder) Go() chan InResult {
	// Ender's always have priors
	priorOut := spe.priorStep.Go() // TODO: handle errors here?
	return spe.exec(priorOut, spe)
}

func (sp *PipelineImpl) Go() chan OutResult {
	var priorOut chan OutResult
	if sp.priorStep != nil {
		priorOut = sp.priorStep.Go() // TODO: handle errors here?
	}
	return sp.exec(priorOut, sp)
}

func (sp *PipelineImpl) OnError(f func(error)) Pipeline {
	sp.errorHandler = f
	return sp
}

func (sp *PipelineImpl) makeChild() PipelineImpl {
	child := *sp
	child.priorStep = sp
	child.exec = nil
	return child
}

func (sp *PipelineImpl) Batch(size int) Pipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *PipelineImpl) (out chan OutResult) {
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

func (sp *PipelineImpl) Transform(f func(result interface{}) (interface{}, error)) Pipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *PipelineImpl) chan OutResult {
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

func (sp *PipelineImpl) To(f func(result interface{}) error) PipelineEnd {
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
				// this nonsense is necessary because channels don't support inheritance
				result <- i
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
			select {
			case res, ok := <-in:
				if !ok {
					return "", iterator.Done
				}
				return res.Output(), res.Err()
			}
		},
	}

	return &PipelineImpl{
		exec: func(_ chan OutResult, sp *PipelineImpl) chan OutResult {
			return x.Start(sp.ctx, sp.bufferSize)
		},
		ctx: context.Background(), // this is just the default
	}

}

func FromIter(x IterProducer) Pipeline {
	return &PipelineImpl{
		exec: func(_ chan OutResult, sp *PipelineImpl) chan OutResult {
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
