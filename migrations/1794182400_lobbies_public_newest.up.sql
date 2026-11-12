CREATE INDEX IF NOT EXISTS lobbies_public_newest ON public.lobbies USING btree (game, created_at DESC, code)
    WHERE public;
