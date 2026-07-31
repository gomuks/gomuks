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

package tui

import (
	"context"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/tui/debug"
)

type LoginView struct {
	*mauview.Form

	container *mauview.Centerer

	serverLabel   *mauview.TextField
	usernameLabel *mauview.TextField
	passwordLabel *mauview.TextField

	server   *mauview.InputField
	username *mauview.InputField
	password *mauview.InputField
	error    *mauview.TextView

	loginButton *mauview.Button
	quitButton  *mauview.Button

	loading bool

	parent *GomuksTUI
}

func (ui *GomuksTUI) NewLoginView() mauview.Component {
	view := &LoginView{
		Form: mauview.NewForm(),

		serverLabel:   mauview.NewTextField().SetText("Backend"),
		usernameLabel: mauview.NewTextField().SetText("Username"),
		passwordLabel: mauview.NewTextField().SetText("Password"),

		server:   mauview.NewInputField(),
		username: mauview.NewInputField(),
		password: mauview.NewInputField(),

		loginButton: mauview.NewButton("Login"),
		quitButton:  mauview.NewButton("Quit"),

		parent: ui,
	}

	view.server.SetPlaceholder("http://localhost:29325").SetText(view.parent.Config.Server).SetTextColor(tcell.ColorWhite)
	view.username.SetPlaceholder("username").SetText(view.parent.Config.Username).SetTextColor(tcell.ColorWhite)
	view.password.SetPlaceholder("correct horse battery staple").SetMaskCharacter('*').SetTextColor(tcell.ColorWhite)

	view.quitButton.
		SetOnClick(func() { ui.Finish() }).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetForegroundColor(tcell.ColorWhite).
		SetFocusedForegroundColor(tcell.ColorWhite)
	view.loginButton.
		SetOnClick(view.Login).
		SetBackgroundColor(tcell.ColorDarkCyan).
		SetForegroundColor(tcell.ColorWhite).
		SetFocusedForegroundColor(tcell.ColorWhite)

	view.
		SetColumns([]int{1, 10, 1, 30, 1}).
		SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	view.
		AddFormItem(view.server, 3, 1, 1, 1).
		AddFormItem(view.username, 3, 3, 1, 1).
		AddFormItem(view.password, 3, 5, 1, 1).
		AddFormItem(view.loginButton, 1, 7, 3, 1).
		AddFormItem(view.quitButton, 1, 9, 3, 1).
		AddComponent(view.serverLabel, 1, 1, 1, 1).
		AddComponent(view.usernameLabel, 1, 3, 1, 1).
		AddComponent(view.passwordLabel, 1, 5, 1, 1)
	view.FocusNextItem()
	ui.LoginView = view

	view.container = mauview.Center(mauview.NewBox(view).SetTitle("Log in to gomuks"), 45, 13)
	view.container.SetAlwaysFocusChild(true)
	return view.container
}

func (view *LoginView) Error(err string) {
	if len(err) == 0 && view.error != nil {
		debug.Print("Hiding error")
		view.RemoveComponent(view.error)
		view.container.SetHeight(13)
		view.SetRows([]int{1, 1, 1, 1, 1, 1, 1, 1, 1})
		view.error = nil
	} else if len(err) > 0 {
		debug.Print("Showing error", err)
		if view.error == nil {
			view.error = mauview.NewTextView().SetTextColor(tcell.ColorRed)
			view.AddComponent(view.error, 1, 11, 3, 1)
		}
		view.error.SetText(err + "\n\nMake sure you enter your gomuks backend\naddress, not a Matrix homeserver.")
		errorHeight := int(math.Ceil(float64(runewidth.StringWidth(err))/41)) + 3
		view.container.SetHeight(14 + errorHeight)
		view.SetRow(11, errorHeight)
	}

	view.parent.Render()
}

func (view *LoginView) actuallyLogin(server, username, password string) {
	debug.Printf("Logging into %s as %s...", server, username)
	view.parent.Config.Server = server
	var err error
	view.parent.gmx, err = client.NewGomuksClient(server)
	if err != nil {
		view.Error(err.Error())
		debug.Print("Init error:", err)
	} else if err = view.parent.gmx.Authenticate(context.TODO(), username, password); err != nil {
		view.Error(err.Error())
		debug.Print("Auth error:", err)
	} else {
		view.parent.Config.Username = username
		view.parent.Config.Password = password
		view.parent.Config.Save()
		view.parent.Connect()
		view.parent.SetView(ViewMain)
	}
}

func (view *LoginView) Login() {
	if view.loading {
		return
	}
	serverAddr := view.server.GetText()
	mxid := view.username.GetText()
	password := view.password.GetText()

	view.loading = true
	view.loginButton.SetText("Logging in...")
	go view.actuallyLogin(serverAddr, mxid, password)
}
