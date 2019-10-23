package pipeline

import (
	"context"

	"google.golang.org/api/iterator"
)

type StringPipeline struct {
	ctx         context.Context
	bufferSize  int
	parallelism int
	priorStep   Pipeline
	// exec acts like a receiver function, but is late bound
	exec         func(chan StringOutResult, *StringPipeline) chan StringOutResult
	errorHandler func(error)
}

// TODO: the With and On functions need clarification around chaining
func (sp *StringPipeline) WithContext(ctx context.Context) Pipeline {
	sp.ctx = ctx
	return sp
}

func (sp *StringPipeline) WithBuffer(i int) Pipeline {
	sp.bufferSize = i
	return sp
}

func (sp *StringPipeline) WithParallelism(i int) Pipeline {
	sp.parallelism = i
	return sp
}

type StringPipelineEnder struct {
	ctx          context.Context
	bufferSize   int
	parallelism  int
	priorStep    Pipeline
	exec         func(chan StringOutResult, *StringPipelineEnder) chan StringInResult
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

func (spe *StringPipelineEnder) Go() chan StringInResult {
	// Ender's always have priors
	priorOut := spe.priorStep.Go() // TODO: handle errors here?
	return spe.exec(priorOut, spe)
}

func (sp *StringPipeline) Go() chan StringOutResult {
	var priorOut chan StringOutResult
	if sp.priorStep != nil {
		priorOut = sp.priorStep.Go() // TODO: handle errors here?
	}
	return sp.exec(priorOut, sp)
}

func (sp *StringPipeline) OnError(f func(error)) Pipeline {
	sp.errorHandler = f
	return sp
}

func (sp *StringPipeline) makeChild() StringPipeline {
	child := *sp
	child.priorStep = sp
	child.exec = nil
	return child
}

func (sp *StringPipeline) Transform(f func(result string) (string, error)) Pipeline {
	next := sp.makeChild()
	next.exec = func(in chan StringOutResult, nx *StringPipeline) chan StringOutResult {
		result := make(chan StringOutResult, sp.bufferSize)
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

func (sp *StringPipeline) To(f func(result string) error) PipelineEnd {
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
	next.exec = func(in chan StringOutResult, nx *StringPipelineEnder) chan StringInResult {
		result := make(chan StringInResult, sp.bufferSize)
		g := func(result string) (string, error) { return "", f(result) }
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

type Pipeline interface {
	Transform(func(string) (string, error)) Pipeline
	To(func(string) error) PipelineEnd
	Go() chan StringOutResult
	OnError(func(error)) Pipeline
	WithContext(ctx context.Context) Pipeline
	WithBuffer(int) Pipeline
	WithParallelism(int) Pipeline
}

type PipelineEnd interface {
	Go() chan StringInResult
	OnError(func(error)) PipelineEnd
	WithContext(ctx context.Context) PipelineEnd
	WithBuffer(int) PipelineEnd
	WithParallelism(int) PipelineEnd
}

func FromChan(in chan StringOutResult) Pipeline {
	x := StringIterProducer{
		Iterator: func() (s string, e error) {
			select {
			case res, ok := <-in:
				if !ok {
					return "", iterator.Done
				}
				return res.Output(), res.Err()
			}
		},
	}

	return &StringPipeline{
		exec: func(_ chan StringOutResult, sp *StringPipeline) chan StringOutResult {
			return x.Start(sp.ctx, sp.bufferSize)
		},
		ctx: context.Background(), // this is just the default
	}

}

func From(f func() (string, error)) Pipeline {
	x := StringIterProducer{
		Iterator: f,
	}
	return &StringPipeline{
		exec: func(_ chan StringOutResult, sp *StringPipeline) chan StringOutResult {
			return x.Start(sp.ctx, sp.bufferSize)
		},
		ctx: context.Background(), // this is just the default
	}
}
