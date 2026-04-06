-- v18 (compatible with v10+): Add sticky duration to events
ALTER TABLE event ADD COLUMN sticky_duration INTEGER;
CREATE INDEX event_sticky_idx ON event (room_id, timestamp) WHERE sticky_duration IS NOT NULL;
