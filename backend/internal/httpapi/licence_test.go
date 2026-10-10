package httpapi

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/denisolnce/porta-pagi/backend/internal/config"
	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

// Signed by the owner's tool for an example domain, last day 2020-01-01
// (internal/licence's test says why it is long past): the villages below
// set their clock to 2019 to be licensed by it, and to 2020 to see it lapse.
const exampleKey = "PP1.eyJkb21haW4iOiJ0ZXN0LmV4YW1wbGUub3JnIiwidW50aWwiOiIyMDIwLTAxLTAxIn0.XIbQyRsjiKPKEmhE-umQQIJwlo7Adh66BLMAgknXfLWH5vQ89G2g8Npy5m5CXnl-bIaF8cdbBzOJibyIFQaNDw"

func licenceVillage(t *testing.T, key, on string) *client {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv := New(st, config.Config{BootstrapCode: "x", VillageName: "Testna vas", Languages: []string{"en"}, PublicURL: "https://test.example.org", LicenceKey: key})
	d, _ := time.Parse(time.DateOnly, on)
	srv.now = func() time.Time { return d.Add(12 * time.Hour) }
	c := &client{t: t, h: srv.Handler()}
	_, obj, _ := c.do("POST", "/api/bootstrap", map[string]any{"code": "x", "name": "House 1"})
	c.token = obj["token"].(string)
	return c
}

// addHouses creates houses 2..n after the bootstrap's first.
func addHouses(c *client, n int) {
	for i := 2; i <= n; i++ {
		code, _, _ := c.do("POST", "/api/houses", map[string]any{"name": fmt.Sprintf("House %d", i)})
		c.must(201, code, fmt.Sprintf("house %d", i))
	}
}

// TestLicenceCap: ten house accounts free; the eleventh is refused without a
// valid key, and only the creating is; common places do not count; the
// steward's line carries the state and never a count. The demo build is
// exempt.
func TestLicenceCap(t *testing.T) {
	c := licenceVillage(t, "", "2026-10-10")
	addHouses(c, 10)
	code, obj, _ := c.do("POST", "/api/houses", map[string]any{"name": "House 11"})
	if DemoMode {
		c.must(201, code, "the demo has no cap")
		_, obj, _ = c.do("GET", "/api/licence", nil)
		if obj["state"] != "demo" {
			t.Fatalf("demo line: %v", obj)
		}
		return
	}
	c.must(403, code, "the eleventh house without a key")
	if !strings.HasPrefix(fmt.Sprint(obj["error"]), "licence:") {
		t.Fatalf("the refusal names the licence: %v", obj)
	}
	code, _, _ = c.do("POST", "/api/houses", map[string]any{"name": "Village green", "kind": "common"})
	c.must(201, code, "a common place is land, not an account")
	code, _, _ = c.do("POST", "/api/houses", map[string]any{"name": "Sneaky", "kind": "barn"})
	c.must(403, code, "an unknown kind becomes a house and is counted")
	code, obj, _ = c.do("GET", "/api/licence", nil)
	c.must(200, code, "licence line")
	if obj["state"] != "free" || len(obj) != 1 {
		t.Fatalf("ten houses, no key: want only {state: free}, got %v", obj)
	}

	lic := licenceVillage(t, exampleKey, "2019-06-01")
	addHouses(lic, 12)
	_, obj, _ = lic.do("GET", "/api/licence", nil)
	if obj["state"] != "licensed" || obj["domain"] != "test.example.org" || obj["until"] != "2020-01-01" {
		t.Fatalf("licensed line: %v", obj)
	}

	lapsed := licenceVillage(t, exampleKey, "2020-01-02")
	addHouses(lapsed, 10)
	code, _, _ = lapsed.do("POST", "/api/houses", map[string]any{"name": "House 11"})
	lapsed.must(403, code, "a lapsed key refuses the next house")
	code, _, list := lapsed.do("GET", "/api/houses", nil)
	if code != 200 || len(list) != 10 {
		t.Fatalf("a lapsed key evicts nobody: %d houses", len(list))
	}
}
