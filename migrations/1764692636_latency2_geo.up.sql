BEGIN;

-- Required for 'earth' data type and distance calculations used in lobby listing query.
CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

ALTER TABLE "peers"
    ADD COLUMN IF NOT EXISTS "geo" earth;

COMMIT;
