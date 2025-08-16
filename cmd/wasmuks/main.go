// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:build js

package main

import (
	"context"
	"encoding/json"
	"runtime"
	"syscall/js"

	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/exbytes"
	"go.mau.fi/util/exstrings"
	"go.mau.fi/util/ptr"
	"go.mau.fi/zeroconfig"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/hicli"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	_ "go.mau.fi/gomuks/pkg/sqlite-wasm-js"
	"go.mau.fi/gomuks/version"
)

var gmx *gomuks.Gomuks

func postMessage(cmd jsoncmd.Name, reqID int64, data any) {
	var dataJSON json.RawMessage
	var ok bool
	if dataJSON, ok = data.(json.RawMessage); !ok {
		var err error
		dataJSON, err = json.Marshal(data)
		if err != nil {
			gmx.Log.Err(err).Msg("Failed to marshal data for postMessage")
			return
		}
	}
	js.Global().Call("postMessage", js.ValueOf(map[string]any{
		"command":    string(cmd),
		"request_id": int(reqID),
		"data":       exbytes.UnsafeString(dataJSON),
	}))
}

func jsMessageListener(_ js.Value, args []js.Value) any {
	data := args[0].Get("data")
	wrappedCmd := &hicli.JSONCommand{
		Command:   jsoncmd.Name(data.Get("command").String()),
		RequestID: int64(data.Get("request_id").Int()),
		Data:      exstrings.UnsafeBytes(data.Get("data").String()),
	}
	if wrappedCmd.Command == "wasm-upload" {
		fileName := data.Get("filename").String()
		encrypt := data.Get("encrypt").Bool()
		payloadVal := data.Get("payload")
		payload := make([]byte, payloadVal.Length())
		js.CopyBytesToGo(payload, payloadVal)
		go func() {
			ctx := gmx.Log.With().Str("action", "wasmuks upload").Logger().WithContext(context.Background())
			resp, err := uploadMedia(ctx, fileName, encrypt, payload)
			if err != nil {
				postMessage(jsoncmd.RespError, wrappedCmd.RequestID, ptr.Ptr(gomuks.ToRespError(err)))
			} else {
				postMessage(jsoncmd.RespSuccess, wrappedCmd.RequestID, resp)
			}
		}()
		return nil
	}
	go func() {
		resp := gmx.Client.SubmitJSONCommand(context.Background(), wrappedCmd)
		postMessage(resp.Command, resp.RequestID, resp.Data)
	}()
	return nil
}

func main() {
	hicli.InitialDeviceDisplayName = "gomuks web"
	gmx = gomuks.NewGomuks()
	gmx.Version = version.Version
	gmx.Commit = version.Commit
	gmx.LinkifiedVersion = version.LinkifiedVersion
	gmx.BuildTime = version.ParsedBuildTime
	gmx.Config = gomuks.Config{
		Logging: zeroconfig.Config{
			Writers: []zeroconfig.WriterConfig{{
				Type: zeroconfig.WriterTypeJS,
			}},
			Timestamp: ptr.Ptr(false),
		},
	}
	gmx.GetDBConfig = func() dbutil.PoolConfig {
		return dbutil.PoolConfig{
			Type:         "sqlite-wasm-js",
			URI:          "file:/gomuks.db?_txlock=immediate",
			MaxOpenConns: 5,
			MaxIdleConns: 1,
		}
	}

	gmx.EventBuffer = gomuks.NewEventBuffer(0)
	gmx.EventBuffer.Subscribe(0, nil, func(evt *gomuks.BufferedEvent) {
		postMessage(evt.Command, evt.RequestID, evt.Data)
	})
	gmx.DisablePush = true
	js.Global().Call("addEventListener", "message", js.FuncOf(jsMessageListener))
	js.Global().Set("meowDownloadMedia", js.FuncOf(jsDownloadCallback))
	postMessage("wasm-connection", 0, json.RawMessage(`{"connected":true,"reconnecting":false,"error":null}`))

	gmx.SetupLog()
	gmx.Log.Info().
		Str("version", gmx.Version).
		Str("go_version", runtime.Version()).
		Time("built_at", gmx.BuildTime).
		Msg("Initializing gomuks in wasm")
	gmx.StartClient()
	gmx.Log.Info().Msg("Initialization complete")
	postMessage(jsoncmd.EventClientState, 0, gmx.Client.State())
	postMessage(jsoncmd.EventSyncStatus, 0, gmx.Client.SyncStatus.Load())
	if gmx.Client.IsLoggedIn() {
		ctx := gmx.Log.WithContext(context.Background())
		for payload := range gmx.Client.GetInitialSync(ctx, 100) {
			postMessage(jsoncmd.EventSyncComplete, 0, payload)
		}
		postMessage(jsoncmd.EventInitComplete, 0, gmx.Client.SyncStatus.Load())
	}

	select {}
}
