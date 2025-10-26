// gomuks - A terminal Matrix client written in Go.
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

package ui

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gdamore/tcell/v2"
	"github.com/zyedidia/clipboard"
	"go.mau.fi/mauview"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/exzerolog"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/debug"
)

type View string

// Allowed views in GomuksTUI
const (
	ViewLogin View = "login"
	ViewMain  View = "main"
)

type GomuksTUI struct {
	gmx *client.GomuksClient
	app *mauview.Application

	Config *config.Config

	MainView  *MainView
	LoginView *LoginView

	views map[View]mauview.Component
}

func init() {
	mauview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	mauview.Styles.PrimaryTextColor = tcell.ColorDefault
	mauview.Styles.BorderColor = tcell.ColorDefault
	mauview.Styles.ContrastBackgroundColor = tcell.ColorDarkGreen
	if tcellDB := os.Getenv("TCELLDB"); len(tcellDB) == 0 {
		if info, err := os.Stat("/usr/share/tcell/database"); err == nil && info.IsDir() {
			_ = os.Setenv("TCELLDB", "/usr/share/tcell/database")
		}
	}
}

func NewGomuksTUI() *GomuksTUI {
	ui := &GomuksTUI{
		app:    mauview.NewApplication(),
		Config: config.NewConfig(),
	}
	debug.OnRecover = ui.app.ForceStop
	return ui
}

func (ui *GomuksTUI) Run() {
	ui.Config.LoadAll()
	log := exerrors.Must(ui.Config.LogConfig.Compile())
	exzerolog.SetupDefaults(log)
	loggedIn := false
	if ui.Config.Server != "" && ui.Config.Username != "" && ui.Config.Password != "" {
		ui.gmx = exerrors.Must(client.NewGomuksClient(ui.Config.Server))
		exerrors.PanicIfNotNil(ui.gmx.Authenticate(context.TODO(), ui.Config.Username, ui.Config.Password))
		loggedIn = true
	}

	mauview.Backspace2RemovesWord = ui.Config.Backspace2RemovesWord
	mauview.Backspace1RemovesWord = ui.Config.Backspace1RemovesWord
	ui.app.SetAlwaysClear(ui.Config.AlwaysClearScreen)
	_ = clipboard.Initialize()
	ui.views = map[View]mauview.Component{
		ViewLogin: ui.NewLoginView(),
		ViewMain:  ui.NewMainView(),
	}
	if loggedIn {
		ui.SetView(ViewMain)
	} else {
		ui.SetView(ViewLogin)
	}
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		go ui.Stop()
		<-c
		ui.Finish()
	}()

	if ui.gmx != nil {
		go ui.Connect()
	}
	exerrors.PanicIfNotNil(ui.app.Start())
}

func (ui *GomuksTUI) Connect() {
	ui.gmx.SendNotification = ui.MainView.NotifyMessage
	ui.gmx.EventHandler = func(ctx context.Context, rawEvt any) {
		switch rawEvt.(type) {
		case *jsoncmd.SyncComplete:
			// TODO make this smarter
			debug.Print("Rendering...")
			ui.Render()
		}
	}
	ui.MainView.matrix = ui.gmx
	exerrors.PanicIfNotNil(ui.gmx.Connect(context.TODO()))
}

func (ui *GomuksTUI) Stop() {
	debug.Print("Stopping")
	ui.gmx.Disconnect()
	debug.Print("Disconnection complete")
	ui.app.Stop()
	debug.Print("Stopped")
	os.Exit(0)
}

func (ui *GomuksTUI) Finish() {
	ui.app.ForceStop()
	os.Exit(0)
}

func (ui *GomuksTUI) Render() {
	ui.app.Redraw()
}

func (ui *GomuksTUI) OnLogin() {
	ui.SetView(ViewMain)
}

func (ui *GomuksTUI) OnLogout() {
	ui.SetView(ViewLogin)
}

func (ui *GomuksTUI) HandleNewPreferences() {
	ui.Render()
}

func (ui *GomuksTUI) SetView(name View) {
	ui.app.SetRoot(ui.views[name])
}

func (ui *GomuksTUI) RunExternal(executablePath string, args ...string) error {
	callback := make(chan error)
	ui.app.Suspend(func() {
		cmd := exec.Command(executablePath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = os.Environ()
		callback <- cmd.Run()
	})
	return <-callback
}
