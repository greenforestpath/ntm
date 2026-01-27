package ensemble

import (
	"errors"
	"time"
)

// LoadSession loads an ensemble session state from SQLite.
func LoadSession(sessionName string) (*EnsembleSession, error) {
	if sessionName == "" {
		return nil, errors.New("session name is required")
	}

	store, err := defaultSQLiteStore()
	if err != nil {
		return nil, err
	}

	return store.Load(sessionName)
}

// SaveSession persists an ensemble session state to SQLite.
func SaveSession(sessionName string, state *EnsembleSession) error {
	if state == nil {
		return errors.New("ensemble state is nil")
	}
	if sessionName == "" {
		sessionName = state.SessionName
	}
	if sessionName == "" {
		return errors.New("session name is required")
	}

	if state.SessionName == "" {
		state.SessionName = sessionName
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now().UTC()
	}

	store, err := defaultSQLiteStore()
	if err != nil {
		return err
	}

	if err := store.Save(state); err != nil {
		return err
	}
	return nil
}
