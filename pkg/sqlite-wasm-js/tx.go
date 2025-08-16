// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build js

package sqlite_wasm_js

import (
	"database/sql/driver"
)

type Tx struct {
	*Conn
}

var (
	_ driver.Tx = &Tx{}
)

func (t *Tx) Commit() error {
	rc := t.d.CAPI.Call("sqlite3_exec", t.cptr, "COMMIT", 0, 0, 0).Int()
	if rc != SQLITE_OK {
		return t.d.MakeError(t.Conn, "sqlite3_exec", rc)
	}
	return nil
}

func (t *Tx) Rollback() error {
	rc := t.d.CAPI.Call("sqlite3_exec", t.cptr, "ROLLBACK", 0, 0, 0).Int()
	if rc != SQLITE_OK {
		return t.d.MakeError(t.Conn, "sqlite3_exec", rc)
	}
	return nil
}
