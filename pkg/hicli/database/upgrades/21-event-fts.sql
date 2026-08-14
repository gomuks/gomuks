-- v21 (compatible with v10+): Add full-text search table for event content
CREATE VIRTUAL TABLE event_search USING fts5 (
	"from",
	body,
	tokenize='porter unicode61 remove_diacritics 2',
	content='',
	contentless_delete=1,
	detail=full
);

INSERT INTO event_search(rowid, "from", body)
SELECT rowid, sender, COALESCE(decrypted, content) ->> '$.body'
FROM event
WHERE COALESCE(decrypted_type, type) IN ('m.room.message', 'm.sticker')
  AND state_key IS NULL
  AND COALESCE(decrypted, content) ->> '$.body' <> '';

CREATE TRIGGER event_insert_add_search_index
	AFTER INSERT
	ON event
	WHEN COALESCE(NEW.decrypted_type, NEW.type) IN ('m.room.message', 'm.sticker')
		AND NEW.state_key IS NULL
		AND COALESCE(NEW.decrypted, NEW.content) ->> '$.body' <> ''
BEGIN
	INSERT INTO event_search(rowid, "from", body)
	VALUES (NEW.rowid, NEW.sender, COALESCE(NEW.decrypted, NEW.content) ->> '$.body');
END;

CREATE TRIGGER event_decrypted_add_search_index
	AFTER UPDATE
	ON event
	WHEN NEW.type = 'm.room.encrypted'
		AND OLD.decrypted IS NULL
		AND OLD.decrypted_type IS NULL
		AND NEW.decrypted_type IN ('m.room.message', 'm.sticker')
		AND NEW.state_key IS NULL
		AND NEW.decrypted ->> '$.body' <> ''
BEGIN
	INSERT INTO event_search(rowid, "from", body)
	VALUES (NEW.rowid, NEW.sender, NEW.decrypted ->> '$.body');
END;

CREATE TRIGGER event_delete_remove_search_index
	AFTER DELETE
	ON event
	WHEN COALESCE(OLD.decrypted_type, OLD.type) IN ('m.room.message', 'm.sticker')
		AND OLD.state_key IS NULL
BEGIN
	DELETE FROM event_search WHERE rowid=OLD.rowid;
END;
