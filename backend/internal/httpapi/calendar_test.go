package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/denisolnce/porta-pagi/backend/internal/config"
	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

func TestOccursOn(t *testing.T) {
	cases := []struct {
		start, recur, day string
		want              bool
	}{
		{"2026-06-06T09:00", "", "2026-06-06", true},
		{"2026-06-06T09:00", "", "2026-06-13", false},
		{"2026-06-06T09:00", "weekly", "2026-09-05", true},
		{"2026-06-06T09:00", "weekly", "2026-09-04", false},
		{"2026-06-06T09:00", "weekly", "2026-05-30", false}, // before the series began
		{"2026-01-31T18:00", "monthly", "2026-02-28", true}, // a short month's last day
		{"2026-01-31T18:00", "monthly", "2026-03-31", true},
		{"2026-01-31T18:00", "monthly", "2026-03-30", false},
		{"2000-02-29", "yearly", "2027-02-28", true}, // no 29th that year
		{"2000-02-29", "yearly", "2028-02-28", false},
		{"2000-02-29", "yearly", "2028-02-29", true},
		{"1987-09-04T06:20", "yearly", "2026-09-04", true},
		{"1987-09-04T06:20", "daily", "2026-09-04", false},
	}
	for _, c := range cases {
		if got := occursOn(c.start, c.recur, c.day); got != c.want {
			t.Errorf("occursOn(%s, %q, %s) = %v", c.start, c.recur, c.day, got)
		}
	}
}

// TestSeries: a weekly work party is answered one Saturday at a time, comes in
// the list whatever its first day, and moving how it repeats stales answers.
func TestSeries(t *testing.T) {
	_, _, steward, volk := newVillage(t) // now: Friday 2026-09-04
	code, o, _ := steward.do("POST", "/api/events", map[string]any{"title": "Skupno kuhanje", "kind": "work", "starts_at": "2026-06-06T09:00", "ends_at": "2026-06-06T12:00", "recur": "weekly"})
	steward.must(201, code, "weekly work party")
	id := itoa(o["id"].(float64))
	code, _, _ = steward.do("POST", "/api/events", map[string]any{"title": "x", "starts_at": "2026-09-10T09:00", "recur": "daily"})
	steward.must(400, code, "an unknown repeat")

	// Ninety days old and still listed: its Saturdays are ahead.
	listed := func() map[string]any {
		_, _, rows := volk.do("GET", "/api/events", nil)
		for _, r := range rows {
			if itoa(r["id"].(float64)) == id {
				return r
			}
		}
		t.Fatal("the series left the list")
		return nil
	}
	if ev := listed(); ev["recur"] != "weekly" {
		t.Fatalf("recur: %v", ev["recur"])
	}

	volk.must(204, first(volk.do("POST", "/api/events/"+id+"/signup", map[string]any{"state": "yes", "occurs_on": "2026-09-05"})), "a Saturday")
	volk.must(400, first(volk.do("POST", "/api/events/"+id+"/signup", map[string]any{"state": "yes", "occurs_on": "2026-09-04"})), "a Friday")
	volk.must(400, first(volk.do("POST", "/api/events/"+id+"/signup", map[string]any{"state": "yes"})), "no day")
	volk.must(204, first(volk.do("POST", "/api/events/"+id+"/signup", map[string]any{"state": "maybe", "occurs_on": "2026-09-12"})), "the next Saturday")

	var list []map[string]any
	json.Unmarshal([]byte(listed()["signup_list"].(string)), &list)
	days := map[string]string{}
	for _, a := range list {
		days[a["occurs_on"].(string)+" "+a["name"].(string)] = a["state"].(string)
	}
	if days["2026-09-05 Zeleni Volk"] != "yes" || days["2026-09-12 Zeleni Volk"] != "maybe" || days["2026-06-06 S"] != "yes" {
		t.Fatalf("answers per day: %v", days)
	}
	if ev := listed(); ev["signups"].(float64) != 0 || ev["mine"] != nil {
		t.Fatalf("a series' answers leaked into the one-off counts: %v %v", ev["signups"], ev["mine"])
	}

	volk.must(204, first(volk.do("DELETE", "/api/events/"+id+"/signup?on=2026-09-12", nil)), "take one day back")
	json.Unmarshal([]byte(listed()["signup_list"].(string)), &list)
	if len(list) != 2 {
		t.Fatalf("taking one day back took %d", 3-len(list))
	}

	// Monthly now: the Saturdays answered no longer exist as such.
	volk.must(204, first(volk.do("PUT", "/api/events/"+id, map[string]any{"recur": "monthly"})), "repeat monthly")
	json.Unmarshal([]byte(listed()["signup_list"].(string)), &list)
	for _, a := range list {
		if a["stale"].(float64) != 1 {
			t.Fatalf("an answer survived the change of repeat: %v", a)
		}
	}

	// A one-off ignores a day sent with the answer.
	_, o, _ = steward.do("POST", "/api/events", map[string]any{"title": "Film", "starts_at": "2026-09-10T20:00"})
	once := itoa(o["id"].(float64))
	volk.must(204, first(volk.do("POST", "/api/events/"+once+"/signup", map[string]any{"state": "no", "occurs_on": "2026-09-10"})), "one-off")
	_, _, rows := volk.do("GET", "/api/events", nil)
	for _, r := range rows {
		if itoa(r["id"].(float64)) == once && r["mine"] != "no" {
			t.Fatalf("one-off answer: %v", r["mine"])
		}
	}
}

