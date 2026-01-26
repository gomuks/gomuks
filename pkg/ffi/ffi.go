// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

/*
#include "gomuksffi.h"
#include <stdlib.h>

static inline void _gomuks_callEventCallback(EventCallback cb, const char *command, int64_t request_id, GomuksOwnedBuffer data) {
	cb(command, request_id, data);
}
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/cgo"
	"unsafe"

	"github.com/rs/zerolog"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/ptr"
	"go.mau.fi/zeroconfig"

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
	data := exerrors.Must(json.Marshal(command.Data))
	C._gomuks_callEventCallback(callback, commandNames[command.Command], C.int64_t(command.RequestID), bytesToOwnedBuffer(data))
}

//export GomuksInit
func GomuksInit() C.GomuksHandle {
	gomuks.DisablePush = true
	hicli.InitialDeviceDisplayName = "gomuks ffi" // TODO customizable name
	gmx := gomuks.NewGomuks()
	gmx.DisableAuth = true
	cmdCtx, cancelCmdCtx := context.WithCancel(context.Background())
	return C.GomuksHandle(cgo.NewHandle(&gomuksHandle{
		Gomuks: gmx,
		ctx:    cmdCtx,
		cancel: cancelCmdCtx,
	}))
}

//export GomuksStart
func GomuksStart(handle C.GomuksHandle, callback C.EventCallback) C.int {
	gmx := cgo.Handle(handle).Value().(*gomuksHandle)

	// TODO customizable storage directories and config
	gmx.InitDirectories()
	gmx.Config = gomuks.Config{
		Logging: zeroconfig.Config{
			MinLevel: ptr.Ptr(zerolog.DebugLevel),
			Writers: []zeroconfig.WriterConfig{{
				Type:   zeroconfig.WriterTypeStdout,
				Format: zeroconfig.LogFormatPrettyColored,
			}, {
				Type:   zeroconfig.WriterTypeFile,
				Format: "json",
				FileConfig: zeroconfig.FileConfig{
					Filename:   filepath.Join(gmx.LogDir, "gomuks.log"),
					MaxSize:    100,
					MaxBackups: 10,
				},
			}},
		},
	}
	gmx.EventBuffer = gomuks.NewEventBuffer(0)
	gmx.SetupLog()
	gmx.ctx = gmx.Log.WithContext(gmx.ctx)
	gmx.Log.Info().
		Str("version", version.Gomuks.FormattedVersion).
		Str("go_version", runtime.Version()).
		Time("built_at", version.Gomuks.BuildTime).
		Msg("Starting gomuks FFI")

	eventChan := make(chan *gomuks.BufferedEvent, 1024)
	gmx.EventBuffer.Subscribe(0, nil, func(event *gomuks.BufferedEvent) {
		eventChan <- event
	})

	exitCode := gmx.StartClientWithoutExit(gmx.ctx)
	if exitCode != 0 {
		return C.int(exitCode)
	}
	gmx.Log.Info().Msg("Initialization complete")

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
			go gmx.runEventChan(eventChan, callback)
		}()
	} else {
		go gmx.runEventChan(eventChan, callback)
	}
	return 0
}

func (gmx *gomuksHandle) runEventChan(ch chan *gomuks.BufferedEvent, callback C.EventCallback) {
	doneChan := gmx.ctx.Done()
	for {
		select {
		case evt := <-ch:
			sendBufferedEvent(callback, evt)
		case <-doneChan:
			return
		}
	}
}

//export GomuksDestroy
func GomuksDestroy(handle C.GomuksHandle) {
	h := cgo.Handle(handle)
	gmx := h.Value().(*gomuksHandle)
	h.Delete()
	log := gmx.Log
	if log == nil {
		log = ptr.Ptr(zerolog.Nop())
	}
	log.Info().Msg("Shutting down gomuks FFI...")
	gmx.cancel()
	gmx.DirectStop()
	log.Info().Msg("Shutdown complete")
}

//export GomuksSubmitCommand
func GomuksSubmitCommand(handle C.GomuksHandle, command *C.char, data C.GomuksBorrowedBuffer) C.GomuksResponse {
	gmx := cgo.Handle(handle).Value().(*gomuksHandle)
	if gmx.Client == nil {
		panic(fmt.Errorf("GomuksSubmitCommand called before GomuksStart"))
	}
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
