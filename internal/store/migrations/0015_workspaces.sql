-- Execution workspaces for capturing runtime artifact locations
-- Enables agent to track distinct filesystem workspaces during execution

CREATE TABLE IF NOT EXISTS execution_workspaces (
    id            TEXT PRIMARY KEY,
    company_id    TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    agent_id      TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    issue_id      TEXT REFERENCES issues(id) ON DELETE SET NULL,
    path          TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'error')),
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    UNIQUE(agent_id, path)
);

CREATE INDEX IF NOT EXISTS execution_workspaces_company_idx ON execution_workspaces(company_id);
CREATE INDEX IF NOT EXISTS execution_workspaces_agent_idx ON execution_workspaces(agent_id);
CREATE INDEX IF NOT EXISTS execution_workspaces_status_idx ON execution_workspaces(status);

-- Add workspace_id to heartbeat_runs
ALTER TABLE heartbeat_runs ADD COLUMN workspace_id TEXT REFERENCES execution_workspaces(id) ON DELETE SET NULL;
