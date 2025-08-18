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
	"sync/atomic"
	"syscall/js"

	"go.mau.fi/util/exerrors"
)

var noContextFunc = context.Background()

type Conn struct {
	d    *Driver
	ptr  js.Value
	cptr js.Value

	closed atomic.Bool

	txlock  string
	sahpool bool
}

var (
	_ driver.Conn               = &Conn{}
	_ driver.ConnPrepareContext = &Conn{}
	_ driver.ConnBeginTx        = &Conn{}
	_ driver.Execer             = &Conn{}
	_ driver.ExecerContext      = &Conn{}
	_ driver.Queryer            = &Conn{}
	_ driver.QueryerContext     = &Conn{}
	//_ driver.NamedValueChecker = &Conn{}
	_ driver.Validator = &Conn{}
	_ driver.Pinger    = &Conn{}
)

func (c *Conn) IsValid() bool {
	return !c.closed.Load()
}

func (c *Conn) Ping(ctx context.Context) error {
	return nil
}

func (c *Conn) Close() error {
	c.closed.Store(true)
	rc := c.d.CAPI.Call("sqlite3_close_v2", c.cptr).Int()
	if rc != SQLITE_OK {
		return c.d.MakeError(c, "sqlite3_close_v2", rc)
	}
	return nil
}

func (c *Conn) connectHook(ctx context.Context) (dc driver.Conn, err error) {
	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()
	_, err = c.ExecContext(ctx, "PRAGMA foreign_keys = ON", nil)
	if err != nil {
		return
	}
	if c.sahpool {
		// WAL mode is only useful when using the SAH pool, so don't enable it otherwise
		_, err = c.ExecContext(ctx, "PRAGMA journal_mode = WAL", nil)
		if err != nil {
			return
		}
		_, err = c.ExecContext(ctx, "PRAGMA synchronous = NORMAL", nil)
		if err != nil {
			return
		}
	}
	_, err = c.ExecContext(ctx, "PRAGMA busy_timeout = 10000", nil)
	if err != nil {
		return
	}
	return c, nil
}

//func (c *Conn) CheckNamedValue(value *driver.NamedValue) error {
//	return nil
//}

func (c *Conn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, retErr error) {
	defer catchIntoError(&retErr)
	res := c.d.Meow.Call("prepare", c.cptr, query)
	rc := res.Get("rc")
	ptr := res.Get("ptr")
	if !rc.IsUndefined() {
		return nil, c.d.MakeError(c, "sqlite3_prepare_v2", rc.Int())
	} else if ptr.IsUndefined() {
		return nil, fmt.Errorf("sqlite3_prepare_v2 returned no error and no statement")
	} else {
		return &Stmt{d: c.d, c: c, cptr: ptr}, nil
	}
}

func (c *Conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if len(args) == 0 {
		rc := c.d.CAPI.Call("sqlite3_exec", c.cptr, query, 0, 0, 0).Int()
		if rc != SQLITE_OK {
			return nil, c.d.MakeError(c, "sqlite3_exec", rc)
		}
		return &Result{
			lastInsertID: c.lastInsertRowID(),
			rowsAffected: c.rowsAffected(),
		}, nil
	}
	stmt, err := c.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	res, err := stmt.(*Stmt).ExecContext(ctx, args)
	if err != nil {
		return nil, err
	}
	return res, stmt.Close()
}

func (c *Conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	stmt, err := c.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	res, err := stmt.(*Stmt).QueryContext(ctx, args)
	if err != nil {
		_ = stmt.Close()
		return nil, err
	}
	res.(*Rows).closeStmt = true
	return res, nil
}

func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	rc := c.d.CAPI.Call("sqlite3_exec", c.cptr, "BEGIN "+c.txlock, 0, 0, 0).Int()
	if rc != SQLITE_OK {
		return nil, c.d.MakeError(c, "sqlite3_exec", rc)
	}
	return &Tx{c}, nil
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(noContextFunc, query)
}

func valuesToNamedValues(args []driver.Value) []driver.NamedValue {
	values := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		values[i] = driver.NamedValue{
			Ordinal: i + 1,
			Value:   arg,
		}
	}
	return values
}

func (c *Conn) lastInsertRowID() int64 {
	return exerrors.Must(parseStrOrNumber(c.d.Meow.Call("last_insert_rowid", c.cptr)))
}

func (c *Conn) rowsAffected() int64 {
	// TODO this could use sqlite3_changes64 instead to get a bigint
	return int64(c.d.CAPI.Call("sqlite3_changes", c.cptr).Int())
}

func (c *Conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	return c.ExecContext(noContextFunc, query, valuesToNamedValues(args))
}

func (c *Conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	return c.QueryContext(noContextFunc, query, valuesToNamedValues(args))
}

func (c *Conn) Begin() (driver.Tx, error) {
	return c.BeginTx(noContextFunc, driver.TxOptions{})
}
