package pipeline

import (
	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type ObjectProducer struct {
	Setup    func() error
	Iterator *storage.ObjectIterator
}

func (op *ObjectProducer) Start(ctx context.Context, bufferSize int) (resultChan chan *storage.ObjectAttrs, errChan chan error) {
	resultChan = make(chan *storage.ObjectAttrs, bufferSize)
	errChan = make(chan error, bufferSize)
	go func() {
		defer close(resultChan)
		defer close(errChan)
		err := op.Setup()
		if err != nil {
			errChan <- err
			return
		}
		for {
			select {
			case <-ctx.Done():
				// attempt a non-blocking write of ctx.Error()
				select {
				case errChan <- ctx.Err():
				default:
					return
				}
			default:
				result, err := op.Iterator.Next()
				if err != nil {
					if err == iterator.Done {
						return
					}
					errChan <- err
				} else {
					resultChan <- result
				}
			}
		}
	}()
	return
}
