CREATE TABLE IF NOT EXISTS routines (
    id                   TEXT PRIMARY KEY,
    company_id           TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    agent_id             TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name                 TEXT NOT NULL,
    cron_expr            TEXT NOT NULL,
    enabled              INTEGER NOT NULL DEFAULT 1,
    last_run_at          TEXT,
    dispatch_fingerprint TEXT,
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL,
    UNIQUE(company_id, name)
);

CREATE INDEX IF NOT EXISTS routines_company_idx ON routines(company_id);
CREATE INDEX IF NOT EXISTS routines_agent_idx   ON routines(agent_id);
CREATE INDEX IF NOT EXISTS routines_enabled_idx ON routines(enabled);
