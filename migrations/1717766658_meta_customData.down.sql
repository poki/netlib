BEGIN;

ALTER TABLE "lobbies" RENAME COLUMN "custom_data" TO "meta";

COMMIT;
