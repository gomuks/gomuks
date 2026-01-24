// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

/*
#include "gomuks-ffi.h"
#include <stdlib.h>

static inline void _gomuks_callEventCallback(EventCallback cb, const char *command, int64_t request_id, GomuksBorrowedBuffer data) {
	cb(command, request_id, data);
}
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/cgo"
	"unsafe"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/version"
)

var commandNames = map[jsoncmd.Name]*C.char{}

func init() {
	for _, name := range jsoncmd.AllNames {
		commandNames[name] = C.CString(string(name))
	}
}

func bytesToBorrowedBuffer(b []byte) C.GomuksBorrowedBuffer {
	return C.GomuksBorrowedBuffer{
		base:   (*C.uint8_t)(unsafe.SliceData(b)),
		length: C.size_t(len(b)),
	}
}

func bytesToOwnedBuffer(b []byte) C.GomuksOwnedBuffer {
	return C.GomuksOwnedBuffer{
		base:   (*C.uint8_t)(C.CBytes(b)),
		length: C.size_t(len(b)),
	}
}

func borrowBufferBytes(buf C.GomuksBorrowedBuffer) []byte {
	return unsafe.Slice((*byte)(buf.base), buf.length)
}

type gomuksHandle struct {
	*gomuks.Gomuks
	ctx    context.Context
	cancel context.CancelFunc
}

func sendBufferedEvent[T any](callback C.EventCallback, command *jsoncmd.Container[T]) {
	data, _ := json.Marshal(command.Data)
	C._gomuks_callEventCallback(callback, commandNames[command.Command], C.int64_t(command.RequestID), bytesToBorrowedBuffer(data))
	runtime.KeepAlive(data)
}

//export GomuksInit
func GomuksInit() C.GomuksHandle {
	gmx := gomuks.NewGomuks()
	gmx.DisableAuth = true
	hicli.InitialDeviceDisplayName = "gomuks ffi" // TODO customizable name

	// TODO customizable storage directories
	gmx.InitDirectories()
	err := gmx.LoadConfig()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(9)
	}
	gmx.SetupLog()
	gmx.Log.Info().
		Str("version", version.Gomuks.FormattedVersion).
		Str("go_version", runtime.Version()).
		Time("built_at", version.Gomuks.BuildTime).
		Msg("Initializing gomuks FFI")

	cmdCtx, cancelCmdCtx := context.WithCancel(context.Background())
	cmdCtx = gmx.Log.WithContext(cmdCtx)
	return C.GomuksHandle(cgo.NewHandle(&gomuksHandle{
		Gomuks: gmx,
		ctx:    cmdCtx,
		cancel: cancelCmdCtx,
	}))
}

//export GomuksStart
func GomuksStart(handle C.GomuksHandle, callback C.EventCallback) {
	gmx := cgo.Handle(handle).Value().(*gomuksHandle)
	gmx.StartClient()
	gmx.Log.Info().Msg("Initialization complete")

	gmx.EventBuffer.Subscribe(0, nil, func(event *gomuks.BufferedEvent) {
		sendBufferedEvent(callback, event)
	})
	gmx.Log.Info().Msg("Sending initial state to client")
	sendBufferedEvent(callback, jsoncmd.SpecClientState.Format(gmx.Client.State()))
	sendBufferedEvent(callback, jsoncmd.SpecSyncStatus.Format(gmx.Client.SyncStatus.Load()))
	if gmx.Client.IsLoggedIn() {
		go func() {
			var roomCount int
			for payload := range gmx.Client.GetInitialSync(gmx.ctx, 100) {
				roomCount += len(payload.Rooms)
				sendBufferedEvent(callback, jsoncmd.SpecSyncComplete.Format(payload))
			}
			if gmx.ctx.Err() != nil {
				return
			}
			sendBufferedEvent(callback, jsoncmd.SpecInitComplete.Format(jsoncmd.Empty{}))
			gmx.Log.Info().Int("room_count", roomCount).Msg("Sent initial rooms to client")
		}()
	}
}

//export GomuksDestroy
func GomuksDestroy(handle C.GomuksHandle) {
	h := cgo.Handle(handle)
	gmx := h.Value().(*gomuksHandle)
	gmx.Log.Info().Msg("Shutting down gomuks FFI...")
	gmx.cancel()
	gmx.DirectStop()
	gmx.Log.Info().Msg("Shutdown complete")
	h.Delete()
}

//export GomuksSubmitCommand
func GomuksSubmitCommand(handle C.GomuksHandle, command *C.char, data C.GomuksBorrowedBuffer) C.GomuksResponse {
	gmx := cgo.Handle(handle).Value().(*gomuksHandle)
	res := gmx.Client.SubmitJSONCommand(gmx.ctx, &hicli.JSONCommand{
		Command: jsoncmd.Name(C.GoString(command)),
		Data:    borrowBufferBytes(data),
	})
	return C.GomuksResponse{
		buf:     bytesToOwnedBuffer(res.Data),
		command: commandNames[res.Command],
	}
}

//export GomuksFreeBuffer
func GomuksFreeBuffer(buf C.GomuksOwnedBuffer) {
	C.free(unsafe.Pointer(buf.base))
}

func main() {
	// Required for some reason, not actually used
}
