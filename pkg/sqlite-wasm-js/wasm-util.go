// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build js

package sqlite_wasm_js

import (
	"fmt"
	"runtime/debug"
)

type jsError struct {
	val any
}

func (e jsError) Unwrap() error {
	err, ok := e.val.(error)
	if ok {
		return err
	}
	return nil
}

func (e jsError) Error() string {
	if err := e.Unwrap(); err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%v", e.val)
}

func catchIntoError(into *error) {
	if r := recover(); r != nil {
		fmt.Println("MEOW 3:<", r)
		fmt.Println(string(debug.Stack()))
		*into = jsError{val: r}
	}
}

func catchIntoErrorFmt(into *error, format string, args ...any) {
	if r := recover(); r != nil {
		args = append(args, jsError{val: r})
		*into = fmt.Errorf(format, args...)
	}
}
