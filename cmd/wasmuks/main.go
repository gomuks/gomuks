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

	postMessage := func(cmd jsoncmd.Name, reqID int64, data any) {
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
	gmx.EventBuffer = gomuks.NewEventBuffer(0)
	gmx.EventBuffer.Subscribe(0, nil, func(evt *gomuks.BufferedEvent) {
		postMessage(evt.Command, evt.RequestID, evt.Data)
	})
	js.Global().Call("addEventListener", "message", js.FuncOf(func(_ js.Value, args []js.Value) any {
		data := args[0].Get("data")
		wrappedCmd := &hicli.JSONCommand{
			Command:   jsoncmd.Name(data.Get("command").String()),
			RequestID: int64(data.Get("request_id").Int()),
			Data:      exstrings.UnsafeBytes(data.Get("data").String()),
		}
		go func() {
			resp := gmx.Client.SubmitJSONCommand(context.Background(), wrappedCmd)
			postMessage(resp.Command, resp.RequestID, resp.Data)
		}()
		return nil
	}))
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
