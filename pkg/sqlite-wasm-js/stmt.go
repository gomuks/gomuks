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
	"reflect"
	"strconv"
	"strings"
	"syscall/js"
	"time"
	"unsafe"

	"go.mau.fi/util/exstrings"
)

type Stmt struct {
	d *Driver
	c *Conn

	cptr js.Value
}

var (
	_ driver.Stmt             = &Stmt{}
	_ driver.StmtExecContext  = &Stmt{}
	_ driver.StmtQueryContext = &Stmt{}
)

func (s *Stmt) Close() error {
	rc := s.d.CAPI.Call("sqlite3_finalize", s.cptr).Int()
	if rc != 0 {
		return s.d.MakeError(s.c, "sqlite3_finalize", rc)
	}
	return nil
}

func (s *Stmt) NumInput() int {
	return s.d.CAPI.Call("sqlite3_bind_parameter_count", s.cptr).Int()
}

var bigInt = js.Global().Get("BigInt")

const maxSafeJSInt = 1<<53 - 1
const minSafeJSInt = -maxSafeJSInt

func safeInt(val js.Value) int {
	if val.IsUndefined() {
		return 0
	}
	return val.Int()
}

func (s *Stmt) bindNil(_ context.Context, index int) error {
	rc := safeInt(s.d.CAPI.Call("sqlite3_bind_null", s.cptr, index))
	if rc != SQLITE_OK {
		return s.d.MakeError(s.c, "sqlite3_bind_null", rc)
	}
	return nil
}

func (s *Stmt) bindBytes(funcName string, index int, value []byte) js.Value {
	ptr := s.d.WASM.Call("alloc", max(len(value), 1))
	if len(value) > 0 {
		heap8u := s.d.WASM.Call("heap8u").Call("subarray", ptr.Int(), ptr.Int()+len(value))
		js.CopyBytesToJS(heap8u, value)
	}
	return s.d.CAPI.Call(funcName, s.cptr, index, ptr, len(value), SQLITE_WASM_DEALLOC)
}

func (s *Stmt) bindNonPointerValue(_ context.Context, index int, val any) error {
	var rc js.Value
	var funcName string
	switch typedVal := val.(type) {
	case string:
		funcName = "sqlite3_bind_text"
		rc = s.bindBytes(funcName, index, exstrings.UnsafeBytes(typedVal))
	case []byte:
		funcName = "sqlite3_bind_blob"
		rc = s.bindBytes(funcName, index, typedVal)
	case float32, float64:
		funcName = "sqlite3_bind_double"
		rc = s.d.CAPI.Call(funcName, s.cptr, index, typedVal)
	case bool:
		realVal := 0
		if typedVal {
			realVal = 1
		}
		funcName = "sqlite3_bind_int"
		rc = s.d.CAPI.Call(funcName, s.cptr, index, realVal)
	case int64:
		funcName = "sqlite3_bind_int64"
		var numberVal js.Value
		if typedVal > maxSafeJSInt || typedVal < minSafeJSInt {
			numberVal = bigInt.New(strconv.FormatInt(typedVal, 10))
		} else {
			numberVal = js.ValueOf(typedVal)
		}
		rc = s.d.CAPI.Call(funcName, s.cptr, index, numberVal)
	case uint64:
		funcName = "sqlite3_bind_int64"
		var numberVal js.Value
		if typedVal > maxSafeJSInt {
			numberVal = bigInt.New(strconv.FormatUint(typedVal, 10))
		} else {
			numberVal = js.ValueOf(typedVal)
		}
		rc = s.d.CAPI.Call(funcName, s.cptr, index, numberVal)
	default:
		return fmt.Errorf("unsupported type %T", val)
	}
	if realRC := rc.Int(); realRC != 0 {
		return s.d.MakeError(s.c, funcName, realRC)
	}
	return nil
}

