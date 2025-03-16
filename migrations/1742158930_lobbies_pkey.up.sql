BEGIN;

ALTER TABLE lobbies
    DROP CONSTRAINT lobbies_pkey;

ALTER TABLE lobbies
    ADD CONSTRAINT lobbies_pkey
    PRIMARY KEY (code, game);

COMMIT;
