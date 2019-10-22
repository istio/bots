package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestSetupError(t *testing.T) {
	expectedError := errors.New("fake error")
	sp := &StringIterProducer{
		Setup: func() error {
			return expectedError
		},
		Iterator: nil,
	}
	resultChan := sp.Start(context.TODO(), 1)
	result, ok := <-resultChan
	assert.Assert(t, ok)
	assert.ErrorType(t, result.Err(), expectedError)
	result, ok = <-resultChan
	assert.Assert(t, !ok)
}

func TestFake(t *testing.T) {
	var a []string
	a = nil
	b := []string{"foo"}
	c := append(a, b...)
	assert.Assert(t, c != nil)
}

func TestTransform(t *testing.T) {
	var things []string
	for i := 0; i < 20; i++ {
		things = append(things, fmt.Sprintf("pathtopr/%d/somethingelse", i))
	}
	slt := StringLogTransformer{ErrHandler: func(e error) {
		t.Log(e)
	}}
	ctx := context.Background()
	sourceChan := BuildProducer(ctx, things)
	// our sample transform function returns only the part of the prpath that represents the pr number, and
	// only if the prnum is between high and low, inclusive
	resultChan := slt.Transform(ctx, sourceChan, func(prPath string) (prNum string, err error) {
		const high = 10
		const low = 7
		prParts := strings.Split(prPath, "/")
		if len(prParts) < 2 {
			err = errors.New("too few segments in pr path")
			return
		}
		prNumInt, err := strconv.Atoi(prParts[len(prParts)-2])
		if err != nil {
			return
		} else if prNumInt <= high && prNumInt >= low {
			prNum = prParts[len(prParts)-2]
			return
		}
		err = Skip
		return
	})
	for element := range resultChan {
		// no errors occur
		assert.NilError(t, element.Err())
		// zero is too low
		assert.Assert(t, element.Output() != "0")
		// eleven is too high
		assert.Assert(t, element.Output() != "11")
	}
}

func TestSuccess(t *testing.T) {
	things := []string{"1", "2", "3", "4"}
	resultChan := BuildProducer(context.Background(), things)
	var resultCount int
	for result := range resultChan {
		assert.NilError(t, result.Err())
		resultCount++
	}
	assert.Equal(t, resultCount, len(things))
}
