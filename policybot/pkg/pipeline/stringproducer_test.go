package pipeline

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"google.golang.org/api/iterator"
	"gotest.tools/assert"
)

func TestSetupError(t *testing.T) {
	expectedError := errors.New("fake error")
	sp := &StringProducer{
		Setup: func() error {
			return expectedError
		},
		Iterator: nil,
	}
	resultChan := sp.Start(context.TODO(), 1)
	result, ok := <-resultChan
	assert.Assert(t, ok)
	assert.ErrorType(t, result.Err, expectedError)
	result, ok = <-resultChan
	assert.Assert(t, !ok)
}

func TestSuccess(t *testing.T) {
	var things []int
	var count int

	sp := &StringProducer{
		Setup: func() error {
			things = []int{1, 2, 3, 4}
			return nil
		},
		Iterator: func() (string, error) {
			if count > len(things)-1 {
				return "", iterator.Done
			}
			count++
			return strconv.Itoa(things[count-1]), nil
		},
	}
	var resultCount int
	resultChan := sp.Start(context.TODO(), 1)
	for result := range resultChan {
		assert.NilError(t, result.Err)
		resultCount++
	}
	assert.Equal(t, resultCount, len(things))
}
