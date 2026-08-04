-- v26 (compatible with v10+): Ignore redactions that are soft failed or not yet sent
DROP TRIGGER event_update_redacted_by;

CREATE TRIGGER event_update_redacted_by
	AFTER INSERT
	ON event
	WHEN NEW.type = 'm.room.redaction'
		AND NOT COALESCE(NEW.unsigned->>'io.element.synapse.soft_failed', false)
		AND NEW.event_id LIKE '$%'
BEGIN
	UPDATE event SET redacted_by = NEW.event_id WHERE room_id = NEW.room_id AND event_id = NEW.content ->> 'redacts';
END;

CREATE TRIGGER event_update_redacted_by_on_send_success
	AFTER UPDATE
	ON event
	WHEN OLD.type = 'm.room.redaction' AND NEW.type = 'm.room.redaction'
		AND (COALESCE(OLD.unsigned->>'io.element.synapse.soft_failed', false)
			OR OLD.event_id NOT LIKE '$%')
		AND NOT COALESCE(NEW.unsigned->>'io.element.synapse.soft_failed', false)
		AND NEW.event_id LIKE '$%'
BEGIN
	UPDATE event SET redacted_by = NEW.event_id WHERE room_id = NEW.room_id AND event_id = NEW.content ->> 'redacts';
END;
