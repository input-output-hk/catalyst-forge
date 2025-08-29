package mapping

// Placeholder for future fsnotify-based hot reload to keep this change small.
// We expose a no-op Reload for now; wiring can be added once needed.

// Reload attempts to re-load the mapping config from the same path.
func (e *Engine) Reload() error {
	// No-op in initial version.
	if e.logger != nil {
		e.logger.Debug("mapping reload noop")
	}
	return nil
}
