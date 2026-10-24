-- v32 (compatible with v10+): Add bridge info to rooms
ALTER TABLE room ADD COLUMN bridge_info TEXT;

UPDATE room
SET bridge_info = (
	SELECT CASE
		WHEN e.content -> '$.network' IS NOT NULL
			THEN json_object('protocol', e.content -> '$.protocol', 'network', e.content -> '$.network')
		ELSE json_object('protocol', e.content -> '$.protocol')
	END
	FROM current_state cs
	JOIN event e ON cs.event_rowid = e.rowid
	WHERE cs.room_id = room.room_id
		AND cs.event_type IN ('m.bridge', 'uk.half-shot.bridge')
		AND e.content -> '$.protocol' IS NOT NULL
	ORDER BY e.timestamp DESC, e.rowid DESC
	LIMIT 1
)
WHERE EXISTS (
	SELECT 1
	FROM current_state cs
	JOIN event e ON cs.event_rowid = e.rowid
	WHERE cs.room_id = room.room_id
		AND cs.event_type IN ('m.bridge', 'uk.half-shot.bridge')
		AND e.content -> '$.protocol' IS NOT NULL
);
