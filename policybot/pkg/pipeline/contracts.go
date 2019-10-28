package pipeline

import (
	"context"
)

type Pipeline interface {
	Transform(func(interface{}) (interface{}, error)) Pipeline
	To(func(interface{}) error) PipelineEnd
	Batch(size int) Pipeline
	Go() chan OutResult
	Expand() Pipeline

	OnError(func(error)) Pipeline
	WithContext(ctx context.Context) Pipeline
	WithBuffer(int) Pipeline
	WithParallelism(int) Pipeline
}

type PipelineEnd interface {
	WithContext(ctx context.Context) PipelineEnd
	WithBuffer(int) PipelineEnd
	WithParallelism(int) PipelineEnd
	OnError(func(error)) PipelineEnd
	Go() chan InResult
}
