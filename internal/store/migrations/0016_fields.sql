-- Add priority/estimate to issues
ALTER TABLE issues ADD COLUMN priority TEXT NOT NULL DEFAULT 'medium';
ALTER TABLE issues ADD COLUMN estimate INTEGER;

-- Add budget tracking to agents
ALTER TABLE agents ADD COLUMN budget_limit INTEGER;
ALTER TABLE agents ADD COLUMN budget_used INTEGER NOT NULL DEFAULT 0;

-- Add token/cost tracking to heartbeat_runs
ALTER TABLE heartbeat_runs ADD COLUMN prompt_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE heartbeat_runs ADD COLUMN completion_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE heartbeat_runs ADD COLUMN cost INTEGER NOT NULL DEFAULT 0;
