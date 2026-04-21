-- Add configuration column to agents table
ALTER TABLE agents ADD COLUMN configuration TEXT DEFAULT '{}';
