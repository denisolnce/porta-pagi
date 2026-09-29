//go:build demo

package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

// The door with no code, compiled into the demo image only (-tags demo). A
// visitor picks a house and is in as that house, no invite, no pairing; the
// page shows a switcher so both sides of every permission can be seen. This
// is the unauthenticated write surface every village's binary refuses to
// have — which is why it is fenced by compilation and a test pins its
// absence — and the demo ships as its own image, porta-pagi-demo.
//
// On the hour the village is wiped and seeded again, so a visitor may click
// everything, Delete included, and the next one finds the village whole.

func init() {
	DemoMode = true
	demoRoutes = func(s *Server) {
		s.mux.HandleFunc("GET /api/demo/houses", s.demoHouses)
		s.mux.HandleFunc("POST /api/demo/enter", s.demoEnter)
		s.demoLocked = true
	}
	DemoStart = func(ctx context.Context, s *Server) { s.RunDemoReset(ctx) }
}

func (s *Server) demoHouses(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.Rows(r.Context(), `SELECT id, name, crest, color, is_steward FROM houses WHERE kind='house' ORDER BY is_steward DESC, name`)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, rows)
}

// demoEnter hands out a device token for the named house. Nothing is checked:
// that is the point of the demo, and the reason this file is not in a
// village's build.
func (s *Server) demoEnter(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	id, _ := m["id"].(float64)
	row, err := s.st.One(r.Context(), `SELECT id FROM houses WHERE kind='house' AND id=?`, int64(id))
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeErr(w, 404, "no such house")
		return
	}
	// A public door mints rows; the same cap /api/join has keeps a loop from
	// filling the table before the hour resets it.
	if !s.tries.allow(clientIP(r), 60, 10*time.Minute) {
		writeErr(w, 429, "slow down")
		return
	}
	tok, err := s.newDevice(r.Context(), row["id"].(int64), "visitor")
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"token": tok})
}

// RunDemoReset wipes and reseeds the village at the top of every hour.
func (s *Server) RunDemoReset(ctx context.Context) {
	for {
		now := s.now()
		next := now.Truncate(time.Hour).Add(time.Hour)
		select {
		case <-ctx.Done():
			return
		case <-time.After(next.Sub(now)):
		}
		if err := s.ResetDemo(); err != nil {
			log.Printf("demo reset failed: %v", err)
		} else {
			log.Print("demo village reset")
		}
	}
}

// ResetDemo empties every table the village writes and seeds the demo again,
// as one transaction: no request lands on an empty village in between. The
// table list comes from the schema itself, so a table added tomorrow is
// wiped too; only the migrations ledger and settings (the VAPID keys) stay.
// Every foreign key in the schema cascades or nulls on delete, so the order
// is safe today; deferring the checks is a fence for a key that one day is
// not.
func (s *Server) ResetDemo() error {
	ctx := context.Background()
	tables, err := s.st.Rows(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name NOT IN ('schema_migrations', 'settings') AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return err
	}
	err = s.st.Tx(ctx, func(tx *store.Store) error {
		if _, err := tx.Exec(ctx, `PRAGMA defer_foreign_keys = ON`); err != nil {
			return err
		}
		for _, t := range tables {
			if _, err := tx.Exec(ctx, `DELETE FROM "`+t["name"].(string)+`"`); err != nil {
				return err
			}
		}
		_, err := seedDemo(ctx, tx, s.now())
		return err
	})
	if err != nil {
		return err
	}
	wcache.mu.Lock()
	wcache.body = nil
	wcache.mu.Unlock()
	return nil
}
