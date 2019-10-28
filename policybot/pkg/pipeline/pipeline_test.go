package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/iterator"
	"gotest.tools/assert"
)

type testDataSource struct {
	index  int
	source []interface{}
	errMap map[int]error
}

func (ds *testDataSource) iterate() (interface{}, error) {
	i := ds.index
	ds.index++
	if err, ok := ds.errMap[i]; ok {
		return "", err
	} else if i >= len(ds.source) {
		return "", iterator.Done
	}
	return ds.source[i], nil
}

func TestPipeline(t *testing.T) {
	// This is a sample data source that emits numbers as text
	// it will skip emitting "two" to demonstrate that feature
	d := testDataSource{
		source: []interface{}{"zero", "one", "two", "three", "four", "five"},
		errMap: map[int]error{2: Skip, 5: errors.New("foo")},
	}
	// this is an async test, so if it hasn't finished in a minute, exit
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var errcount, rescount int
	// from our datasource, transform the text to describe piggies
	out := From(d.iterate).WithContext(ctx).WithBuffer(2).
		Transform(func(input interface{}) (s interface{}, e error) {
			if input == "one" {
				return input.(string) + " piggy", nil
			}
			return input.(string) + " piggies", nil
		}).OnError(func(e error) {
		errcount++
		assert.ErrorContains(t, e, "foo")
	}).WithParallelism(2).Go()

	for result := range out {
		rescount++
		fmt.Printf("checking result %v\n", result)
		assert.NilError(t, result.Err())
		assert.Assert(t, !strings.HasPrefix(result.Output().(string), "two"))
		assert.Assert(t, strings.Contains(result.Output().(string), "pig"))
	}
	assert.Equal(t, errcount, 1)
	assert.Equal(t, rescount, 4)
}
