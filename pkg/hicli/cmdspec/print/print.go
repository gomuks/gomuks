// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"go.mau.fi/util/exbytes"
	"go.mau.fi/util/exerrors"
	"maunium.net/go/mautrix/event"

	"go.mau.fi/gomuks/pkg/hicli/cmdspec"
)

func main() {
	output := exerrors.Must(json.Marshal(&event.BotCommandsEventContent{
		Sigil:    "/",
		Commands: cmdspec.BuiltInCommands,
	}))
	if len(os.Args) > 1 && os.Args[1] != "-" {
		exerrors.PanicIfNotNil(os.WriteFile(os.Args[1], output, 0644))
	} else {
		fmt.Println(exbytes.UnsafeString(output))
	}
}
