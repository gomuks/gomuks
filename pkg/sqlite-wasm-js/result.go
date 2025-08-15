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

type Result struct {
	lastInsertID int64
	rowsAffected int64
}

var (
	_ driver.Result = &Result{}
)

func (r *Result) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r *Result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}
