BEGIN;

ALTER TABLE "peers"
    ADD COLUMN IF NOT EXISTS "latency_vector" vector(11);


-- Index to speed up lobby queries that now join on peers to get latency_vector.
CREATE INDEX "peers_peer_with_latency_idx"
    ON "peers" ("peer") INCLUDE ("latency_vector")
    WHERE "latency_vector" IS NOT NULL;


-- Estimates lobby latency for a new peer by averaging network triangulation with existing peers.
--
-- See: https://www.wisdom.weizmann.ac.il/~robi/papers/K-triangulation-SPAA07.pdf
--
-- for each peer, compute a Chebyshev distance lower bound (max |peer[i]-peers[i]|)
-- and a robust upper bound as the mean of the k smallest (peer[i] + peers[i]) and blend them by w (clamped),
-- (to prevent triangle inequality violations from producing estimates below the lower bound)
-- then return the average of those per-peer estimates.
--
-- k = number of smallest sums.
-- w (0..1) biases toward the upper bound (more w = more bias toward upper bound).
-- eps is used to model measurement noise (higher eps = more noise = more bias toward lower bound).
CREATE OR REPLACE FUNCTION lobby_latency_estimate(
    peer_vector  vector(11),
    peers_vector vector(11)[],
    k            int     DEFAULT 3,
    w            float8  DEFAULT 0.7,
    eps          float8  DEFAULT 5
) RETURNS float8
LANGUAGE sql
IMMUTABLE
AS $$
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
$$;

COMMIT;
