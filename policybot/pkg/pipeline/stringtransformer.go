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

func (slt *StringLogTransformer) Transform(ctx context.Context, in chan StringOutResult, transformer func(string) (string, error)) chan StringInOutResult {
	return StringTransform(ctx, slt.Parallelism, slt.BufferSize, in, transformer, slt.ErrHandler)
}

// StringTransform consumes a channel of string or errors and produces a channel of string or errors.
// All incoming errors will be passed to the error handler, which returns nothing.
// All incoming errorless strings will be passed to the tranform function, whose results will be written to the
// resulting channel *unless* the returned error is Skip, in which case that element is skipped.
func StringTransform(ctx context.Context, parallelism int, bufferSize int, in chan StringOutResult, transformer func(string) (string, error), errhandler func(error)) chan StringInOutResult {
	// TODO: can we have a channel factory to do this?
	outChan := make(chan StringInOutResult)
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
				//do stuff, write to out maybe
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
				if err == Skip {
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
