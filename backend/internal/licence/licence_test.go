package licence

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// A key the owner's tool signed with the real private key (porta-pagi-cloud
// ops/licence.mjs sign test.example.org 2027-10-10): the format on both sides
// is one format. It licenses an example domain only.
const realKey = "PP1.eyJkb21haW4iOiJ0ZXN0LmV4YW1wbGUub3JnIiwidW50aWwiOiIyMDI3LTEwLTEwIn0.g6-5Zve6KWTc62BLbrwC94JmqwOr9ReXwmCkqKN0eonjnJ9kU4Ak85gR_njwJtDIES7kxlvkZDGXHMiLq-yvDQ"

func day(s string) time.Time { t, _ := time.Parse(time.DateOnly, s); return t.Add(12 * time.Hour) }

// signWith makes a key with a throwaway pair, swapped in for the owner's.
func signWith(t *testing.T, domain, until string) string {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	old := publicKey
	publicKey = pub
	t.Cleanup(func() { publicKey = old })
	body, _ := json.Marshal(Key{Domain: domain, Until: until})
	msg := "PP1." + base64.RawURLEncoding.EncodeToString(body)
	return msg + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(msg)))
}

func TestTheOwnersToolSignsWhatThePortalReads(t *testing.T) {
	k, err := Parse(realKey)
	if err != nil || k.Domain != "test.example.org" || k.Until != "2027-10-10" {
		t.Fatalf("got %+v, %v", k, err)
	}
	if s := Check(realKey, "test.example.org", 40, day("2027-01-01")); s.State != Licensed {
		t.Fatalf("got %+v", s)
	}
}

func TestStates(t *testing.T) {
	key := signWith(t, "vas.example.org", "2027-10-10")
	for _, c := range []struct {
		name, raw, host string
		houses          int
		on              string
		want            State
		another         bool
	}{
		{"no key, room left", "", "vas.example.org", 9, "2026-10-10", Free, true},
		{"no key, the tenth is the last", "", "vas.example.org", 10, "2026-10-10", Free, false},
		{"no key, above the grant", "", "vas.example.org", 12, "2026-10-10", Outgrown, false},
		{"licensed", key, "vas.example.org", 30, "2026-10-10", Licensed, true},
		{"the host is compared without case", key, "VAS.example.org", 30, "2026-10-10", Licensed, true},
		{"the last day counts", key, "vas.example.org", 30, "2027-10-10", Licensed, true},
		{"lapsed: nobody goes, nobody new", key, "vas.example.org", 30, "2027-10-11", Lapsed, false},
		{"lapsed but under the grant", key, "vas.example.org", 4, "2027-10-11", Lapsed, true},
		{"another village's key", key, "other.example.org", 30, "2026-10-10", OtherDomain, false},
		{"not a key", "hello", "vas.example.org", 30, "2026-10-10", Invalid, false},
		{"a tampered key", key[:len(key)-2] + "AA", "vas.example.org", 30, "2026-10-10", Invalid, false},
	} {
		s := Check(c.raw, c.host, c.houses, day(c.on))
		if s.State != c.want || s.AllowsAnother(c.houses) != c.another {
			t.Errorf("%s: got %s, another=%v", c.name, s.State, s.AllowsAnother(c.houses))
		}
	}
}

func TestChangeDate(t *testing.T) {
	old := BuildDate
	t.Cleanup(func() { BuildDate = old })
	BuildDate = "2026-10-10"
	if s := Check("", "vas.example.org", 50, day("2030-10-09")); s.State != Outgrown {
		t.Fatalf("a day before the Change Date: got %s", s.State)
	}
	if s := Check("", "vas.example.org", 50, day("2030-10-10")); s.State != Open || !s.AllowsAnother(50) {
		t.Fatalf("on the Change Date: got %s", s.State)
	}
	BuildDate = ""
	if s := Check("", "vas.example.org", 50, day("2040-01-01")); s.State != Outgrown {
		t.Fatalf("a build with no date never stands down: got %s", s.State)
	}
}
