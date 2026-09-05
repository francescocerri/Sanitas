-- Schema minimo per validare la pipeline DB -> API -> CI -> deploy.
-- Non è la progettazione definitiva del dominio turni.
--
-- Own schema, not the shared database's public one — see docs/adr/0014.
-- Embedded into the binary and run by this service's own startup (with
-- retry, see cmd/server/main.go), not docker-entrypoint-initdb.d — see
-- docs/adr/0019. The connection already sets search_path=turni, but this
-- SET is kept so the script stays correct even run over a plain connection.
CREATE SCHEMA IF NOT EXISTS turni;
SET search_path TO turni;

CREATE TABLE IF NOT EXISTS turni (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Real FK into anagrafica's schema now that it exists (ADR-0014) — the
    -- one field of this placeholder model that stopped being a text
    -- placeholder. data/ora_inizio/ora_fine stay TEXT, same reasoning as
    -- before (see docs/adr/0005): still not the final domain design.
    volontario_id UUID NOT NULL REFERENCES anagrafica.users(id),
    data TEXT NOT NULL,
    ora_inizio TEXT NOT NULL,
    ora_fine TEXT NOT NULL,
    stato TEXT NOT NULL DEFAULT 'pianificato',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
