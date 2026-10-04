-- v30 (compatible with v10+): Improve various indexes
DROP INDEX event_relates_to_idx;
CREATE INDEX event_relates_to_idx ON event (room_id, relates_to) WHERE relates_to IS NOT NULL;

DROP INDEX event_megolm_session_id_idx;
CREATE INDEX event_failed_decryption_idx ON event (room_id, megolm_session_id) WHERE decryption_error IS NOT NULL;

DROP INDEX event_redacted_by_idx;
CREATE INDEX event_room_preview_idx ON event (room_id, timestamp DESC)
	WHERE redacted_by IS NULL AND (relation_type IS NULL OR relation_type <> 'm.replace');

DROP INDEX room_account_data_mod_timestamp_idx;
CREATE INDEX room_account_data_mod_timestamp_idx ON room_account_data (mod_timestamp);

CREATE INDEX current_state_shared_rooms_idx ON current_state (state_key)
	WHERE event_type='m.room.member' AND membership='join';

CREATE INDEX receipt_event_idx ON receipt (room_id, event_id);

CREATE INDEX event_room_mention_idx ON event (room_id, timestamp DESC) WHERE unread_type > 0;
