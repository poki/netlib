BEGIN;

DROP FUNCTION IF EXISTS est_rtt_ms_earth(double precision, double precision, double precision, double precision);

ALTER TABLE "peers"
    DROP COLUMN IF EXISTS "lat",
    DROP COLUMN IF EXISTS "lon";

COMMIT;
