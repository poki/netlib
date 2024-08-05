BEGIN;

ALTER TABLE "lobbies"
    DROP COLUMN "max_players";

COMMIT;
