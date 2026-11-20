package syncstate

import (
	"io"
	"log/slog"
	"os"
	"testing"
	"time"
)

func inTempDir(t *testing.T) {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestDistributorStateIsIndependent(t *testing.T) {
	inTempDir(t)
	t1 := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 21, 2, 0, 0, 0, time.UTC)

	if err := WriteDistributorState(26, t1); err != nil {
		t.Fatal(err)
	}
	if err := WriteDistributorState(423, t2); err != nil {
		t.Fatal(err)
	}
	if got := ReadDistributorState(26, quiet); !got.Equal(t1) {
		t.Fatalf("26 = %v, want %v", got, t1)
	}
	if got := ReadDistributorState(423, quiet); !got.Equal(t2) {
		t.Fatalf("423 = %v, want %v (writing 423 must not touch 26)", got, t2)
	}
}

func TestDistributorFallsBackToGlobalState(t *testing.T) {
	inTempDir(t)
	global := time.Date(2026, 11, 19, 2, 0, 0, 0, time.UTC)
	// Old-format file: only last_sync_completed.
	if err := os.WriteFile(stateFile, []byte(`{"last_sync_completed":"2026-11-19T02:00:00Z"}`), 0644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{26, 423} {
		if got := ReadDistributorState(id, quiet); !got.Equal(global) {
			t.Fatalf("id %d = %v, want global %v", id, got, global)
		}
	}
}

func TestDistributorFallsBackTo48HoursWithoutFile(t *testing.T) {
	inTempDir(t)
	got := ReadDistributorState(423, quiet)
	if d := time.Since(got); d < 47*time.Hour || d > 49*time.Hour {
		t.Fatalf("fallback age = %v, want about 48h", d)
	}
}

func TestWriteStateKeepsDistributorEntries(t *testing.T) {
	inTempDir(t)
	t1 := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 21, 2, 0, 0, 0, time.UTC)
	if err := WriteDistributorState(423, t1); err != nil {
		t.Fatal(err)
	}
	if err := WriteState(t2); err != nil {
		t.Fatal(err)
	}
	if got := ReadDistributorState(423, quiet); !got.Equal(t1) {
		t.Fatalf("423 entry lost or changed: %v", got)
	}
	if got := ReadState(quiet); !got.Equal(t2) {
		t.Fatalf("global = %v, want %v", got, t2)
	}
}
