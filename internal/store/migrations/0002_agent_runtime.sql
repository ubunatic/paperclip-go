-- Add runtime_state column to agents table
ALTER TABLE agents ADD COLUMN runtime_state TEXT NOT NULL DEFAULT 'idle';
