// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build js

package sqlite_wasm_js

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

type Rows struct {
	ctx         context.Context
	columns     []string
	columnTypes []string
	closeStmt   bool
	*Stmt
}

var (
	_ driver.Rows = &Rows{}
	//_ driver.RowsNextResultSet              = &Rows{}
	_ driver.RowsColumnTypeScanType         = &Rows{}
	_ driver.RowsColumnTypeDatabaseTypeName = &Rows{}
	//_ driver.RowsColumnTypeLength = &Rows{}
	//_ driver.RowsColumnTypeNullable = &Rows{}
	//_ driver.RowsColumnTypePrecisionScale = &Rows{}
)

func (r *Rows) Columns() []string {
	return r.columns
}

func (r *Rows) Close() error {
	if r.closeStmt {
		return r.Stmt.Close()
	}
	return r.reset(r.ctx)
}

func parseStrOrNumber(val js.Value) (int64, error) {
	switch val.Type() {
	case js.TypeNumber:
		return int64(val.Int()), nil
	case js.TypeString:
		return strconv.ParseInt(val.String(), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected JS type %s for integer", val.Type().String())
	}
}

func (r *Rows) scanColumn(index int) (driver.Value, error) {
	columnType := r.d.CAPI.Call("sqlite3_column_type", r.cptr, index).Int()
	switch columnType {
	case SQLITE_INTEGER:
		return parseStrOrNumber(r.d.Meow.Call("read_int64_column", r.cptr, index))
	case SQLITE_FLOAT:
		return r.d.CAPI.Call("sqlite3_column_double", r.cptr, index).Float(), nil
	case SQLITE_TEXT:
		// TODO use blob for strings too? (especially if the direct sqlite -> go heap transfer can be done)
		return r.d.CAPI.Call("sqlite3_column_text", r.cptr, index).String(), nil
	case SQLITE_BLOB:
		ptr := r.d.CAPI.Call("sqlite3_column_blob", r.cptr, index).Int()
		length := r.d.CAPI.Call("sqlite3_column_bytes", r.cptr, index).Int()
		if length == 0 {
			return []byte{}, nil
		}
		heap := r.d.WASM.Call("heap8u").Call("subarray", ptr, ptr+length)
		dst := make([]byte, length)
		n := js.CopyBytesToGo(dst, heap)
		if n != len(dst) {
			panic(fmt.Sprintf("failed to copy %d bytes from sqlite3 heap to go heap, only copied %d bytes", len(dst), n))
		}
		return dst, nil
	case SQLITE_NULL:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported column type %d", columnType)
	}
}

const sqliteTimeFormat = "2006-01-02 15:04:05.999999999-07:00"

func (r *Rows) Next(dest []driver.Value) error {
	if len(dest) != len(r.columns) {
		return fmt.Errorf("can't scan %d columns into %d values", len(r.columns), len(dest))
	}
	hasMoreRows, err := r.step(r.ctx)
	if err != nil {
		return err
	} else if !hasMoreRows {
		return io.EOF
	}
	for i := range dest {
		dest[i], err = r.scanColumn(i)
		if err != nil {
			return fmt.Errorf("failed to scan %s: %w", r.columns[i], err)
		}
		if r.columnTypes[i] == "timestamp" {
			destStr, _ := dest[i].(string)
			if destStr == "" {
				dest[i] = time.Time{}
			} else {
				dest[i], err = time.ParseInLocation(sqliteTimeFormat, destStr, time.UTC)
				if err != nil {
					return fmt.Errorf("failed to parse timestamp %v: %w", dest[i], err)
				}
			}
		}
	}
	return nil
}

//func (r *Rows) HasNextResultSet() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (r *Rows) NextResultSet() error {
//	hasMoreRows, err := r.step(r.ctx)
//	if err != nil {
//		return err
//	} else if !hasMoreRows {
//		return io.EOF
//	}
//	return nil
//}

func (r *Rows) ColumnTypeScanType(index int) reflect.Type {
	switch r.d.CAPI.Call("sqlite3_column_type", r.cptr, index).Int() {
	case SQLITE_INTEGER:
		return reflect.TypeOf(int64(0))
	case SQLITE_FLOAT:
		return reflect.TypeOf(float64(0))
	case SQLITE_TEXT:
		if r.columnTypes[index] == "timestamp" {
			return reflect.TypeOf(time.Time{})
		}
		return reflect.TypeOf("")
	case SQLITE_BLOB:
		return reflect.TypeOf([]byte{})
	case SQLITE_NULL:
		fallthrough
	default:
		return nil
	}
}

func (r *Rows) ColumnTypeDatabaseTypeName(index int) string {
	return strings.ToUpper(r.columnTypes[index])
}
