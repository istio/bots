package pipeline

import (
	"context"
)
type ScalarPipeline interface {

	Transform(func(interface{}) (interface{}, error)) ScalarPipeline
	To(func(interface{}) error) PipelineEnd
	Batch(size int) ScalarPipeline
	Go() chan OutResult
	Expand() ScalarPipeline

	OnError(func(error)) ScalarPipeline
	WithContext(ctx context.Context) ScalarPipeline
	WithBuffer(int) ScalarPipeline
	WithParallelism(int) ScalarPipeline
}

type PipelineEnd interface {
	WithContext(ctx context.Context) PipelineEnd
	WithBuffer(int) PipelineEnd
	WithParallelism(int) PipelineEnd
	OnError(func(error)) PipelineEnd
	Go() chan InResult
}