// TestBirthdays: a date to know, not a gathering. Written down without a
// ring, answered by nobody, rung on the morning of the day, once.
func TestBirthdays(t *testing.T) {
	srv, fake, steward, volk := newVillage(t) // now: 2026-09-04 10:00
	for i, c := range []*client{steward, volk} {
		c.must(204, first(c.do("POST", "/api/push/subscribe", map[string]any{
			"endpoint": "https://push.example/" + itoa(float64(i)), "lang": "sl", "keys": map[string]any{"p256dh": "p", "auth": "a"}})), "subscribe")
	}
	code, o, _ := volk.do("POST", "/api/events", map[string]any{"title": "Ana", "kind": "birthday", "starts_at": "1987-09-04T06:20", "ends_at": "1987-09-04T08:00", "place": "x", "recur": ""})
	volk.must(201, code, "birthday")
	id := itoa(o["id"].(float64))
	volk.must(400, first(volk.do("POST", "/api/events", map[string]any{"title": "B", "kind": "birthday", "starts_at": "1987-13-40"})), "not a date")
	time.Sleep(100 * time.Millisecond)
	if fake.count() != 0 {
		t.Fatalf("writing a birthday down rang %d phones", fake.count())
	}
	_, _, rows := steward.do("GET", "/api/events", nil)
	var b map[string]any
	for _, r := range rows {
		if itoa(r["id"].(float64)) == id {
			b = r
		}
	}
	if b == nil || b["recur"] != "yearly" || b["ends_at"] != nil || b["place"] != "" || b["signup_list"] != "[]" {
		t.Fatalf("birthday row: %v", b)
	}
	steward.must(400, first(steward.do("POST", "/api/events/"+id+"/signup", map[string]any{"state": "yes", "occurs_on": "2026-09-04"})), "answer a birthday")
	steward.must(400, first(steward.do("PUT", "/api/events/"+id, map[string]any{"kind": "event"})), "birthday to event")
	steward.must(400, first(steward.do("PUT", "/api/events/"+id, map[string]any{"recur": ""})), "a birthday one-off")
	steward.must(204, first(steward.do("PUT", "/api/events/"+id, map[string]any{"notes": "rada ima borovnice"})), "edit a birthday")

	// Before seven nothing rings, and the day is not used up.
	srv.now = func() time.Time { return time.Date(2026, 9, 4, 6, 30, 0, 0, time.Local) }
	if n, err := srv.ringBirthdays(context.Background()); err != nil || n != 0 {
		t.Fatalf("rang at 06:30: %d %v", n, err)
	}
	// A house that switched birthdays off hears nothing.
	steward.must(204, first(steward.do("PUT", "/api/me/prefs", map[string]any{"off": []any{"birthdays"}})), "switch off")
	srv.now = func() time.Time { return time.Date(2026, 9, 4, 7, 0, 0, 0, time.Local) }
	if n, err := srv.ringBirthdays(context.Background()); err != nil || n != 1 {
		t.Fatalf("morning ring: %d %v", n, err)
	}
	waitFor(t, 1, fake)
	fake.mu.Lock()
	var p Payload
	json.Unmarshal(fake.payloads[0], &p)
	if len(fake.sent) != 1 || p.Title != "🎂 Ana" || p.Body != "danes ima rojstni dan" || p.URL != "#/tavern?day=2026-09-04" {
		t.Fatalf("ring: %d %+v", len(fake.sent), p)
	}
	fake.mu.Unlock()
	if n, _ := srv.ringBirthdays(context.Background()); n != 0 {
		t.Fatal("rang twice on one day")
	}
}

