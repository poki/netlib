ALTER TABLE peers DROP COLUMN connection_id;
DROP INDEX IF EXISTS lobbies_empty_updated_at;
DROP INDEX IF EXISTS lobbies_peers;
