BEGIN;

CREATE TABLE "peer_activity" (
  "peer" VARCHAR(20) NOT NULL PRIMARY KEY,
  "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX "peer_activity_updated_at" ON "peer_activity" ("updated_at");

COMMIT;
