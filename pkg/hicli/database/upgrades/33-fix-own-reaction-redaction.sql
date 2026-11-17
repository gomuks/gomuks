-- v33 (compatible with v10+): Fix redacting own reactions storing arrays as strings
DROP TRIGGER event_redact_fill_reactions;

CREATE TRIGGER event_redact_fill_reactions
	AFTER UPDATE
	ON event
	WHEN NEW.type = 'm.reaction'
		AND NEW.relation_type = 'm.annotation'
		AND NEW.redacted_by IS NOT NULL
		AND OLD.redacted_by IS NULL
		AND typeof(NEW.content ->> '$."m.relates_to".key') = 'text'
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

	UPDATE event
	SET own_reactions = json_set(
		own_reactions,
		'$.' || json_quote(NEW.content ->> '$."m.relates_to".key'),
		json((
			SELECT json_group_array(value)
			FROM json_each(own_reactions, '$.' || json_quote(NEW.content ->> '$."m.relates_to".key'))
			WHERE value <> NEW.event_id
		)))
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND NEW.sender IN (SELECT user_id FROM account)
	  AND own_reactions IS NOT NULL;
END;

UPDATE event
SET own_reactions = (
	SELECT json_group_object(key, json(
		CASE
			WHEN NOT json_valid(value) THEN '[]'
			WHEN json_type(value) = 'array' THEN value
			ELSE '[]'
		END
	))
	FROM json_each(own_reactions)
)
WHERE own_reactions IS NOT NULL
  AND EXISTS(SELECT 1 FROM json_each(own_reactions) WHERE type <> 'array');
