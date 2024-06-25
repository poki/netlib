BEGIN;

ALTER TABLE "lobbies"
    DROP COLUMN "leader",
    DROP COLUMN "term";

COMMIT;
