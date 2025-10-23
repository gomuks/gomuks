// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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

package version

import (
	"fmt"

	"go.mau.fi/util/progver"
	"maunium.net/go/mautrix"
)

var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var Gomuks = progver.ProgramVersion{
	Name:        "gomuks",
	URL:         "https://github.com/gomuks/gomuks",
	BaseVersion: "25.10",
	SemCalVer:   true,
}.Init(Tag, Commit, BuildTime)

func init() {
	mautrix.DefaultUserAgent = fmt.Sprintf("gomuks/%s %s", Gomuks.FormattedVersion, mautrix.DefaultUserAgent)
}