// TestAlmanac: a year written once, eclipses only where the village is known,
// listed from 60 days back, and a thread whose replies reach only the houses
// already in it.
func TestAlmanac(t *testing.T) {
	srv, fake, steward, volk := newVillage(t)
	srv.cfg.WeatherCoords = "45.6458,14.8496"
	ctx := context.Background()
	if err := srv.fillAlmanac(ctx, 2026); err != nil {
		t.Fatal(err)
	}
	count := func(q string) int64 {
		row, _ := srv.st.One(ctx, `SELECT count(*) AS n FROM almanac WHERE `+q)
		return row["n"].(int64)
	}
	if n := count(`kind='full_moon' AND starts_at LIKE '2026-%'`); n < 12 || n > 13 {
		t.Fatalf("full moons: %d", n)
	}
	if n := count(`kind LIKE '%equinox' OR kind LIKE '%solstice'`); n != 4 {
		t.Fatalf("seasons: %d", n)
	}
	if n := count(`kind LIKE '%eclipse'`); n == 0 {
		t.Fatal("no eclipse over Kočevje in 2026")
	}
	row, _ := srv.st.One(ctx, `SELECT detail FROM almanac WHERE kind='june_solstice'`)
	if !strings.Contains(row["detail"].(string), `"rise"`) {
		t.Fatalf("solstice without the day's length: %v", row["detail"])
	}
	all := count(`1`)
	if err := srv.fillAlmanac(ctx, 2026); err != nil || count(`1`) != all {
		t.Fatalf("a second fill wrote again: %v", err)
	}

	// No place, no eclipses: a year written without one says only what needs none.
	st, _ := store.Open(t.TempDir())
	defer st.Close()
	bare := New(st, config.Config{Languages: []string{"en"}})
	if err := bare.fillAlmanac(ctx, 2026); err != nil {
		t.Fatal(err)
	}
	if row, _ := bare.st.One(ctx, `SELECT count(*) AS n FROM almanac WHERE kind LIKE '%eclipse'`); row["n"].(int64) != 0 {
		t.Fatal("eclipses for a village with no place")
	}
	// The village names its place later: the year is written again, with its
	// eclipses, and the moons keep their rows.
	moons, _ := bare.st.One(ctx, `SELECT group_concat(id) AS ids FROM almanac WHERE kind='full_moon'`)
	bare.cfg.WeatherCoords = "45.6458,14.8496"
	if err := bare.fillAlmanac(ctx, 2026); err != nil {
		t.Fatal(err)
	}
	if row, _ := bare.st.One(ctx, `SELECT count(*) AS n FROM almanac WHERE kind LIKE '%eclipse'`); row["n"].(int64) == 0 {
		t.Fatal("a place named later brought no eclipses")
	}
	if again, _ := bare.st.One(ctx, `SELECT group_concat(id) AS ids FROM almanac WHERE kind='full_moon'`); again["ids"] != moons["ids"] {
		t.Fatal("the moons were written as new rows, and their threads with them")
	}
	// Easter 2026 is 5 April; the full moon of 2 April carries the day.
	if row, _ := srv.st.One(ctx, `SELECT starts_at FROM almanac WHERE kind='full_moon' AND detail LIKE '%"easter":"2026-04-05"%'`); row == nil || row["starts_at"].(string)[:10] != "2026-04-02" {
		t.Fatalf("Easter's full moon: %v", row)
	}

	_, _, rows := volk.do("GET", "/api/almanac", nil)
	if len(rows) == 0 || rows[0]["starts_at"].(string) < "2026-07-06" {
		t.Fatalf("listed from %v", rows[0]["starts_at"])
	}
	moon := ""
	for _, r := range rows {
		if r["title"] != almanacTitles[r["kind"].(string)] || r["title"] == "" {
			t.Fatalf("a row without its title: %v", r)
		}
		if r["kind"] == "full_moon" {
			moon = itoa(r["id"].(float64))
			break
		}
	}
	for i, c := range []*client{steward, volk} {
		c.must(204, first(c.do("POST", "/api/push/subscribe", map[string]any{
			"endpoint": "https://push.example/a" + itoa(float64(i)), "lang": "en", "keys": map[string]any{"p256dh": "p", "auth": "a"}})), "subscribe")
	}
	volk.must(201, first(volk.do("POST", "/api/threads/almanac/"+moon, map[string]any{"body": "Lepa luna nocoj"})), "comment")
	time.Sleep(100 * time.Millisecond)
	if fake.count() != 0 {
		t.Fatalf("the first word on the moon rang %d phones", fake.count())
	}
	steward.must(201, first(steward.do("POST", "/api/threads/almanac/"+moon, map[string]any{"body": "Res je"})), "reply")
	waitFor(t, 1, fake)
	fake.mu.Lock()
	defer fake.mu.Unlock()
	var p Payload
	json.Unmarshal(fake.payloads[0], &p)
	if len(fake.sent) != 1 || p.Title != "💬 S on: Full moon" {
		t.Fatalf("reply rang: %d %+v", len(fake.sent), p)
	}
}
