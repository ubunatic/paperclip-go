CREATE TABLE IF NOT EXISTS issue_thread_interactions (
    id                   TEXT PRIMARY KEY,
    company_id           TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    issue_id             TEXT NOT NULL REFERENCES issues(id) ON DELETE SET NULL,
    agent_id             TEXT REFERENCES agents(id) ON DELETE SET NULL,
    comment_id           TEXT REFERENCES comments(id) ON DELETE SET NULL,
    run_id               TEXT REFERENCES heartbeat_runs(id) ON DELETE SET NULL,
    kind                 TEXT NOT NULL,
    status               TEXT NOT NULL CHECK (status IN ('pending', 'resolved')),
    idempotency_key      TEXT NOT NULL,
    result               TEXT,
    resolved_at          TEXT,
    resolved_by_agent_id TEXT REFERENCES agents(id) ON DELETE SET NULL,
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL,
    UNIQUE(issue_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS issue_thread_interactions_company_idx    ON issue_thread_interactions(company_id);
CREATE INDEX IF NOT EXISTS issue_thread_interactions_issue_idx      ON issue_thread_interactions(issue_id);
CREATE INDEX IF NOT EXISTS issue_thread_interactions_status_idx     ON issue_thread_interactions(status);
CREATE INDEX IF NOT EXISTS issue_thread_interactions_idempotency_idx ON issue_thread_interactions(idempotency_key);
