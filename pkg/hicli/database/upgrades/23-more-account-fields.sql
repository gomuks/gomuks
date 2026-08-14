-- v23 (compatible with v10+): Add more fields for accounts
ALTER TABLE account ADD COLUMN client_id TEXT NOT NULL DEFAULT '';
ALTER TABLE account ADD COLUMN refresh_token TEXT NOT NULL DEFAULT '';
ALTER TABLE account ADD COLUMN expiry INTEGER NOT NULL DEFAULT 0;
ALTER TABLE account ADD COLUMN displayname TEXT NOT NULL DEFAULT '';
ALTER TABLE account ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';
