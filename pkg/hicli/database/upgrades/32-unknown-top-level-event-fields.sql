-- v32 (compatible with v10+): Allow storing unknown top-level event fields
ALTER TABLE event ADD COLUMN unknown_top_level TEXT;
