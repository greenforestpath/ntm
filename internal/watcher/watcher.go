// Package watcher provides file watching with debouncing using fsnotify.
package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ErrClosed is returned when operations are called on a closed Watcher.
var ErrClosed = errors.New("watcher: watcher is closed")

// EventType represents the type of file system event.
type EventType uint32

const (
	// Create is triggered when a file or directory is created.
	Create EventType = 1 << iota
	// Write is triggered when a file is modified.
	Write
	// Remove is triggered when a file or directory is removed.
	Remove
	// Rename is triggered when a file or directory is renamed.
	Rename
	// Chmod is triggered when file permissions change.
	Chmod
	// All events.
	All = Create | Write | Remove | Rename | Chmod
)

// Event represents a file system event.
type Event struct {
	// Path is the absolute path to the file or directory.
	Path string
	// Type is the type of event.
	Type EventType
	// IsDir is true if the event is for a directory.
	IsDir bool
}

// eventTypeFromFsnotify converts fsnotify.Op to EventType.
func eventTypeFromFsnotify(op fsnotify.Op) EventType {
	var t EventType
	if op.Has(fsnotify.Create) {
		t |= Create
	}
	if op.Has(fsnotify.Write) {
		t |= Write
	}
	if op.Has(fsnotify.Remove) {
		t |= Remove
	}
	if op.Has(fsnotify.Rename) {
		t |= Rename
	}
	if op.Has(fsnotify.Chmod) {
		t |= Chmod
	}
	return t
}

// Handler is called when a file system event occurs.
// Multiple events may be coalesced into a single call due to debouncing.
type Handler func(events []Event)

// ErrorHandler is called when a watch error occurs.
type ErrorHandler func(err error)

// Watcher watches files and directories for changes.
type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	debouncer    *Debouncer
	handler      Handler
	errorHandler ErrorHandler
	eventFilter  EventType
	recursive    bool

	mu            sync.Mutex
	watchedPaths  map[string]bool
	pendingEvents []Event
	closed        bool
}

// New creates a new Watcher.
// By default, all event types are watched. Use WithEventFilter to filter events.
// Use WithRecursive to watch directories recursively.
func New(handler Handler, opts ...Option) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher:    fsWatcher,
		debouncer:    NewDebouncer(DefaultDebounceDuration),
		handler:      handler,
		eventFilter:  All,
		watchedPaths: make(map[string]bool),
	}

	for _, opt := range opts {
		opt(w)
	}

	go w.run()

	return w, nil
}

// Option configures a Watcher.
type Option func(*Watcher)

// WithDebounceDuration sets the debounce duration for coalescing events.
// Use a real time.Duration to avoid hidden unit conversions.
func WithDebounceDuration(d time.Duration) Option {
	return func(w *Watcher) {
		if d > 0 {
			w.debouncer = NewDebouncer(d)
		}
	}
}

// WithDebouncer sets a custom debouncer.
func WithDebouncer(d *Debouncer) Option {
	return func(w *Watcher) {
		if d != nil {
			w.debouncer = d
		}
	}
}

// WithEventFilter sets which event types to watch.
func WithEventFilter(filter EventType) Option {
	return func(w *Watcher) {
		w.eventFilter = filter
	}
}

// WithRecursive enables recursive watching of directories.
func WithRecursive(recursive bool) Option {
	return func(w *Watcher) {
		w.recursive = recursive
	}
}

// WithErrorHandler sets the error handler.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(w *Watcher) {
		w.errorHandler = handler
	}
}

// Add adds a path to the watcher.
// If the path is a directory and recursive is enabled, all subdirectories are also watched.
func (w *Watcher) Add(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrClosed
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if w.watchedPaths[absPath] {
		return nil // Already watching
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() && w.recursive {
		return w.addRecursive(absPath)
	}

	if err := w.fsWatcher.Add(absPath); err != nil {
		return err
	}
	w.watchedPaths[absPath] = true

	return nil
}

// addRecursive adds a directory and all its subdirectories to the watcher.
// Must be called with w.mu held.
func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't access, but continue walking
			if w.errorHandler != nil {
				w.errorHandler(fmt.Errorf("walking %s: %w", path, err))
			}
			return filepath.SkipDir
		}
		if d.IsDir() {
			if w.watchedPaths[path] {
				return nil
			}
			if err := w.fsWatcher.Add(path); err != nil {
				// Report error but continue
				if w.errorHandler != nil {
					w.errorHandler(fmt.Errorf("watching %s: %w", path, err))
				}
				// If we can't watch this directory, we probably can't watch its children?
				// fsnotify might fail for other reasons (limit reached).
				// We'll try to continue.
				return nil
			}
			w.watchedPaths[path] = true
		}
		return nil
	})
}

// Remove removes a path from the watcher.
func (w *Watcher) Remove(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrClosed
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if !w.watchedPaths[absPath] {
		return nil // Not watching
	}

	if err := w.fsWatcher.Remove(absPath); err != nil {
		return err
	}
	delete(w.watchedPaths, absPath)

	return nil
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true
	w.debouncer.Cancel()
	return w.fsWatcher.Close()
}

// WatchedPaths returns a slice of all currently watched paths.
func (w *Watcher) WatchedPaths() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	paths := make([]string, 0, len(w.watchedPaths))
	for p := range w.watchedPaths {
		paths = append(paths, p)
	}
	return paths
}

// run processes events from fsnotify.
func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			if w.errorHandler != nil {
				w.errorHandler(err)
			}
		}
	}
}

// handleEvent processes a single fsnotify event.
func (w *Watcher) handleEvent(fsEvent fsnotify.Event) {
	eventType := eventTypeFromFsnotify(fsEvent.Op)

	// Filter events
	if eventType&w.eventFilter == 0 {
		return
	}

	// Check if it's a directory
	isDir := false
	if info, err := os.Stat(fsEvent.Name); err == nil {
		isDir = info.IsDir()
	}

	event := Event{
		Path:  fsEvent.Name,
		Type:  eventType,
		IsDir: isDir,
	}

	// If recursive and a new directory was created, watch it
	if w.recursive && isDir && eventType&Create != 0 {
		w.mu.Lock()
		if !w.closed && !w.watchedPaths[fsEvent.Name] {
			if err := w.fsWatcher.Add(fsEvent.Name); err != nil {
				// Report error via error handler if available
				if w.errorHandler != nil {
					w.errorHandler(err)
				}
			} else {
				w.watchedPaths[fsEvent.Name] = true
			}
		}
		w.mu.Unlock()
	}

	// If a watched directory was removed, clean up
	if eventType&Remove != 0 || eventType&Rename != 0 {
		w.mu.Lock()
		if w.watchedPaths[fsEvent.Name] {
			delete(w.watchedPaths, fsEvent.Name)
		}
		w.mu.Unlock()
	}

	w.mu.Lock()
	w.pendingEvents = append(w.pendingEvents, event)
	w.mu.Unlock()

	w.debouncer.Trigger(func() {
		w.mu.Lock()
		toDeliver := w.pendingEvents
		w.pendingEvents = nil
		w.mu.Unlock()

		if len(toDeliver) > 0 && w.handler != nil {
			w.handler(toDeliver)
		}
	})
}
