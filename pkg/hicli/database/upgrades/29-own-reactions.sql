-- v29 (compatible with v10+): Aggregate own reactions
ALTER TABLE event ADD COLUMN own_reactions TEXT;

WITH reaction_ids AS (
	SELECT
		relates_to,
		content->>'$."m.relates_to".key' AS reaction,
		json_group_array(event_id) AS event_ids
	FROM event NOT INDEXED
	WHERE type = 'm.reaction'
	  AND relation_type = 'm.annotation'
	  AND redacted_by IS NULL
	  AND sender IN (SELECT user_id FROM account)
	  AND event_id NOT LIKE '~%'
	GROUP BY 1, 2
), own_reactions AS (
	SELECT
		relates_to,
		json_group_object(reaction, json(event_ids)) AS reaction_map
	FROM reaction_ids
	GROUP BY relates_to
)
UPDATE event AS target
SET own_reactions = own.reaction_map
FROM own_reactions AS own
WHERE target.event_id = own.relates_to;

DROP TRIGGER event_insert_fill_reactions;
DROP TRIGGER event_redact_fill_reactions;

CREATE TRIGGER event_insert_fill_reactions
	AFTER INSERT
	ON event
	WHEN NEW.type = 'm.reaction'
		AND NEW.relation_type = 'm.annotation'
		AND NEW.redacted_by IS NULL
		AND typeof(NEW.content ->> '$."m.relates_to".key') = 'text'
		AND NEW.event_id NOT LIKE '~%'
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

	UPDATE event
	SET own_reactions=json_insert(
		COALESCE(own_reactions, '{}'),
		'$.' || json_quote(NEW.content ->> '$."m.relates_to".key') || '[#]',
		NEW.event_id)
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND NEW.sender IN (SELECT user_id FROM account);
END;

CREATE TRIGGER event_send_complete_fill_reactions
	AFTER UPDATE
	ON event
	WHEN NEW.type = 'm.reaction'
		AND NEW.relation_type = 'm.annotation'
		AND OLD.event_id LIKE '~%'
		AND NEW.event_id LIKE '$%'
		AND typeof(NEW.content ->> '$."m.relates_to".key') = 'text'
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

	UPDATE event
	SET own_reactions=json_insert(
		COALESCE(own_reactions, '{}'),
		'$.' || json_quote(NEW.content ->> '$."m.relates_to".key') || '[#]',
		NEW.event_id)
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND NEW.redacted_by IS NULL
	  AND NEW.sender IN (SELECT user_id FROM account);
END;

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
		(
			SELECT json_group_array(value)
			FROM json_each(own_reactions, '$.' || json_quote(NEW.content ->> '$."m.relates_to".key'))
			WHERE value <> NEW.event_id
		))
	WHERE event_id = NEW.relates_to
	  AND room_id = NEW.room_id
	  AND NEW.sender IN (SELECT user_id FROM account)
	  AND own_reactions IS NOT NULL;
END;
