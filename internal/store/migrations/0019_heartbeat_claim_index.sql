CREATE UNIQUE INDEX IF NOT EXISTS heartbeat_runs_agent_inflight
    ON heartbeat_runs(agent_id)
    WHERE status = 'running';
