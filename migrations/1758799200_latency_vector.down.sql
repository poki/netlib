BEGIN;

ALTER TABLE "peers"
    DROP COLUMN IF EXISTS "latency_vector";

DROP INDEX IF EXISTS "peers_peer_with_latency_idx";

DROP FUNCTION IF EXISTS lobby_latency_estimate(vector(11), vector(11)[], int, float8);

COMMIT;
