-- Schema minimo per validare la pipeline DB -> API -> CI -> deploy.
-- Non è la progettazione definitiva del dominio turni.
CREATE TABLE IF NOT EXISTS turni (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    volontario_id TEXT NOT NULL,
    data TEXT NOT NULL,
    ora_inizio TEXT NOT NULL,
    ora_fine TEXT NOT NULL,
    stato TEXT NOT NULL DEFAULT 'pianificato',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
