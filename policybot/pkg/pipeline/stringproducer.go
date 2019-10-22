package pipeline

import (
	"context"
	"errors"

	"google.golang.org/api/iterator"
)

var Skip = errors.New("This iteration should be skipped")

type StringIterProducer struct {
	Setup    func() error
	Iterator func() (string, error)
}

type StringInResult interface {
	Input() string
	Err() error
}

type StringOutResult interface {
	Err() error
	Output() string
}

type StringInOutResult interface {
	Input() string
	Err() error
	Output() string
}

type simpleOut struct {
	err error
	out string
}

type simpleInOut struct {
	simpleOut
	in string
}

func (s simpleInOut) Input() string {
	return s.in
}

func (s simpleOut) Err() error {
	return s.err
}

func (s simpleOut) Output() string {
	return s.out
}

func (sp *StringIterProducer) Start(ctx context.Context, bufferSize int) (resultChan chan StringOutResult) {
	resultChan = make(chan StringOutResult, bufferSize)
	go func() {
		defer close(resultChan)
		err := sp.Setup()
		if err != nil {
			resultChan <- simpleOut{err, ""}
			return
		}
		for {
			select {
			case <-ctx.Done():
				// attempt a non-blocking write of ctx.Error()
				select {
				case resultChan <- simpleOut{ctx.Err(), ""}:
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
					resultChan <- simpleOut{err, ""}
				} else {
					resultChan <- simpleOut{nil, result}
				}
			}
		}
	}()
	return
}

func BuildSlice(resultChan chan StringOutResult) ([]string, error) {
	var items = make([]string, 0)
	for result := range resultChan {
		if result.Err() != nil {
			return nil, result.Err()
		}
		items = append(items, result.Output())
	}
	return items, nil
}

func BuildProducer(ctx context.Context, input []string) chan StringOutResult {
	var count int
	sp := &StringIterProducer{
		Setup: func() error {
			return nil
		},
		Iterator: func() (string, error) {
			if count > len(input)-1 {
				return "", iterator.Done
			}
			count++
			return input[count-1], nil
		},
	}
	return sp.Start(ctx, 1)
}
