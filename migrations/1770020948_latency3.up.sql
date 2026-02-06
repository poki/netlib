BEGIN;

ALTER TABLE "peers"
    ADD COLUMN IF NOT EXISTS "country" VARCHAR(2),
    ADD COLUMN IF NOT EXISTS "region" VARCHAR(16);

CREATE TABLE IF NOT EXISTS "latency_meta" (
    "id" INT PRIMARY KEY,
    "version" INT NOT NULL
);

CREATE TABLE IF NOT EXISTS "latencies" (
    "from_country" VARCHAR(2) NOT NULL,
    "from_region" VARCHAR(16),
    "to_country" VARCHAR(2) NOT NULL,
    "to_region" VARCHAR(16),
    "latency_ms_p50" DOUBLE PRECISION NOT NULL
);

CREATE INDEX IF NOT EXISTS "latencies_lookup_idx"
    ON "latencies" ("from_country", "to_country", "from_region", "to_region");

CREATE OR REPLACE FUNCTION lobby_latency_estimate(
    peer_ids VARCHAR(20)[],
    from_country text,
    from_region text
) RETURNS float8
LANGUAGE sql
STABLE
AS $$
    SELECT CASE
        WHEN $2 IS NULL OR $2 = '' OR $2 = 'XX' THEN 250.0
        ELSE (
            SELECT ROUND(AVG(COALESCE(latency_lookup.latency_ms_p50, 250.0)))
            FROM peers p
            LEFT JOIN LATERAL (
                SELECT l.latency_ms_p50
                FROM latencies l
                WHERE l.from_country = $2
                  AND l.to_country = NULLIF(NULLIF(p.country, ''), 'XX')
                  AND (l.from_region = NULLIF($3, '') OR l.from_region IS NULL)
                  AND (l.to_region = NULLIF(p.region, '') OR l.to_region IS NULL)
                ORDER BY
                  (l.from_region IS NOT NULL)::int + (l.to_region IS NOT NULL)::int DESC
                LIMIT 1
            ) AS latency_lookup ON TRUE
            WHERE p.peer = ANY ($1)
        )
    END;
$$;

COMMIT;
