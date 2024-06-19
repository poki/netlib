BEGIN;

ALTER TABLE "lobbies"
    DROP COLUMN "can_update_by",
    DROP COLUMN "creator";

DROP TYPE canUpdateByEnum;

COMMIT;
