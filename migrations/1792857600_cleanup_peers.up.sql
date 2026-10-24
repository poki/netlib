CREATE INDEX CONCURRENTLY IF NOT EXISTS lobbies_peers ON lobbies USING gin (peers);
