-- Approvals table for human-in-loop approval gates.
-- Tracks approval requests with optional request/response bodies for context.

CREATE TABLE IF NOT EXISTS approvals (
    id              TEXT PRIMARY KEY,
    company_id      TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    agent_id        TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    issue_id        TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    request_body    TEXT,
    response_body   TEXT,
    created_at      TEXT NOT NULL,
    resolved_at     TEXT,
    UNIQUE(id),
    CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX IF NOT EXISTS approvals_company_idx ON approvals(company_id);
CREATE INDEX IF NOT EXISTS approvals_agent_idx ON approvals(agent_id);
CREATE INDEX IF NOT EXISTS approvals_issue_idx ON approvals(issue_id);
CREATE INDEX IF NOT EXISTS approvals_status_idx ON approvals(status);
