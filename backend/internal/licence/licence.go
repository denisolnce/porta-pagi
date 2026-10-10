// Package licence reads the key a village above ten houses runs with.
//
// The Business Source License grant (LICENSE) is free use for one village of
// up to ten house accounts. The licence text is the fence; this package only
// makes the honest path the easy one (porta-pagi-cloud docs/commercial.md
// §13, "The cap in code, honestly"):
//
//   - a key is the village's domain and a last day, signed by the owner with
//     a private key that is on no repo (porta-pagi-cloud ops/licence.mjs),
//     checked here against the public half — offline, nothing calls home;
//   - what a missing, lapsed or foreign key refuses is one thing only: the
//     steward's creating of the eleventh house. Nobody is evicted, nothing
//     else changes, and an invite already handed out keeps working;
//   - a build past its Change Date (four years after it was built) is AGPL,
//     asks for no key, and the check stands down.
package licence

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// FreeHouses is the grant: house accounts a village may have without a key.
// Common places (kind 'common') are land, not accounts, and are not counted.
const FreeHouses = 10

// publicKey is the half of the owner's signing key that checks a licence.
// The private half: ~/.config/porta-pagi/licence-ed25519.pem on the owner's
// machine and its password manager, never a repo.
var publicKey = mustKey("eH3b5MnSVBZAIPLxXZZ0hwVEzoO5RkO8nlhZpvuL5w0")

// BuildDate is the day this binary was built, set by the Dockerfile:
// -ldflags "-X .../internal/licence.BuildDate=2026-10-10". Empty in a
// `go run` or a test build, and then the check never stands down.
var BuildDate string

// State is what the steward's settings say about the licence, as one line.
type State string

const (
	Free        State = "free"         // no key, at most ten houses
	Outgrown    State = "outgrown"     // no usable key, more than ten houses
	Licensed    State = "licensed"     // a valid key for this village's domain
	OtherDomain State = "other_domain" // a valid key, for another domain
	Lapsed      State = "lapsed"       // a valid key past its last day
	Invalid     State = "invalid"      // a key that is not ours, or not a key
	Open        State = "open"         // past the Change Date: AGPL, no key asked
)

// Status is the licence as the steward reads it. It carries no count of
// houses: the line never says how many, only what the licence is.
type Status struct {
	State  State  `json:"state"`
	Domain string `json:"domain,omitempty"` // the key's, when there is one
	Until  string `json:"until,omitempty"`  // the key's last day, YYYY-MM-DD
}

// Key is what the owner signed.
type Key struct {
	Domain string `json:"domain"`
	Until  string `json:"until"`
}

// Parse checks a key's signature and reads it. It does not judge the domain
// or the date: Check does.
func Parse(raw string) (Key, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 3 || parts[0] != "PP1" {
		return Key{}, errors.New("not a licence key")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(publicKey, []byte(parts[0]+"."+parts[1]), sig) {
		return Key{}, errors.New("licence key not signed by the owner")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Key{}, err
	}
	var k Key
	if err := json.Unmarshal(body, &k); err != nil {
		return Key{}, err
	}
	if _, err := time.Parse(time.DateOnly, k.Until); err != nil || k.Domain == "" {
		return Key{}, errors.New("licence key without a domain or a last day")
	}
	return k, nil
}

// Check says what the licence is for a village at host (PUBLIC_URL's host)
// with this many house accounts, on the day now falls in.
func Check(raw, host string, houses int, now time.Time) Status {
	if standsDown(now) {
		return Status{State: Open}
	}
	if strings.TrimSpace(raw) == "" {
		if houses > FreeHouses {
			return Status{State: Outgrown}
		}
		return Status{State: Free}
	}
	k, err := Parse(raw)
	if err != nil {
		return Status{State: Invalid}
	}
	s := Status{Domain: k.Domain, Until: k.Until}
	switch {
	case !strings.EqualFold(k.Domain, host):
		s.State = OtherDomain
	case now.Format(time.DateOnly) > k.Until: // the last day counts in full
		s.State = Lapsed
	default:
		s.State = Licensed
	}
	return s
}

// AllowsAnother says whether a village with this many house accounts may
// gain one more: always under the grant, beyond it only with a valid key or
// past the Change Date.
func (s Status) AllowsAnother(houses int) bool {
	return houses < FreeHouses || s.State == Licensed || s.State == Open
}

// standsDown: four years after this build, the Change Date of the BSL, the
// version is AGPL and no key is asked.
func standsDown(now time.Time) bool {
	built, err := time.Parse(time.DateOnly, BuildDate)
	if err != nil {
		return false
	}
	return !now.Before(built.AddDate(4, 0, 0))
}

func mustKey(b64 string) ed25519.PublicKey {
	k, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil || len(k) != ed25519.PublicKeySize {
		panic("licence: bad public key")
	}
	return ed25519.PublicKey(k)
}
