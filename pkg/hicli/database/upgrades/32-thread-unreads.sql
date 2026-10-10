-- v32 (compatible with v10+): Add unread counts per thread
ALTER TABLE room ADD COLUMN thread_unreads TEXT;
-- Same counting as calculateUnreadsQuery, for all rooms at once
CREATE TEMPORARY TABLE thread_unreads_upgrade AS
WITH read_up_to AS (
	SELECT r.room_id, r.thread_id, MAX(t.rowid) AS rowid
	FROM receipt r
	JOIN event e ON e.event_id = r.event_id
	JOIN timeline t ON t.event_rowid = e.rowid AND t.room_id = r.room_id
	WHERE r.user_id = (SELECT user_id FROM account)
	GROUP BY r.room_id, r.thread_id
)
SELECT
	room_id,
	thread_id,
	SUM(CASE WHEN unread_type & 0100 THEN 1 ELSE 0 END) AS highlights,
	SUM(CASE WHEN unread_type & 0010 THEN 1 ELSE 0 END) AS notifications,
	SUM(CASE WHEN unread_type & 0001 THEN 1 ELSE 0 END) AS messages
FROM (
	SELECT
		timeline.room_id,
		timeline.rowid,
		unread_type,
		CASE WHEN event.relation_type = 'm.thread' THEN event.relates_to ELSE 'main' END AS thread_id
	FROM timeline
	JOIN event ON event.rowid = timeline.event_rowid
	WHERE unread_type > 0 AND redacted_by IS NULL
) unread
WHERE unread.rowid > COALESCE(
	(
		SELECT MAX(rowid)
		FROM read_up_to
		WHERE read_up_to.room_id = unread.room_id AND read_up_to.thread_id IN ('', unread.thread_id)
	),
	(SELECT 0 FROM read_up_to WHERE read_up_to.room_id = unread.room_id LIMIT 1)
)
GROUP BY room_id, thread_id;
UPDATE room SET
	thread_unreads = (
		SELECT json_group_object(thread_id, json_object(
			'unread_highlights', highlights,
			'unread_notifications', notifications,
			'unread_messages', messages
		))
		FROM thread_unreads_upgrade tu WHERE tu.room_id = room.room_id
	),
	unread_highlights = (SELECT SUM(highlights) FROM thread_unreads_upgrade tu WHERE tu.room_id = room.room_id),
	unread_notifications = (SELECT SUM(notifications) FROM thread_unreads_upgrade tu WHERE tu.room_id = room.room_id),
	unread_messages = (SELECT SUM(messages) FROM thread_unreads_upgrade tu WHERE tu.room_id = room.room_id)
WHERE room_id IN (SELECT room_id FROM thread_unreads_upgrade);
UPDATE room SET unread_highlights = 0, unread_notifications = 0, unread_messages = 0
WHERE room_id NOT IN (SELECT room_id FROM thread_unreads_upgrade);
DROP TABLE thread_unreads_upgrade;
