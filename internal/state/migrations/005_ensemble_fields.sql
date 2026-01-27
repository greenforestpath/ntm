-- NTM State Store: Ensemble Field Extensions
-- Version: 005
-- Description: Adds synthesis output/error and assignment timestamps

ALTER TABLE ensemble_sessions ADD COLUMN synthesis_output TEXT;
ALTER TABLE ensemble_sessions ADD COLUMN error TEXT;

ALTER TABLE mode_assignments ADD COLUMN assigned_at TIMESTAMP;
ALTER TABLE mode_assignments ADD COLUMN completed_at TIMESTAMP;
ALTER TABLE mode_assignments ADD COLUMN error TEXT;
