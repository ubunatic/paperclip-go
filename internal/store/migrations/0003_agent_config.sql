-- Add configuration column to agents table
ALTER TABLE agents ADD COLUMN configuration TEXT NOT NULL DEFAULT '{}';
-- Backfill existing rows with empty configuration
UPDATE agents SET configuration = '{}' WHERE configuration IS NULL;
