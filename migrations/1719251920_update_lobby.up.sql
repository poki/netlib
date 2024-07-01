BEGIN;

CREATE TYPE canUpdateByEnum AS ENUM('creator', 'leader', 'anyone', 'none');

ALTER TABLE "lobbies"
    ADD COLUMN "can_update_by" canUpdateByEnum NOT NULL DEFAULT 'creator',
    ADD COLUMN "creator" VARCHAR(20) NOT NULL DEFAULT '';

COMMIT;
