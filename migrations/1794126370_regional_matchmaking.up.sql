BEGIN;

ALTER TABLE lobbies ADD COLUMN matchmaking_region text;

CREATE INDEX lobbies_matchmaking_region ON lobbies (game, matchmaking_region)
    WHERE matchmaking_region IS NOT NULL AND public AND password IS NULL
      AND cardinality(peers) > 0
      AND (max_players = 0 OR array_length(peers, 1) < max_players);

COMMIT;