func (s *Stmt) BindValue(ctx context.Context, val driver.NamedValue) error {
	index := val.Ordinal
	if val.Name != "" {
		index = s.d.CAPI.Call("sqlite3_bind_parameter_index", s.cptr, val.Name).Int()
		if index == 0 {
			return fmt.Errorf("no parameter named %q found", val.Name)
		}
	}

	switch typedVal := val.Value.(type) {
	case time.Time:
		val.Value = typedVal.UTC().Format(sqliteTimeFormat)
	case *time.Time:
		val.Value = typedVal.UTC().Format(sqliteTimeFormat)
	case int:
		val.Value = int64(typedVal)
	case int8:
		val.Value = int64(typedVal)
	case int16:
		val.Value = int64(typedVal)
	case int32:
		val.Value = int64(typedVal)
	case uint:
		val.Value = int64(typedVal)
	case uint8:
		val.Value = int64(typedVal)
	case uint16:
		val.Value = int64(typedVal)
	case uint32:
		val.Value = int64(typedVal)
	}

	// Fast path for supported unwrapped types
	switch val.Value.(type) {
	case int64, uint64, float32, float64, bool, string, []byte:
		return s.bindNonPointerValue(ctx, index, val.Value)
	case nil:
		return s.bindNil(ctx, index)
	}

	// Reflect path for wrapped types (pointers, custom type definitions of supported types)
	reflectVal := reflect.ValueOf(val.Value)
	for {
		switch reflectVal.Kind() {
		case reflect.Pointer:
			if reflectVal.IsNil() {
				return s.bindNil(ctx, index)
			}
			reflectVal = reflectVal.Elem()
			continue
		case reflect.Slice:
			if reflectVal.Elem().Kind() != reflect.Uint8 {
				return fmt.Errorf("unsupported slice type %T", reflectVal.Interface())
			}
			var typedVal []byte = unsafe.Slice((*byte)(reflectVal.UnsafePointer()), reflectVal.Len())
			return s.bindNonPointerValue(ctx, index, typedVal)
		case reflect.String:
			return s.bindNonPointerValue(ctx, index, reflectVal.String())
		case reflect.Bool:
			return s.bindNonPointerValue(ctx, index, reflectVal.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return s.bindNonPointerValue(ctx, index, reflectVal.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return s.bindNonPointerValue(ctx, index, reflectVal.Uint())
		case reflect.Float32, reflect.Float64:
			return s.bindNonPointerValue(ctx, index, reflectVal.Float())
		case reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan,
			reflect.Func, reflect.Interface, reflect.Map, reflect.Struct,
			reflect.UnsafePointer, reflect.Uintptr, reflect.Invalid:
			return fmt.Errorf("unsupported type %T", val.Value)
		}
		panic(fmt.Errorf("unreachable code reached with reflect kind %v", reflectVal.Kind()))
	}
}

func (s *Stmt) bind(ctx context.Context, args []driver.NamedValue) error {
	s.clearBindings(ctx)
	for _, arg := range args {
		err := s.BindValue(ctx, arg)
		if err != nil {
			return fmt.Errorf("failed to bind %d: %w", arg.Ordinal, err)
		}
	}
	return nil
}

func (s *Stmt) step(ctx context.Context) (bool, error) {
	rc := s.d.CAPI.Call("sqlite3_step", s.cptr).Int()
	if rc != SQLITE_OK && rc != SQLITE_ROW && rc != SQLITE_DONE {
		return false, s.d.MakeError(s.c, "sqlite3_step", rc)
	}
	return rc == SQLITE_ROW, nil
}

func (s *Stmt) reset(_ context.Context) error {
	rc := s.d.CAPI.Call("sqlite3_reset", s.cptr).Int()
	if rc != SQLITE_OK {
		return s.d.MakeError(s.c, "sqlite3_reset", rc)
	}
	return nil
}

func (s *Stmt) clearBindings(_ context.Context) {
	s.d.CAPI.Call("sqlite3_clear_bindings", s.cptr)
}

func (s *Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (res driver.Result, retErr error) {
	defer catchIntoError(&retErr)
	err := s.bind(ctx, args)
	if err != nil {
		return nil, err
	}
	_, err = s.step(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.reset(ctx); err != nil {
		return nil, err
	}
	return &Result{
		lastInsertID: s.c.rowsAffected(),
		rowsAffected: s.c.lastInsertRowID(),
	}, nil
}

func (s *Stmt) columns(_ context.Context) ([]string, []string) {
	count := s.d.CAPI.Call("sqlite3_column_count", s.cptr).Int()
	columns := make([]string, count)
	columnTypes := make([]string, count)
	for i := range columns {
		columns[i] = s.d.CAPI.Call("sqlite3_column_name", s.cptr, i).String()
		columnTypes[i] = strings.ToLower(s.d.CAPI.Call("sqlite3_column_decltype", s.cptr, i).String())
	}
	return columns, columnTypes
}

func (s *Stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	err := s.bind(ctx, args)
	if err != nil {
		return nil, err
	}
	cols, colTypes := s.columns(ctx)
	return &Rows{ctx: ctx, Stmt: s, columns: cols, columnTypes: colTypes}, nil
}

func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(noContextFunc, valuesToNamedValues(args))
}

func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(noContextFunc, valuesToNamedValues(args))
}
