package pipeline

import (
	"context"
	"errors"

	"google.golang.org/api/iterator"
)

var Skip = errors.New("This iteration should be skipped")

type StringProducer struct {
	Setup    func() error
	Iterator func() (string, error)
}

type StringReslt struct {
	Err    error
	Result string
}

func (sp *StringProducer) Start(ctx context.Context, bufferSize int) (resultChan chan StringReslt) {
	resultChan = make(chan StringReslt, bufferSize)
	go func() {
		defer close(resultChan)
		err := sp.Setup()
		if err != nil {
			resultChan <- StringReslt{err, ""}
			return
		}
		for {
			select {
			case <-ctx.Done():
				// attempt a non-blocking write of ctx.Error()
				select {
				case resultChan <- StringReslt{ctx.Err(), ""}:
				default:
					return
				}
			default:
				result, err := sp.Iterator()
				if err != nil {
					if err == Skip {
						continue
					}
					if err == iterator.Done {
						return
					}
					resultChan <- StringReslt{err, ""}
				} else {
					resultChan <- StringReslt{nil, result}
				}
			}
		}
	}()
	return
}

func BuildSlice(resultChan chan StringReslt) ([]string, error) {
	var items = make([]string, 0)
	for result := range resultChan {
		if result.Err != nil {
			return nil, result.Err
		}
		items = append(items, result.Result)
	}
	return items, nil
}
