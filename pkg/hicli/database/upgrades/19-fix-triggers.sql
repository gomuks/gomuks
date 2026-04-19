-- v19 (compatible with v10+): Fix various event triggers
DROP TRIGGER event_update_last_edit_when_redacted;
DROP TRIGGER event_insert_update_last_edit;
DROP TRIGGER event_insert_fill_reactions;
DROP TRIGGER event_redact_fill_reactions;

CREATE TRIGGER event_update_last_edit_when_redacted
	AFTER UPDATE
	ON event
	WHEN OLD.redacted_by IS NULL
		AND NEW.redacted_by IS NOT NULL
		AND NEW.relation_type = 'm.replace'
		AND NEW.state_key IS NULL
BEGIN
	UPDATE event
	SET last_edit_rowid = COALESCE(
		(SELECT rowid
		 FROM event edit
		 WHERE edit.room_id = event.room_id
		   AND edit.relates_to = event.event_id
		   AND edit.relation_type = 'm.replace'
		   AND edit.type = event.type
		   AND edit.sender = event.sender
		   AND edit.redacted_by IS NULL
		   AND edit.state_key IS NULL
		 ORDER BY edit.timestamp DESC
		 LIMIT 1),
		0)
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND last_edit_rowid = NEW.rowid
	  AND state_key IS NULL
	  AND (relation_type IS NULL OR relation_type NOT IN ('m.replace', 'm.annotation'));
END;

CREATE TRIGGER event_insert_update_last_edit
	AFTER INSERT
	ON event
	WHEN NEW.relation_type = 'm.replace'
		AND NEW.redacted_by IS NULL
		AND NEW.state_key IS NULL
BEGIN
	UPDATE event
	SET last_edit_rowid = NEW.rowid
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND type = NEW.type
	  AND sender = NEW.sender
	  AND state_key IS NULL
	  AND (relation_type IS NULL OR relation_type NOT IN ('m.replace', 'm.annotation'))
	  AND NEW.timestamp >
	      COALESCE((SELECT prev_edit.timestamp FROM event prev_edit WHERE prev_edit.rowid = event.last_edit_rowid), 0);
END;

CREATE TRIGGER event_insert_fill_reactions
	AFTER INSERT
	ON event
	WHEN NEW.type = 'm.reaction'
		AND NEW.relation_type = 'm.annotation'
		AND NEW.redacted_by IS NULL
		AND typeof(NEW.content ->> '$."m.relates_to".key') = 'text'
		AND NEW.content ->> '$."m.relates_to".key' NOT LIKE '%"%'
BEGIN
	UPDATE event
	SET reactions=json_set(
		reactions,
		'$.' || json_quote(NEW.content ->> '$."m.relates_to".key'),
		coalesce(
			reactions ->> ('$.' || json_quote(NEW.content ->> '$."m.relates_to".key')),
			0
		) + 1)
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND reactions IS NOT NULL;
END;

CREATE TRIGGER event_redact_fill_reactions
	AFTER UPDATE
	ON event
	WHEN NEW.type = 'm.reaction'
		AND NEW.relation_type = 'm.annotation'
		AND NEW.redacted_by IS NOT NULL
		AND OLD.redacted_by IS NULL
		AND typeof(NEW.content ->> '$."m.relates_to".key') = 'text'
		AND NEW.content ->> '$."m.relates_to".key' NOT LIKE '%"%'
BEGIN
	UPDATE event
	SET reactions=json_set(
		reactions,
		'$.' || json_quote(NEW.content ->> '$."m.relates_to".key'),
		coalesce(
			reactions ->> ('$.' || json_quote(NEW.content ->> '$."m.relates_to".key')),
			0
		) - 1)
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND reactions IS NOT NULL;
END;
