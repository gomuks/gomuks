-- v31 (compatible with v10+): Remove zero values in last_edit_rowid
UPDATE event SET last_edit_rowid=NULL WHERE last_edit_rowid=0;
DROP TRIGGER event_update_last_edit_when_redacted;
CREATE TRIGGER event_update_last_edit_when_redacted
	AFTER UPDATE
	ON event
	WHEN OLD.redacted_by IS NULL
		AND NEW.redacted_by IS NOT NULL
		AND NEW.relation_type = 'm.replace'
		AND NEW.state_key IS NULL
BEGIN
	UPDATE event
	SET last_edit_rowid = (
		SELECT rowid
		FROM event edit
		WHERE edit.room_id = event.room_id
		  AND edit.relates_to = event.event_id
		  AND edit.relation_type = 'm.replace'
		  AND edit.type = event.type
		  AND edit.sender = event.sender
		  AND edit.redacted_by IS NULL
		  AND edit.state_key IS NULL
		ORDER BY edit.timestamp DESC
		LIMIT 1
	)
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND last_edit_rowid = NEW.rowid
	  AND state_key IS NULL
	  AND (relation_type IS NULL OR relation_type NOT IN ('m.replace', 'm.annotation'));
END;
