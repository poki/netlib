BEGIN;

CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

ALTER TABLE "peers"
    ADD COLUMN IF NOT EXISTS "geo" earth;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'vector') THEN
        EXECUTE 'CREATE EXTENSION IF NOT EXISTS vector';
        EXECUTE 'ALTER TABLE "peers" ADD COLUMN IF NOT EXISTS "latency_vector" vector(11)';
        EXECUTE 'CREATE INDEX "peers_peer_with_latency_idx" ON "peers" ("peer") INCLUDE ("latency_vector") WHERE "latency_vector" IS NOT NULL';
        EXECUTE $fn$
CREATE OR REPLACE FUNCTION lobby_latency_estimate(
    peer_vector  vector(11),
    peers_vector vector(11)[],
    k            int     DEFAULT 3,
    w            float8  DEFAULT 0.7,
    eps          float8  DEFAULT 5
) RETURNS float8
LANGUAGE sql
IMMUTABLE
AS $body$
WITH peer AS (
    SELECT (peer_vector)::real[] AS pv
),
peers AS (
    SELECT p::real[] AS psv
    FROM unnest(peers_vector) AS t(p)
),
per_peer AS (
  SELECT
        -- lb = max_i |peer[i] - peers[i]| (Chebyshev distance)
        (
            SELECT MAX(ABS(a - b))
            FROM peer
            JOIN unnest(pv)  WITH ORDINALITY AS pve(a, i) ON TRUE
            JOIN unnest(psv) WITH ORDINALITY AS psve(b, i) USING (i)
        ) AS lb,
        -- ub = average of K smallest (peer[i] + peers[i])
        (
            SELECT AVG(s)
            FROM (
                SELECT (a + b) AS s
                FROM peer
                JOIN unnest(pv)  WITH ORDINALITY AS yu(a,i) ON TRUE
                JOIN unnest(psv) WITH ORDINALITY AS bu(b,i) USING (i)
                ORDER BY s ASC
                LIMIT k
            ) kbest
        ) AS ub
    FROM peers
),
per_peer_estimate AS (
    SELECT
        GREATEST(
            lb,
            LEAST((w * LEAST(1.0, lb/eps)) * ub + (1.0 - (w * LEAST(1.0, lb/eps))) * lb, ub)
        ) AS estimate
    FROM per_peer
)
SELECT ROUND(AVG(estimate))
FROM per_peer_estimate;
$body$;
$fn$;
    END IF;
END
$$;

COMMIT;
