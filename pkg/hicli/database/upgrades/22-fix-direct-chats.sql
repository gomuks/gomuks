-- v22 (compatible with v10+): Fix DM status of some rooms that are missing the flag
WITH missing_dms AS (
    SELECT e.key AS user_id, r.value AS room_id, room.dm_user_id
    FROM account_data,
         json_each(content) AS e,
         json_each(e.value) AS r
    INNER JOIN room ON room.room_id=r.value
    WHERE account_data.type='m.direct'
      AND room.dm_user_id IS NULL
    GROUP BY r.value HAVING COUNT(*)=1
)
UPDATE room
SET dm_user_id=missing_dms.user_id
FROM missing_dms
WHERE room.room_id=missing_dms.room_id;
