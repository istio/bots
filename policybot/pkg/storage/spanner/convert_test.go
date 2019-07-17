// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spanner

import (
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

var (
	nonNullString        = "nonNull"
	nonNullInt64   int64 = 4321
	nonNullBool          = false
	nonNullFloat64       = 5432.1
	now                  = time.Now().UTC() // Spanner returns UTC time.
	nonNullTime          = now.Add(time.Hour)
)

func TestRowToStruct(t *testing.T) {
	tests := []struct {
		name     string
		row      *spanner.Row
		expected interface{}
	}{
		{
			"nullString",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{"test", spanner.NullString{}}),
			struct {
				ExistingCol string
				NewCol      *string
			}{"test", nil},
		},
		{
			"nonNullString",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{"test", spanner.NullString{Valid: true, StringVal: nonNullString}}),
			struct {
				ExistingCol string
				NewCol      *string
			}{"test", &nonNullString},
		},
		{
			"nullInt64",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{1234, spanner.NullInt64{}}),
			struct {
				ExistingCol int64
				NewCol      *int64
			}{1234, nil},
		},
		{
			"nonNullInt64",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{1234, spanner.NullInt64{Valid: true, Int64: nonNullInt64}}),
			struct {
				ExistingCol int64
				NewCol      *int64
			}{1234, &nonNullInt64},
		},
		{
			"nullBool",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{true, spanner.NullBool{}}),
			struct {
				ExistingCol bool
				NewCol      *bool
			}{true, nil},
		},
		{
			"nonNullBool",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{true, spanner.NullBool{Valid: true, Bool: false}}),
			struct {
				ExistingCol bool
				NewCol      *bool
			}{true, &nonNullBool},
		},
		{
			"nullFloat64",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{1234.5, spanner.NullFloat64{}}),
			struct {
				ExistingCol float64
				NewCol      *float64
			}{1234.5, nil},
		},
		{
			"nonNullFloat64",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{1234.5, spanner.NullFloat64{Valid: true, Float64: nonNullFloat64}}),
			struct {
				ExistingCol float64
				NewCol      *float64
			}{1234.5, &nonNullFloat64},
		},
		{
			"nullTime",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{now, spanner.NullTime{}}),
			struct {
				ExistingCol time.Time
				NewCol      *time.Time
			}{now, nil},
		},
		{
			"nonNullTime",
			newRow(
				t,
				[]string{"ExistingCol", "NewCol"},
				[]interface{}{now, spanner.NullTime{Valid: true, Time: nonNullTime}}),
			struct {
				ExistingCol time.Time
				NewCol      *time.Time
			}{now, &nonNullTime},
		},
	}

	for _, test := range tests {
		out := reflect.New(reflect.TypeOf(test.expected)).Interface()
		if err := rowToStruct(test.row, out); err != nil {
			t.Errorf("%s: error converting row to native struct: %v", test.name, err)
		} else {
			actual := reflect.ValueOf(out).Elem().Interface()
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("%s: converting row to native struct resulted in %v, but expected %v",
					test.name, actual, test.expected)
			}
		}
	}
}

func newRow(t *testing.T, names []string, vals []interface{}) *spanner.Row {
	row, err := spanner.NewRow(names, vals)
	if err != nil {
		t.Errorf("error creating row with names %v and vals %v", names, vals)
		return nil
	}
	return row
}
