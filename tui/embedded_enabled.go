// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
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

//go:build !noembedded

package tui

import (
	"fmt"
	"runtime"

	"go.mau.fi/gomuks/pkg/gomuks"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/version"
)

const HasEmbeddedBackend = true

func (ui *GomuksTUI) InitEmbedded() (*client.GomuksClient, error) {
	backend := gomuks.NewGomuks()
	backend.InitDirectories("")
	err := backend.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	backend.SetupLog()
	backend.Log.Info().
		Str("version", version.Gomuks.FormattedVersion).
		Str("go_version", runtime.Version()).
		Time("built_at", version.Gomuks.BuildTime).
		Msg("Initializing gomuks embedded in terminal")
	backend.StartServer()
	c, err := client.NewGomuksClient(backend.Server.Addr)
	if err != nil {
		return nil, err
	}
	c.SetEmbeddedBackend(backend)
	return c, nil
}
