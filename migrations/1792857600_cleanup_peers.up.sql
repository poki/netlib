CREATE INDEX IF NOT EXISTS lobbies_peers ON lobbies USING gin (peers);
CREATE INDEX IF NOT EXISTS lobbies_empty_updated_at ON lobbies (updated_at) WHERE peers = '{}';
ALTER TABLE peers ADD COLUMN connection_id VARCHAR(20);
