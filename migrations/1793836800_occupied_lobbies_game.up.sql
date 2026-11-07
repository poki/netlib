BEGIN;

CREATE INDEX "lobbies_occupied_game" ON "lobbies" ("game")
    WHERE cardinality(peers) > 0;

COMMIT;
