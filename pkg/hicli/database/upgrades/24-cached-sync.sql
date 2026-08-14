-- v24 (compatible with v10+): Add columns needed for caching room list on the frontend
-- transaction: sqlite-fkey-off
CREATE TABLE new_room (
	room_id              TEXT    NOT NULL PRIMARY KEY,
	room_type            TEXT,
	creation_content     TEXT,
	tombstone_content    TEXT,

	name                 TEXT,
	name_quality         INTEGER NOT NULL DEFAULT 0,
	avatar               TEXT,
	explicit_avatar      INTEGER NOT NULL DEFAULT 0,
	dm_user_id           TEXT,
	topic                TEXT,
	canonical_alias      TEXT,
	lazy_load_summary    TEXT,

	encryption_event     TEXT,
	has_member_list      INTEGER NOT NULL DEFAULT false,

	preview_event_rowid  INTEGER,
	sorting_timestamp    INTEGER,
	mod_timestamp        INTEGER NOT NULL DEFAULT (unixepoch('subsec')*1000),
	unread_highlights    INTEGER NOT NULL DEFAULT 0,
	unread_notifications INTEGER NOT NULL DEFAULT 0,
	unread_messages      INTEGER NOT NULL DEFAULT 0,
	marked_unread        INTEGER NOT NULL DEFAULT false,

	prev_batch           TEXT,

	CONSTRAINT room_preview_event_fkey FOREIGN KEY (preview_event_rowid) REFERENCES event (rowid) ON DELETE SET NULL
) STRICT;

CREATE TABLE new_account_data (
	user_id       TEXT    NOT NULL,
	type          TEXT    NOT NULL,
	content       TEXT    NOT NULL,
	mod_timestamp INTEGER NOT NULL DEFAULT (unixepoch('subsec') * 1000),

	PRIMARY KEY (user_id, type)
) STRICT;

CREATE TABLE new_room_account_data (
	user_id       TEXT    NOT NULL,
	room_id       TEXT    NOT NULL,
	type          TEXT    NOT NULL,
	content       TEXT    NOT NULL,
	mod_timestamp INTEGER NOT NULL DEFAULT (unixepoch('subsec') * 1000),

	PRIMARY KEY (user_id, room_id, type),
	CONSTRAINT room_account_data_room_fkey FOREIGN KEY (room_id) REFERENCES room (room_id) ON DELETE CASCADE
) STRICT;

INSERT INTO new_room (
	room_id, room_type, creation_content, tombstone_content, name, name_quality, avatar, explicit_avatar, dm_user_id,
	topic, canonical_alias, lazy_load_summary, encryption_event, has_member_list, preview_event_rowid, sorting_timestamp,
	unread_highlights, unread_notifications, unread_messages, marked_unread, prev_batch
)
SELECT
	room_id, room_type, creation_content, tombstone_content, name, name_quality, avatar, explicit_avatar, dm_user_id,
	topic, canonical_alias, lazy_load_summary, encryption_event, has_member_list, preview_event_rowid, sorting_timestamp,
	unread_highlights, unread_notifications, unread_messages, marked_unread, prev_batch
FROM room;

INSERT INTO new_account_data (user_id, type, content)
SELECT user_id, type, content FROM account_data;

INSERT INTO new_room_account_data (user_id, room_id, type, content)
SELECT user_id, room_id, type, content FROM room_account_data;

DROP TABLE room;
DROP TABLE account_data;
DROP TABLE room_account_data;

ALTER TABLE new_room RENAME TO room;
ALTER TABLE new_account_data RENAME TO account_data;
ALTER TABLE new_room_account_data RENAME TO room_account_data;

CREATE INDEX room_type_idx ON room (room_type);
CREATE INDEX room_sorting_timestamp_idx ON room (sorting_timestamp DESC);
CREATE INDEX room_mod_timestamp_idx ON room (mod_timestamp);
CREATE INDEX room_preview_idx ON room (preview_event_rowid);
CREATE INDEX account_data_mod_timestamp_idx ON account_data (mod_timestamp);
CREATE INDEX room_account_data_room_id_idx ON room_account_data (room_id);
CREATE INDEX room_account_data_mod_timestamp_idx ON account_data (mod_timestamp);
CREATE TRIGGER invited_room_delete_on_room_insert
	AFTER INSERT
	ON room
BEGIN
	DELETE FROM invited_room WHERE room_id = NEW.room_id;
END;
