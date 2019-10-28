package pipeline

import (
	"context"
)

//type Pipeline interface {
//	OnError(func(error)) SlicePipeline
//	WithContext(ctx context.Context) SlicePipeline
//	WithBuffer(int) SlicePipeline
//	WithParallelism(int) SlicePipeline
//}

type ScalarPipeline interface {

	Transform(func(interface{}) (interface{}, error)) ScalarPipeline
	// this section is super tricky.  technically, the type returned by the parameter function
	// must be a Slice, but since []X is not a []interface{}, the only way to allow reasonable returns
	// is to set the return type to interface{} and require that the return is a slice at runtime.
	// Non-Slice return values will be cast to Slices of len 1.
	TransformToSlice(func(interface{}) (interface{}, error)) SlicePipeline
	To(func(interface{}) error) PipelineEnd
	Batch(size int) ScalarPipeline
	Go() chan OutResult
	Expand() ScalarPipeline

	OnError(func(error)) ScalarPipeline
	WithContext(ctx context.Context) ScalarPipeline
	WithBuffer(int) ScalarPipeline
	WithParallelism(int) ScalarPipeline
}

//type ScalarInPipeline interface {
//	Pipeline
//}

type SlicePipeline interface {
	Transform(func([]interface{}) (interface{}, error)) SlicePipeline
	TransformToScalar(func([]interface{}) (interface{}, error)) ScalarPipeline
	Expand() ScalarPipeline
	To(func([]interface{}) error) PipelineEnd
	Go() chan []OutResult

	OnError(func(error)) SlicePipeline
	WithContext(ctx context.Context) SlicePipeline
	WithBuffer(int) SlicePipeline
	WithParallelism(int) SlicePipeline
}
//
//type SliceInPipeline interface {
//	Pipeline
//}
//
//type ScalarPipeline interface {
//	Pipeline
//}
//
//type SliceToScalarPipeline interface {
//	Transform(func([]interface{}) ([]interface{}, error)) SlicePipeline
//	To(func([]interface{}) error) PipelineEnd
//	Go() chan []OutResult
//	Expand() ScalarPipeline
//	OnError(func(error)) SlicePipeline
//	WithContext(ctx context.Context) SlicePipeline
//	WithBuffer(int) SlicePipeline
//	WithParallelism(int) SlicePipeline
//	TransformToScalar(func([]interface{}) (interface{}, error)) ScalarPipeline
//}
//
//type SlicePipeline interface {
//	Transform(func([]interface{}) ([]interface{}, error)) SlicePipeline
//	To(func([]interface{}) error) PipelineEnd
//	Expand() ScalarPipeline
//	TransformToScalar(func([]interface{}) (interface{}, error)) ScalarPipeline
//}

type PipelineEnd interface {
	WithContext(ctx context.Context) PipelineEnd
	WithBuffer(int) PipelineEnd
	WithParallelism(int) PipelineEnd
	OnError(func(error)) PipelineEnd
	Go() chan InResult
}