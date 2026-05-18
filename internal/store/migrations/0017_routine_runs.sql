CREATE TABLE IF NOT EXISTS routine_runs (
    id          TEXT PRIMARY KEY,
    routine_id  TEXT NOT NULL REFERENCES routines(id),
    agent_id    TEXT NOT NULL REFERENCES agents(id),
    status      TEXT NOT NULL DEFAULT 'running',
    started_at  TEXT NOT NULL,
    finished_at TEXT,
    error       TEXT
);
CREATE INDEX IF NOT EXISTS idx_routine_runs_routine_id ON routine_runs(routine_id);
CREATE INDEX IF NOT EXISTS idx_routine_runs_agent_id   ON routine_runs(agent_id);
