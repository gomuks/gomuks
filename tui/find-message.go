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
	"maunium.net/go/mautrix/event"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/messages"
)

type findFilter func(evt *database.Event) bool

func isLocalEcho(evt *database.Event) bool {
	if evt == nil || evt.ID == "" {
		return true
	}
	return evt.TransactionID != "" && string(evt.ID) == evt.TransactionID
}

func asUIMessage(evt *database.Event) *messages.UIMessage {
	if evt == nil || evt.RenderMeta == nil {
		return nil
	}
	uiMsg, ok := evt.RenderMeta.(*messages.UIMessage)
	if !ok || uiMsg == nil || uiMsg.IsService {
		return nil
	}
	return uiMsg
}

func selectAllowed(evt *database.Event) bool {
	if evt == nil || isLocalEcho(evt) {
		return false
	}
	return evt.GetType() == event.EventMessage
}

func findMessageInTimeline(timeline []*database.Event, current *database.Event, forward bool, allow findFilter) *messages.UIMessage {
	if len(timeline) == 0 {
		return nil
	}

	currentFound := current == nil
	for i := 0; i < len(timeline); i++ {
		index := i
		if !forward {
			index = len(timeline) - i - 1
		}
		evt := timeline[index]
		if isLocalEcho(evt) {
			continue
		}
		uiMsg := asUIMessage(evt)
		if uiMsg == nil {
			continue
		}
		if currentFound {
			if allow == nil || allow(evt) {
				return uiMsg
			}
		} else if current != nil && evt.ID == current.ID {
			currentFound = true
		}
	}
	return nil
}
