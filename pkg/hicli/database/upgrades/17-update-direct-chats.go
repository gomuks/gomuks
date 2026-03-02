// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package upgrades

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func init() {
	Table.Register(-1, 17, 10, "Use m.direct for direct chats", dbutil.TxnModeOn, func(ctx context.Context, db *dbutil.Database) error {
		_, err := db.Exec(ctx, "UPDATE room SET dm_user_id = NULL")
		if err != nil {
			return err
		}
		var directChats []byte
		err = db.QueryRow(ctx, "SELECT content FROM account_data WHERE user_id=(SELECT user_id FROM account) AND type='m.direct'").
			Scan(&directChats)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		} else if err != nil {
			return err
		}
		var parsedDirectChats event.DirectChatsEventContent
		err = json.Unmarshal(directChats, &parsedDirectChats)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to parse m.direct data, skipping database migration")
			return nil
		}
		dmRooms := make(map[id.RoomID]id.UserID, len(parsedDirectChats))
		for userID, rooms := range parsedDirectChats {
			for _, roomID := range rooms {
				val, exists := dmRooms[roomID]
				if exists {
					if val != "" && val != userID {
						dmRooms[roomID] = ""
					}
				} else {
					dmRooms[roomID] = userID
				}
			}
		}
		for roomID, userID := range dmRooms {
			if userID == "" {
				continue
			}
			_, err = db.Exec(ctx, "UPDATE room SET dm_user_id = $1 WHERE room_id = $2", userID, roomID)
			if err != nil {
				return fmt.Errorf("failed to update room %s with DM user %s: %w", roomID, userID, err)
			}
		}
		return nil
	})
}
