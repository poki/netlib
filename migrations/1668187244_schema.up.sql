BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE "lobbies" (
  "code" VARCHAR(20) NOT NULL PRIMARY KEY,
  "game" uuid NOT NULL,
  "peers" VARCHAR(20)[] NULL,
  "public" BOOLEAN NOT NULL DEFAULT FALSE,
  "meta" jsonb NULL,
  "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX "lobbies_game" ON "lobbies" ("game", "public");

COMMIT;
