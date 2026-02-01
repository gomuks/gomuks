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

static inline void _gomuks_callProgressCallback(ProgressCallback cb, double progress) {
	cb(progress);
}
*/
import "C"
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"runtime/cgo"
	"unsafe"

	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/exbytes"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/ptr"
	"go.mau.fi/zeroconfig"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"

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
	var res *jsoncmd.Container[json.RawMessage]
	cmd := jsoncmd.Name(C.GoString(command))
	reqData := borrowBufferBytes(data)
	switch cmd {
	case jsoncmd.ReqGetAccountInfo, jsoncmd.ReqUploadMedia, jsoncmd.ReqExportKeys:
		res = gmx.handleFFICommand(cmd, reqData)
	default:
		res = gmx.Client.SubmitJSONCommand(gmx.ctx, &hicli.JSONCommand{
			Command: jsoncmd.Name(C.GoString(command)),
			Data:    reqData,
		})
	}
	return C.GomuksResponse{
		buf:     bytesToOwnedBuffer(res.Data),
		command: commandNames[res.Command],
	}
}

//export GomuksUploadMediaBytes
func GomuksUploadMediaBytes(handle C.GomuksHandle, params C.GomuksBorrowedBuffer, mediaBytes C.GomuksBorrowedBuffer, cb C.ProgressCallback) C.GomuksResponse {
	return gomuksUploadMediaAny(handle, params, borrowBufferBytes(mediaBytes), cb)
}

//export GomuksUploadMediaPath
func GomuksUploadMediaPath(handle C.GomuksHandle, params C.GomuksBorrowedBuffer, cb C.ProgressCallback) C.GomuksResponse {
	return gomuksUploadMediaAny(handle, params, nil, cb)
}

func gomuksUploadMediaAny(handle C.GomuksHandle, params C.GomuksBorrowedBuffer, direct []byte, cb C.ProgressCallback) C.GomuksResponse {
	gmx := cgo.Handle(handle).Value().(*gomuksHandle)
	if gmx.Client == nil {
		panic(fmt.Errorf("GomuksSubmitCommand called before GomuksStart"))
	}
	reqData := borrowBufferBytes(params)
	return containerToResponse(wrapFFIResponse(jsoncmd.SpecUploadMedia.Run(reqData, func(params *jsoncmd.UploadMediaParams) (*event.MessageEventContent, error) {
		var reader io.Reader
		if direct != nil {
			reader = bytes.NewReader(direct)
		}
		if params.Path != "" && params.Filename == "" {
			params.Filename = filepath.Base(params.Path)
		}
		return gmx.CacheAndUploadMedia(gmx.ctx, reader, *params, func(progress float64) {
			C._gomuks_callProgressCallback(cb, C.double(progress))
		})
	})))
}

func (gmx *gomuksHandle) handleFFICommand(cmd jsoncmd.Name, reqData []byte) *jsoncmd.Container[json.RawMessage] {
	switch cmd {
	case jsoncmd.ReqGetAccountInfo:
		return wrapFFIResponse(gmx.Client.Account, nil)
	case jsoncmd.ReqUploadMedia:
		return wrapFFIResponse(jsoncmd.SpecUploadMedia.Run(reqData, func(params *jsoncmd.UploadMediaParams) (*event.MessageEventContent, error) {
			return gmx.CacheAndUploadMedia(gmx.ctx, nil, *params, nil)
		}))
	case jsoncmd.ReqExportKeys:
		return wrapFFIResponse(jsoncmd.SpecExportKeys.Run(reqData, func(params *jsoncmd.ExportKeysParams) (string, error) {
			var sessions dbutil.RowIter[*crypto.InboundGroupSession]
			if params.RoomID == "" {
				sessions = gmx.Client.CryptoStore.GetAllGroupSessions(gmx.ctx)
			} else {
				sessions = gmx.Client.CryptoStore.GetGroupSessionsForRoom(gmx.ctx, params.RoomID)
			}
			export, err := crypto.ExportKeysIter(params.Passphrase, sessions)
			if errors.Is(err, crypto.ErrNoSessionsForExport) {
				return "", nil
			} else if err != nil {
				return "", err
			}
			return exbytes.UnsafeString(export), nil
		}))
	default:
		panic(fmt.Errorf("invalid call to handleFFICommand(%s)", cmd))
	}
}

func wrapFFIResponse(res any, err error) *jsoncmd.Container[json.RawMessage] {
	if err != nil {
		return &jsoncmd.Container[json.RawMessage]{
			Command: jsoncmd.RespError,
			Data:    exerrors.Must(json.Marshal(err.Error())),
		}
	}
	return &jsoncmd.Container[json.RawMessage]{
		Command: jsoncmd.RespSuccess,
		Data:    exerrors.Must(json.Marshal(res)),
	}
}

func containerToResponse(res *jsoncmd.Container[json.RawMessage]) C.GomuksResponse {
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
