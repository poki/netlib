BEGIN;

DROP INDEX IF EXISTS "peers_peer_with_latency_idx";
DROP FUNCTION IF EXISTS lobby_latency_estimate;

ALTER TABLE "peers"
    DROP COLUMN IF EXISTS "latency_vector",
    DROP COLUMN IF EXISTS "geo";

DROP EXTENSION IF EXISTS vector;
DROP EXTENSION IF EXISTS earthdistance;
DROP EXTENSION IF EXISTS cube;

COMMIT;
