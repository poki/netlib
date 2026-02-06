BEGIN;

DROP INDEX IF EXISTS "latencies_lookup_idx";
DROP TABLE IF EXISTS "latencies";
DROP TABLE IF EXISTS "latency_meta";

ALTER TABLE "peers"
    DROP COLUMN IF EXISTS "country",
    DROP COLUMN IF EXISTS "region";

DROP FUNCTION IF EXISTS lobby_latency_estimate(VARCHAR(20)[], text, text);

COMMIT;
