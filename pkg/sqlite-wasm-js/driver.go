// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build js

package sqlite_wasm_js

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"
)

type Driver struct {
	SQLite js.Value

	OO1  js.Value
	CAPI js.Value
	WASM js.Value
	Meow js.Value
}

var (
	_ driver.Driver = &Driver{}
	//_ driver.DriverContext = &Driver{}
)

func parseOptionalBool(val string, defVal bool) bool {
	if val == "" {
		return defVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defVal
	}
	return b
}

func (d *Driver) Open(connectionString string) (conn driver.Conn, retErr error) {
	defer catchIntoError(&retErr)
	var connectionURI *url.URL
	var err error
	if strings.HasPrefix(connectionString, "file:") || strings.Contains(connectionString, "?") {
		connectionURI, err = url.Parse(connectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse connection string: %w", err)
		}
	} else {
		connectionURI = &url.URL{
			Scheme: "file",
			Path:   connectionString,
		}
	}
	//fmt.Printf("%#v\n", connectionURI)
	query := connectionURI.Query()
	readOnly := parseOptionalBool(query.Get("read_only"), false)
	create := parseOptionalBool(query.Get("create"), true)
	enableTracing := parseOptionalBool(query.Get("enable_tracing"), false)
	connectionMode := query.Get("connection_mode")
	if connectionMode == "" {
		connectionMode = "opfs-sahpool"
	}
	txLock := strings.ToUpper(query.Get("_txlock"))
	switch txLock {
	case "", "IMMEDIATE", "DEFERRED", "EXCLUSIVE":
		// ok
	default:
		return nil, fmt.Errorf("invalid transaction lock mode %q", txLock)
	}
	var constructorFlags string
	if readOnly {
		constructorFlags = "r"
	} else {
		constructorFlags = "w"
		if create {
			constructorFlags += "c"
		}
	}
	if enableTracing {
		constructorFlags += "t"
	}
	var db js.Value
	var sahPool bool
	switch strings.ToLower(connectionMode) {
	case "memory":
		db = d.OO1.Get("DB").New(":memory:", constructorFlags)
	case "opfs":
		db = d.OO1.Get("OpfsDb").New(connectionURI.Path, constructorFlags)
	case "opfs-sahpool":
		db = d.SQLite.Get("PoolUtil").Get("OpfsSAHPoolDb").New(connectionURI.Path)
		sahPool = true
	default:
		return nil, fmt.Errorf("invalid connection mode %q", connectionMode)
	}
	conn, retErr = (&Conn{
		d:       d,
		ptr:     db,
		cptr:    db.Get("pointer"),
		txlock:  txLock,
		sahpool: sahPool,
	}).connectHook(noContextFunc)
	return
}

//func (s *Driver) OpenConnector(connectionString string) (driver.Connector, error) {
//	return nil, fmt.Errorf("not implemented")
//}

func init() {
	val := js.Global().Get("sqlite3")
	if !val.IsUndefined() {
		sql.Register("sqlite-wasm-js", &Driver{
			SQLite: val,

			OO1:  val.Get("oo1"),
			CAPI: val.Get("capi"),
			WASM: val.Get("wasm"),
			Meow: val.Get("meow"),
		})
	}
}
