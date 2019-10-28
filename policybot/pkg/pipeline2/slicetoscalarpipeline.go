package pipeline

import (
	"context"
	"github.com/eapache/channels"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/api/iterator"
)

type SliceToScalarPipelineImpl struct {
	ctx         context.Context
	bufferSize  int
	parallelism int
	priorStep   ScalarPipeline
	// exec acts like a receiver function, but is late bound
	exec         func(chan OutResult, *SliceToScalarPipelineImpl) chan OutResult
	errorHandler func(error)
}

func (sp *SliceToScalarPipelineImpl) TransformToArray(func(interface{}) ([]interface{}, error)) SlicePipeline {
	panic("implement me")
}

// TODO: the With and On functions need clarification around chaining
func (sp *SliceToScalarPipelineImpl) WithContext(ctx context.Context) ScalarPipeline {
	sp.ctx = ctx
	return sp
}

func (sp *SliceToScalarPipelineImpl) WithBuffer(i int) ScalarPipeline {
	sp.bufferSize = i
	return sp
}

func (sp *SliceToScalarPipelineImpl) WithParallelism(i int) ScalarPipeline {
	sp.parallelism = i
	return sp
}

type StringPipelineEnder struct {
	ctx          context.Context
	bufferSize   int
	parallelism  int
	priorStep    ScalarPipeline
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

func (sp *SliceToScalarPipelineImpl) Go() chan OutResult {
	var priorOut chan OutResult
	if sp.priorStep != nil {
		priorOut = sp.priorStep.Go() // TODO: handle errors here?
	}
	return sp.exec(priorOut, sp)
}

func (sp *SliceToScalarPipelineImpl) OnError(f func(error)) ScalarPipeline {
	sp.errorHandler = f
	return sp
}

func (sp *SliceToScalarPipelineImpl) makeChild() SliceToScalarPipelineImpl {
	child := *sp
	child.priorStep = sp
	child.exec = nil
	return child
}

func (sp *SliceToScalarPipelineImpl) Batch(size int) SlicePipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *SliceToScalarPipelineImpl) (out chan OutResult) {
		wrapper := channels.Wrap(in)
		f := channels.NewBatchingChannel(channels.BufferCap(size))
		channels.Pipe(wrapper, f)
		go func() {
			for x := range f.Out() {
				var outSlice []interface{}
				var errSlice error
				switch t := x.(type) {
				case []OutResult:
					for _, i := range t {
						if i.Err() == nil {
							outSlice = append(outSlice, i.Output())
						} else {
							errSlice = multierror.Append(errSlice, i.Err())
						}
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

func (sp *SliceToScalarPipelineImpl) Transform(f func(result interface{}) (interface{}, error)) ScalarPipeline {
	next := sp.makeChild()
	next.exec = func(in chan OutResult, nx *SliceToScalarPipelineImpl) chan OutResult {
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

func (sp *SliceToScalarPipelineImpl) To(f func(result interface{}) error) PipelineEnd {
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
