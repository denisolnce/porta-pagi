package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestTx: a failed transaction leaves nothing, and a transaction inside a
// transaction is refused rather than left to wait on the one connection.
func TestTx(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	boom := errors.New("boom")
	err = st.Tx(ctx, func(tx *Store) error {
		if _, err := tx.Exec(ctx, `INSERT INTO settings (key, value) VALUES ('probe', '1')`); err != nil {
			return err
		}
		// With a timeout: without the guard this would wait on the one
		// connection the outer transaction holds, and the test would hang
		// instead of failing. The sentinel tells a refusal from that timeout.
		inner, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := tx.Tx(inner, func(*Store) error { return nil }); !errors.Is(err, ErrNestedTx) {
			t.Errorf("a nested Tx: %v", err)
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Tx returned %v", err)
	}
	if row, _ := st.One(ctx, `SELECT value FROM settings WHERE key='probe'`); row != nil {
		t.Fatal("a rolled-back write is still there")
	}
	if err := st.Tx(ctx, func(tx *Store) error {
		_, err := tx.Exec(ctx, `INSERT INTO settings (key, value) VALUES ('probe', '2')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if row, _ := st.One(ctx, `SELECT value FROM settings WHERE key='probe'`); row == nil || row["value"] != "2" {
		t.Fatalf("a committed write is missing: %v", row)
	}
}

// TestFeaturesCarryCamp: migration 025 runs on a village that already has
// houses — the owner's — and keeps its campground on, with nobody named as
// having flipped it. A new village gets no row: the default, off.
func TestFeaturesCarryCamp(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if row, _ := st.One(ctx, `SELECT count(*) AS n FROM features`); row["n"].(int64) != 0 {
		t.Fatalf("a new village has a switch row: %v", row)
	}
	// Back to the moment before 025, with a house in the village.
	for _, q := range []string{`DROP TABLE features`, `DELETE FROM schema_migrations WHERE version LIKE '025%'`, `INSERT INTO houses(name) VALUES ('Old House')`} {
		if _, err := st.Exec(ctx, q); err != nil {
			t.Fatal(q, err)
		}
	}
	st.Close()
	st, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	row, err := st.One(ctx, `SELECT enabled, set_by, set_at FROM features WHERE key='camp'`)
	if err != nil || row == nil || row["enabled"].(int64) != 1 || row["set_by"] != nil || row["set_at"] != nil {
		t.Fatalf("the campground after 025 on a village with houses: %v %v", row, err)
	}
}
