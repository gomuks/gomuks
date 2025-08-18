// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build js

package hicli

import (
	"errors"

	sqlite_wasm_js "go.mau.fi/gomuks/pkg/sqlite-wasm-js"
)

func init() {
	isDatabaseBusyError = func(err error) bool {
		var sqliteErr *sqlite_wasm_js.Error
		return errors.As(err, &sqliteErr) && sqliteErr.Code == 5
	}
}
