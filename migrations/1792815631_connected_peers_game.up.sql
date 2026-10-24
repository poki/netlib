BEGIN;

CREATE INDEX "peers_connected_game" ON "peers" ("game") INCLUDE ("peer")
    WHERE "disconnected" = FALSE;

COMMIT;
