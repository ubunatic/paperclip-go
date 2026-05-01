ALTER TABLE heartbeat_runs ADD COLUMN liveness_state TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN liveness_reason TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN continuation_attempt INTEGER NOT NULL DEFAULT 0;
ALTER TABLE heartbeat_runs ADD COLUMN last_useful_action_at TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN next_action TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN scheduled_retry_at TEXT;
ALTER TABLE heartbeat_runs ADD COLUMN scheduled_retry_attempt INTEGER NOT NULL DEFAULT 0;
ALTER TABLE heartbeat_runs ADD COLUMN scheduled_retry_reason TEXT;
