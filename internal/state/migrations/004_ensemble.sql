-- NTM State Store: Ensemble Schema
-- Version: 004
-- Description: Creates tables for ensemble sessions and mode assignments

-- Ensemble sessions table: tracks ensemble runs tied to tmux sessions
CREATE TABLE IF NOT EXISTS ensemble_sessions (
    id INTEGER PRIMARY KEY,
    session_name TEXT NOT NULL UNIQUE,
    question TEXT NOT NULL,
    preset_used TEXT,
    status TEXT NOT NULL,
    synthesis_strategy TEXT,
    created_at TIMESTAMP NOT NULL,
    synthesized_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ensemble_session_name ON ensemble_sessions(session_name);
CREATE INDEX IF NOT EXISTS idx_ensemble_status ON ensemble_sessions(status);

-- Mode assignments table: tracks mode-to-pane assignments for ensembles
CREATE TABLE IF NOT EXISTS mode_assignments (
    id INTEGER PRIMARY KEY,
    ensemble_id INTEGER NOT NULL,
    mode_id TEXT NOT NULL,
    pane_name TEXT NOT NULL,
    agent_type TEXT NOT NULL,
    status TEXT NOT NULL,
    output_path TEXT,
    FOREIGN KEY (ensemble_id) REFERENCES ensemble_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_mode_assignments_ensemble_id ON mode_assignments(ensemble_id);
CREATE INDEX IF NOT EXISTS idx_mode_assignments_mode_id ON mode_assignments(mode_id);
