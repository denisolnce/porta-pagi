package httpapi

import (
	"context"
	"net/http"
	"net/url"

	"github.com/denisolnce/porta-pagi/backend/internal/licence"
)

// The licence, as this village holds it (internal/licence). Two places use
// it: createHouse, which refuses the steward's eleventh house without a
// valid key, and the steward's ⚙️ settings, which read it as one line. No
// count leaves this file: the line says what the licence is, never how many
// houses the village has. The demo build is exempt — a visitor holding the
// steward's keys must not meet a sales wall, and the demo wipes itself
// hourly, so it is no way round the cap.

// licenceNow counts the house accounts and checks the key against them.
func (s *Server) licenceNow(ctx context.Context) (licence.Status, int, error) {
	row, err := s.st.One(ctx, `SELECT count(*) AS n FROM houses WHERE kind = 'house'`)
	if err != nil {
		return licence.Status{}, 0, err
	}
	n := int(row["n"].(int64))
	return licence.Check(s.cfg.LicenceKey, s.publicHost(), n, s.now()), n, nil
}

// mayAddHouse: whether the steward may create one more house account.
func (s *Server) mayAddHouse(ctx context.Context) (bool, error) {
	if DemoMode {
		return true, nil
	}
	st, n, err := s.licenceNow(ctx)
	if err != nil {
		return false, err
	}
	return st.AllowsAnother(n), nil
}

// LicenceState is the line the server logs at start, for whoever reads
// `docker logs`: the state only, never the key.
func (s *Server) LicenceState(ctx context.Context) string {
	if DemoMode {
		return "demo build, no cap"
	}
	st, _, err := s.licenceNow(ctx)
	if err != nil {
		return "unknown: " + err.Error()
	}
	if st.Domain != "" {
		return string(st.State) + " (" + st.Domain + " until " + st.Until + ")"
	}
	return string(st.State)
}

// getLicence: GET /api/licence, the steward's line in ⚙️.
func (s *Server) getLicence(w http.ResponseWriter, r *http.Request) {
	if DemoMode {
		writeJSON(w, 200, map[string]any{"state": "demo"})
		return
	}
	st, _, err := s.licenceNow(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, st)
}

func (s *Server) publicHost() string {
	if u, err := url.Parse(s.cfg.PublicURL); err == nil {
		return u.Hostname()
	}
	return ""
}
