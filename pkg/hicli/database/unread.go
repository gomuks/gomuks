// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const (
	// TODO find out if this needs to be wrapped in another query that limits the number of events it evaluates
	//      (or maybe the timeline store just shouldn't be allowed to grow that big?)
	// An event is read up to the latest receipt in its thread (the main timeline for events outside threads)
	// or in no thread. A thread without such a receipt is unread, unless the room has no receipt at all.
	// Keep in sync with upgrades/32-thread-unreads.sql.
	calculateUnreadsQuery = `
		WITH read_up_to AS (
			SELECT r.thread_id, MAX(t.rowid) AS rowid
			FROM receipt r
			JOIN event e ON e.event_id = r.event_id
			JOIN timeline t ON t.event_rowid = e.rowid
			WHERE r.room_id = $1
				AND r.user_id = $2
				AND t.room_id = $1
			GROUP BY r.thread_id
		)
		SELECT
			thread_id,
			SUM(CASE WHEN unread_type & 0100 THEN 1 ELSE 0 END) AS highlights,
			SUM(CASE WHEN unread_type & 0010 THEN 1 ELSE 0 END) AS notifications,
			SUM(CASE WHEN unread_type & 0001 THEN 1 ELSE 0 END) AS messages
		FROM (
			SELECT
				timeline.rowid,
				unread_type,
				CASE WHEN event.relation_type = 'm.thread' THEN event.relates_to ELSE 'main' END AS thread_id
			FROM timeline
			JOIN event ON event.rowid = timeline.event_rowid
			WHERE timeline.room_id = $1 AND unread_type > 0 AND redacted_by IS NULL
		) unread
		WHERE unread.rowid > COALESCE(
			(SELECT MAX(rowid) FROM read_up_to WHERE read_up_to.thread_id IN ('', unread.thread_id)),
			(SELECT 0 FROM read_up_to LIMIT 1)
		)
		GROUP BY thread_id
	`
)

// CalculateUnreads returns the unread counts of a room, in total and per thread ("main" for the main timeline).
func (rq *RoomQuery) CalculateUnreads(ctx context.Context, roomID id.RoomID, userID id.UserID) (total UnreadCounts, threads map[event.ThreadID]UnreadCounts, err error) {
	rows, err := rq.GetDB().Query(ctx, calculateUnreadsQuery, roomID, userID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var threadID event.ThreadID
		var uc UnreadCounts
		if err = rows.Scan(&threadID, &uc.UnreadHighlights, &uc.UnreadNotifications, &uc.UnreadMessages); err != nil {
			return
		}
		if threads == nil {
			threads = make(map[event.ThreadID]UnreadCounts)
		}
		threads[threadID] = uc
		total.Add(uc)
	}
	err = rows.Err()
	return
}

type UnreadType int

func (ut UnreadType) Is(flag UnreadType) bool {
	return ut&flag != 0
}

const (
	UnreadTypeNone      UnreadType = 0b0000
	UnreadTypeNormal    UnreadType = 0b0001
	UnreadTypeNotify    UnreadType = 0b0010
	UnreadTypeHighlight UnreadType = 0b0100
	UnreadTypeSound     UnreadType = 0b1000
)

type UnreadCounts struct {
	UnreadHighlights    int `json:"unread_highlights"`
	UnreadNotifications int `json:"unread_notifications"`
	UnreadMessages      int `json:"unread_messages"`
}

func (uc *UnreadCounts) IsZero() bool {
	return uc.UnreadHighlights == 0 && uc.UnreadNotifications == 0 && uc.UnreadMessages == 0
}

func (uc *UnreadCounts) Add(other UnreadCounts) {
	uc.UnreadHighlights += other.UnreadHighlights
	uc.UnreadNotifications += other.UnreadNotifications
	uc.UnreadMessages += other.UnreadMessages
}

func (uc *UnreadCounts) AddOne(ut UnreadType) {
	if ut.Is(UnreadTypeNormal) {
		uc.UnreadMessages++
	}
	if ut.Is(UnreadTypeNotify) {
		uc.UnreadNotifications++
	}
	if ut.Is(UnreadTypeHighlight) {
		uc.UnreadHighlights++
	}
}
