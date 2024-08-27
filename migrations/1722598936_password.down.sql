BEGIN;

ALTER TABLE "lobbies"
    DROP COLUMN "password";

COMMIT;
