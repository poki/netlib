BEGIN;

ALTER TABLE "peers"
    ADD COLUMN IF NOT EXISTS "lat" double precision,
    ADD COLUMN IF NOT EXISTS "lon" double precision;

-- Required for distance calculations based on latitude/longitude pairs.
CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

-- Estimate RTT (ms) between two points using a simple linear model on earth distances.
-- Warning: this function will not work on other planets.
CREATE OR REPLACE FUNCTION est_rtt_ms_earth(
    lat1          double precision,
    lon1          double precision,
    lat2          double precision,
    lon2          double precision,
    base_ms       double precision DEFAULT 5.0,  -- base latency in ms, e.g. routing, processing, etc.
    rtt_per_km_ms double precision DEFAULT 0.015 -- additional RTT in ms per km of distance.
) RETURNS double precision
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT
        base_ms + (earth_distance(ll_to_earth(lat1, lon1), ll_to_earth(lat2, lon2)) / 1000.0) * rtt_per_km_ms;
$$;

COMMIT;
