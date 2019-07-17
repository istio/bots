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
	"fmt"
	"reflect"
	"time"

	"cloud.google.com/go/spanner"
)

func readColumn(row *spanner.Row, f interface{}, sf reflect.StructField, t reflect.Type) error {
	err := row.ColumnByName(sf.Name, f)
	if err != nil {
		return fmt.Errorf("rowToStruct: error deserializing into field %s in type %s: %v",
			sf.Name, t.Name(), err)
	}
	return nil
}

func setValue(f reflect.Value, valid bool, val interface{}) {
	if valid {
		f.Set(reflect.ValueOf(val))
	} else {
		f.Set(reflect.Zero(f.Type()))
	}
}

// rowToStruct converts a Spanner row into a storage struct. The Spanner
// library requires use of Spanner-specific types for nullable columns, which
// we don't want in our abstract layer. This function allows the use of
// pointers to indicate null primitive columns. Null arrays are already handled
// by the Spanner code. This code will return an error if a struct does not
// have a corresponding column in the table, but will ignore columns that are
// in Spanner and not in the struct.
func rowToStruct(row *spanner.Row, s interface{}) error {
	ptrType := reflect.TypeOf(s)
	if ptrType.Kind() != reflect.Ptr || ptrType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("rowToStruct: type %v must be a pointer to a struct", ptrType)
	}
	structType := ptrType.Elem()
	structVal := reflect.ValueOf(s).Elem()
	for i := 0; i < structType.NumField(); i++ {
		fieldInfo := structType.Field(i)
		if fieldInfo.PkgPath != "" { // field is unexported
			continue
		}
		switch structVal.Field(i).Interface().(type) {
		case *string:
			ns := spanner.NullString{}
			if err := readColumn(row, &ns, fieldInfo, structType); err != nil {
				return err
			}
			setValue(structVal.Field(i), ns.Valid, &ns.StringVal)
		case *int64:
			ni := spanner.NullInt64{}
			if err := readColumn(row, &ni, fieldInfo, structType); err != nil {
				return err
			}
			setValue(structVal.Field(i), ni.Valid, &ni.Int64)
		case *bool:
			nb := spanner.NullBool{}
			if err := readColumn(row, &nb, fieldInfo, structType); err != nil {
				return err
			}
			setValue(structVal.Field(i), nb.Valid, &nb.Bool)
		case *float64:
			nf := spanner.NullFloat64{}
			if err := readColumn(row, &nf, fieldInfo, structType); err != nil {
				return err
			}
			setValue(structVal.Field(i), nf.Valid, &nf.Float64)
		case *time.Time:
			nt := spanner.NullTime{}
			if err := readColumn(row, &nt, fieldInfo, structType); err != nil {
				return err
			}
			setValue(structVal.Field(i), nt.Valid, &nt.Time)
		default:
			// Use the default behavior for non-nullable or non-primitive columns.
			row.ColumnByName(fieldInfo.Name, structVal.Field(i).Addr().Interface())
		}
	}
	return nil
}
