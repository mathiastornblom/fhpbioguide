// Package syncstate persists the nightly sync job's last successful completion
// time to disk so that the next run knows how far back to fetch data.
package syncstate

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

const stateFile = "sync_state.json"

type syncState struct {
	LastSyncCompleted time.Time `json:"last_sync_completed"`
}

// ReadState returns the timestamp of the last completed sync.
// Falls back to 48 hours ago if the file is missing or unreadable,
// which causes the next run to perform a short backfill automatically.
func ReadState() time.Time {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		fallback := time.Now().Add(-48 * time.Hour)
		log.Printf("sync state: no state file found, defaulting to %s", fallback.Format("2006-01-02T15:04:05"))
		return fallback
	}
	var s syncState
	if err := json.Unmarshal(data, &s); err != nil || s.LastSyncCompleted.IsZero() {
		fallback := time.Now().Add(-48 * time.Hour)
		log.Printf("sync state: could not parse state file, defaulting to %s", fallback.Format("2006-01-02T15:04:05"))
		return fallback
	}
	log.Printf("sync state: last sync completed at %s", s.LastSyncCompleted.Format("2006-01-02T15:04:05"))
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
