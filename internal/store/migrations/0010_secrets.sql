-- Secrets table for storing sensitive application configuration.
-- Includes company_id FK and unique constraint on (company_id, name).

CREATE TABLE IF NOT EXISTS secrets (
    id          TEXT PRIMARY KEY,
    company_id  TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    value       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    UNIQUE(company_id, name)
);

CREATE INDEX IF NOT EXISTS secrets_company_idx ON secrets(company_id);
