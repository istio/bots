package pipeline

import (
	"context"
	"reflect"
	"sync"
)

type ScalarToSlicePipelineImpl struct {
	ctx         context.Context
	bufferSize  int
	parallelism int
	priorStep   ScalarPipeline
	// exec acts like a receiver function, but is late bound
	exec         func(chan OutResult, *PipelineImpl) chan []OutResult
	errorHandler func(error)
}

func (a ScalarToSlicePipelineImpl) Transform(func(interface{}) ([]interface{}, error)) SlicePipeline {
	panic("implement me")
}

func (a ScalarToSlicePipelineImpl) To(func([]interface{}) error) PipelineEnd {
	panic("implement me")
}

func (a ScalarToSlicePipelineImpl) Go() chan []OutResult {
	panic("implement me")
}

func (sp ScalarToSlicePipelineImpl) makeChild() ScalarToSlicePipelineImpl {

}

func (sp *ScalarToSlicePipelineImpl) Expand() ScalarPipeline {
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

func (a ScalarToSlicePipelineImpl) OnError(func(error)) SlicePipeline {
	panic("implement me")
}

func (a ScalarToSlicePipelineImpl) WithContext(ctx context.Context) SlicePipeline {
	panic("implement me")
}

func (a ScalarToSlicePipelineImpl) WithBuffer(int) SlicePipeline {
	panic("implement me")
}

func (a ScalarToSlicePipelineImpl) WithParallelism(int) SlicePipeline {
	panic("implement me")
}

func (a ScalarToSlicePipelineImpl) TransformToScalar(func([]interface{}) (interface{}, error)) ScalarPipeline {
	panic("implement me")
}

