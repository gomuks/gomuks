-- v16 (compatible with v10+): Fix empty dm_user_id in room table
UPDATE room SET dm_user_id = NULL WHERE dm_user_id = '';
