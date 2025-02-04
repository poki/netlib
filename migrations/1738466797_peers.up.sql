BEGIN;

DROP TABLE "timeouts";

CREATE TABLE "peers" (
  "peer" VARCHAR(20) NOT NULL PRIMARY KEY,
  "secret" VARCHAR(24) DEFAULT NULL,
  "game" uuid NOT NULL,
  "disconnected" BOOLEAN NOT NULL DEFAULT FALSE,
  "last_seen" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX "peers_last_seen" ON "peers" ("last_seen");

COMMIT;
