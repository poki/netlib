ALTER TABLE "peers"
    ADD COLUMN IF NOT EXISTS "lat" float8,
    ADD COLUMN IF NOT EXISTS "lon" float8;
