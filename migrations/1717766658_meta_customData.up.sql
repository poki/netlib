BEGIN;

ALTER TABLE "lobbies" RENAME COLUMN "meta" TO "custom_data";

COMMIT;
