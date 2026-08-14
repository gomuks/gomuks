-- v25 (compatible with v10+): Add more columns needed for caching room list on the frontend
-- transaction: sqlite-fkey-off
DROP TRIGGER invited_room_delete_on_room_insert;

CREATE TABLE new_invited_room (
	room_id       TEXT    NOT NULL PRIMARY KEY,
	received_at   INTEGER NOT NULL,
	mod_timestamp INTEGER NOT NULL DEFAULT (unixepoch('subsec') * 1000),
	invite_state  TEXT    NOT NULL
) STRICT;

INSERT INTO new_invited_room (room_id, received_at, invite_state)
SELECT room_id, received_at, invite_state FROM invited_room;

DROP TABLE invited_room;
ALTER TABLE new_invited_room RENAME TO invited_room;

CREATE TABLE left_room (
	room_id       TEXT    NOT NULL PRIMARY KEY,
	mod_timestamp INTEGER NOT NULL DEFAULT (unixepoch('subsec') * 1000)
) STRICT;

CREATE TRIGGER invited_room_delete_on_room_insert
	AFTER INSERT
	ON room
BEGIN
	DELETE FROM invited_room WHERE room_id = NEW.room_id;
	DELETE FROM left_room WHERE room_id = NEW.room_id;
END;
