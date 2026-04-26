-- Rename activity_log columns to match TS schema (actorType, entityType)
-- SQLite RENAME COLUMN requires 3.25.0+
ALTER TABLE activity_log RENAME COLUMN actor_kind TO actor_type;
ALTER TABLE activity_log RENAME COLUMN entity_kind TO entity_type;
