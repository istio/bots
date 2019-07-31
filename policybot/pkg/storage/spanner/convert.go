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
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

func readColumn(row *spanner.Row, columns []string, f interface{}, sf reflect.StructField, t reflect.Type) error {
	for i, column := range columns {
		if strings.EqualFold(column, sf.Name) {
			err := row.Column(i, f)
			if err != nil {
				return fmt.Errorf("readColumn: error deserializing into field %s in type %s: %v",
					sf.Name, t.Name(), err)
			}
			return nil
		}
	}
	return fmt.Errorf("readColumn: could not find field %s in Spanner row with columns: %v", sf.Name, columns)
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
	columns := row.ColumnNames()
	for i := 0; i < structType.NumField(); i++ {
		fieldInfo := structType.Field(i)
		if fieldInfo.PkgPath != "" { // field is unexported
			continue
		}
		switch structVal.Field(i).Interface().(type) {
		case *string:
			ns := spanner.NullString{}
			if err := readColumn(row, columns, &ns, fieldInfo, structType); err != nil {
				return fmt.Errorf("rowToStruct: error reading column %s: %v", fieldInfo.Name, err)
			}
			setValue(structVal.Field(i), ns.Valid, &ns.StringVal)
		case *int64:
			ni := spanner.NullInt64{}
			if err := readColumn(row, columns, &ni, fieldInfo, structType); err != nil {
				return fmt.Errorf("rowToStruct: error reading column %s: %v", fieldInfo.Name, err)
			}
			setValue(structVal.Field(i), ni.Valid, &ni.Int64)
		case *bool:
			nb := spanner.NullBool{}
			if err := readColumn(row, columns, &nb, fieldInfo, structType); err != nil {
				return fmt.Errorf("rowToStruct: error reading column %s: %v", fieldInfo.Name, err)
			}
			setValue(structVal.Field(i), nb.Valid, &nb.Bool)
		case *float64:
			nf := spanner.NullFloat64{}
			if err := readColumn(row, columns, &nf, fieldInfo, structType); err != nil {
				return fmt.Errorf("rowToStruct: error reading column %s: %v", fieldInfo.Name, err)
			}
			setValue(structVal.Field(i), nf.Valid, &nf.Float64)
		case *time.Time:
			nt := spanner.NullTime{}
			if err := readColumn(row, columns, &nt, fieldInfo, structType); err != nil {
				return fmt.Errorf("rowToStruct: error reading column %s: %v", fieldInfo.Name, err)
			}
			setValue(structVal.Field(i), nt.Valid, &nt.Time)
		default:
			// Use the default behavior for non-nullable or non-primitive columns.
			if err := readColumn(row, columns, structVal.Field(i).Addr().Interface(), fieldInfo, structType); err != nil {
				return fmt.Errorf("rowToStruct: error reading column %s: %v", fieldInfo.Name, err)
			}
		}
	}
	return nil
}

// insertStruct returns a Spanner Mutation object representing a row
// insert or update for a struct. This converts pointer types in the struct
// to the Spanner nullable types where necessary.
func insertStruct(table string, s interface{}) (*spanner.Mutation, error) {
	cols, vals, err := exportStruct(s)
	if err != nil {
		return nil, err
	}
	return spanner.Insert(table, cols, vals), nil
}

// insertOrUpdateStruct returns a Spanner Mutation object representing a row
// insert or update for a struct. This converts pointer types in the struct
// to the Spanner nullable types where necessary.
func insertOrUpdateStruct(table string, s interface{}) (*spanner.Mutation, error) {
	cols, vals, err := exportStruct(s)
	if err != nil {
		return nil, err
	}
	return spanner.InsertOrUpdate(table, cols, vals), nil
}

// exportStruct converts a struct into column and value slices, converting
// pointer types into Spanner nullable types where necessary.
func exportStruct(s interface{}) ([]string, []interface{}, error) {
	structType := reflect.TypeOf(s)
	structVal := reflect.ValueOf(s)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
		structVal = structVal.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("exportStruct: type %v must be a struct or pointer to a struct", structType)
	}

	cols := make([]string, 0, structType.NumField())
	vals := make([]interface{}, 0, structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		fieldInfo := structType.Field(i)
		tagInfo := fieldInfo.Tag.Get("spanner")
		if fieldInfo.PkgPath != "" || tagInfo == "-" { // field is unexported or ignored
			continue
		}
		if tagInfo == "" {
			tagInfo = fieldInfo.Name
		}
		cols = append(cols, tagInfo)
		switch f := structVal.Field(i).Interface().(type) {
		case *string:
			ns := spanner.NullString{}
			if ns.Valid = f != nil; ns.Valid {
				ns.StringVal = *f
			}
			vals = append(vals, ns)
		case *int64:
			ni := spanner.NullInt64{}
			if ni.Valid = f != nil; ni.Valid {
				ni.Int64 = *f
			}
			vals = append(vals, ni)
		case *bool:
			nb := spanner.NullBool{}
			if nb.Valid = f != nil; nb.Valid {
				nb.Bool = *f
			}
			vals = append(vals, nb)
		case *float64:
			nf := spanner.NullFloat64{}
			if nf.Valid = f != nil; nf.Valid {
				nf.Float64 = *f
			}
			vals = append(vals, nf)
		case *time.Time:
			nt := spanner.NullTime{}
			if nt.Valid = f != nil; nt.Valid {
				nt.Time = *f
			}
			vals = append(vals, nt)
		default:
			// Use the default behavior for non-nullable or non-primitive columns.
			vals = append(vals, structVal.Field(i).Interface())
		}
	}
	return cols, vals, nil
}
