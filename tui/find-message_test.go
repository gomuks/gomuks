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
	"testing"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/messages"
)

func makeTimelineEvent(eventID string, evtType string, txn string, service bool) *database.Event {
	evt := &database.Event{
		ID:            id.EventID(eventID),
		Type:          evtType,
		TransactionID: txn,
	}
	evt.RenderMeta = &messages.UIMessage{
		Event:     evt,
		IsService: service,
	}
	return evt
}

func TestFindMessageInTimeline_emptyAndNil(t *testing.T) {
	if got := findMessageInTimeline(nil, nil, false, selectAllowed); got != nil {
		t.Fatalf("nil timeline: got %v", got)
	}
	if got := findMessageInTimeline([]*database.Event{}, nil, false, selectAllowed); got != nil {
		t.Fatalf("empty timeline: got %v", got)
	}
}

func TestFindMessageInTimeline_skipsLocalEchoAndService(t *testing.T) {
	echo := makeTimelineEvent("$echo", event.EventMessage.Type, "$echo", false)
	notice := makeTimelineEvent("$svc", event.EventMessage.Type, "", true)
	ok := makeTimelineEvent("$ok", event.EventMessage.Type, "", false)
	member := makeTimelineEvent("$member", event.StateMember.Type, "", false)

	got := findMessageInTimeline([]*database.Event{echo, notice, member, ok}, nil, false, selectAllowed)
	if got == nil || got.ID != "$ok" {
		t.Fatalf("expected $ok from end, got %#v", got)
	}
}

func TestFindMessageInTimeline_previousFromNilSelectsLastAllowed(t *testing.T) {
	a := makeTimelineEvent("$a", event.EventMessage.Type, "", false)
	b := makeTimelineEvent("$b", event.EventMessage.Type, "", false)
	c := makeTimelineEvent("$c", event.EventMessage.Type, "", false)

	got := findMessageInTimeline([]*database.Event{a, b, c}, nil, false, selectAllowed)
	if got == nil || got.ID != "$c" {
		t.Fatalf("expected last message $c, got %#v", got)
	}
}

func TestFindMessageInTimeline_previousFromCurrent(t *testing.T) {
	a := makeTimelineEvent("$a", event.EventMessage.Type, "", false)
	b := makeTimelineEvent("$b", event.EventMessage.Type, "", false)
	c := makeTimelineEvent("$c", event.EventMessage.Type, "", false)
	timeline := []*database.Event{a, b, c}

	got := findMessageInTimeline(timeline, c, false, selectAllowed)
	if got == nil || got.ID != "$b" {
		t.Fatalf("expected $b before $c, got %#v", got)
	}
	got = findMessageInTimeline(timeline, a, false, selectAllowed)
	if got != nil {
		t.Fatalf("expected nil before first, got %#v", got)
	}
}

func TestFindMessageInTimeline_nextFromCurrent(t *testing.T) {
	a := makeTimelineEvent("$a", event.EventMessage.Type, "", false)
	b := makeTimelineEvent("$b", event.EventMessage.Type, "", false)
	c := makeTimelineEvent("$c", event.EventMessage.Type, "", false)
	timeline := []*database.Event{a, b, c}

	got := findMessageInTimeline(timeline, a, true, selectAllowed)
	if got == nil || got.ID != "$b" {
		t.Fatalf("expected $b after $a, got %#v", got)
	}
	got = findMessageInTimeline(timeline, c, true, selectAllowed)
	if got != nil {
		t.Fatalf("expected nil after last, got %#v", got)
	}
}

func TestSelectAllowed_rejectsNonMessage(t *testing.T) {
	evt := makeTimelineEvent("$m", event.StateMember.Type, "", false)
	if selectAllowed(evt) {
		t.Fatal("state events must not be reply targets")
	}
	if selectAllowed(nil) {
		t.Fatal("nil must not be allowed")
	}
}
