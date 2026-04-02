// Package syncstate persists the nightly sync job's last successful completion
// time to disk so that the next run knows how far back to fetch data.
// It also provides cross-process lock and trigger file helpers used to
// coordinate between fhpbioguide (consumer) and fhpreports (trigger writer).
package syncstate

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"
)

const stateFile = "sync_state.json"

// staleLockAge is how old a lock file must be before it is considered stale
// and removed automatically — guards against crashes leaving a permanent lock.
const staleLockAge = 4 * time.Hour

type syncState struct {
	LastSyncCompleted time.Time `json:"last_sync_completed"`
}

// ReadState returns the timestamp of the last completed sync.
// Falls back to 48 hours ago if the file is missing or unreadable,
// which causes the next run to perform a short backfill automatically.
func ReadState(log *slog.Logger) time.Time {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		fallback := time.Now().Add(-48 * time.Hour)
		log.Warn("no state file found, defaulting", "fallback", fallback.Format("2006-01-02T15:04:05"), "component", "syncstate")
		return fallback
	}
	var s syncState
	if err := json.Unmarshal(data, &s); err != nil || s.LastSyncCompleted.IsZero() {
		fallback := time.Now().Add(-48 * time.Hour)
		log.Warn("could not parse state file, defaulting", "fallback", fallback.Format("2006-01-02T15:04:05"), "component", "syncstate", "err", err)
		return fallback
	}
	log.Debug("last sync completed", "timestamp", s.LastSyncCompleted.Format("2006-01-02T15:04:05"), "component", "syncstate")
	return s.LastSyncCompleted
}

// WriteState persists t as the last completed sync time.
// Uses an atomic write (temp file → rename) to prevent corruption
// if the process is killed during the write.
func WriteState(t time.Time) error {
	data, err := json.MarshalIndent(syncState{LastSyncCompleted: t}, "", "  ")
	if err != nil {
		return err
	}
	tmp := stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, stateFile)
}

// AcquireLock attempts to create an exclusive lock file at lockFile.
// Returns (true, nil) when the lock is successfully acquired.
// Returns (false, nil) when another process already holds the lock.
// Stale locks older than staleLockAge are removed automatically to
// recover from crashes that left the lock file behind.
func AcquireLock(lockFile string, log *slog.Logger) (bool, error) {
	if info, err := os.Stat(lockFile); err == nil {
		age := time.Since(info.ModTime())
		if age > staleLockAge {
			log.Warn("removing stale lock", "age", age.Round(time.Minute), "component", "syncstate")
			os.Remove(lockFile)
		} else {
			return false, nil // lock held by another process
		}
	}
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	f.Close()
	return true, nil
}

// ReleaseLock removes the lock file. Safe to call even if the file is gone.
func ReleaseLock(lockFile string, log *slog.Logger) {
	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		log.Error("failed to release lock", "err", err, "component", "syncstate")
	}
}

// LockHeld returns true if the lock file currently exists and is not stale.
func LockHeld(lockFile string) bool {
	info, err := os.Stat(lockFile)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) <= staleLockAge
}

// WriteTrigger creates the trigger file so fhpbioguide's poller picks it up.
// Returns an error if the file already exists (trigger already pending).
func WriteTrigger(triggerFile string) error {
	f, err := os.OpenFile(triggerFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

// TriggerPending returns true if the trigger file exists.
func TriggerPending(triggerFile string) bool {
	_, err := os.Stat(triggerFile)
	return err == nil
}

// ClearTrigger removes the trigger file. Safe to call even if it is gone.
func ClearTrigger(triggerFile string) {
	os.Remove(triggerFile)
}
