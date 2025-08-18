// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build js

package sqlite_wasm_js

import (
	"fmt"
	"syscall/js"
)

type Error struct {
	Function     string
	Code         int
	ExtendedCode int
	Message      string
}

func (e *Error) Error() string {
	if e.Code != e.ExtendedCode && e.ExtendedCode != 0 {
		return fmt.Sprintf("%s: error %d/%d: %s", e.Function, e.Code, e.ExtendedCode, e.Message)
	}
	return fmt.Sprintf("%s: error %d: %s", e.Function, e.Code, e.Message)
}

func (d *Driver) MakeError(c *Conn, funcName string, rc int) error {
	errMsg := "Unknown error"
	if errMsgVal := d.CAPI.Call("sqlite3_errmsg", c.cptr); errMsgVal.Type() == js.TypeString {
		errMsg = errMsgVal.String()
	} else if defMsgVal := d.CAPI.Call("sqlite3_js_rc_str", rc); defMsgVal.Type() == js.TypeString {
		errMsg = defMsgVal.String()
	}
	var extendedCode int
	if extendedCodeVal := d.CAPI.Call("sqlite3_extended_errcode", c.cptr); extendedCodeVal.Type() == js.TypeNumber {
		extendedCode = extendedCodeVal.Int()
	}
	return &Error{Function: funcName, Code: rc, ExtendedCode: extendedCode, Message: errMsg}
}
