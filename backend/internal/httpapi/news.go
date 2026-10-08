package httpapi

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"strings"
)

// Portal News: a short note per change a villager would notice, written in
// the village's languages and shipped inside the binary (news/news.json,
// newest first). A village hears a note once, when the roll that brings it
// arrives: the round rings what settings has not marked as told, in one push
// whatever the number, and marks it. A village with no houses yet has nobody
// to tell, so it marks the past as told and rings nothing. The panel lists
// every note the binary carries. Design and the options refused:
// docs/design-news.md.

//go:embed news/news.json
var newsFS embed.FS

type newsText struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type newsEntry struct {
	ID    string              `json:"id"`
	On    string              `json:"on"`
	Texts map[string]newsText `json:"texts"`
}

var news = func() []newsEntry {
	raw, err := newsFS.ReadFile("news/news.json")
	if err != nil {
		panic(err)
	}
	var out []newsEntry
	if err := json.Unmarshal(raw, &out); err != nil {
		panic("news/news.json: " + err.Error())
	}
	return out
}()

// text is a note in lang, else in the village's first language that has it,
// else in English — the codex's way.
func (s *Server) newsText(e newsEntry, lang string) newsText {
	for _, l := range append([]string{lang}, append(s.cfg.Languages, "en")...) {
		if t, ok := e.Texts[l]; ok && t.Title != "" {
			return t
		}
	}
	return newsText{}
}

func (s *Server) listNews(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, news)
}

// ringNews tells every house what this binary brought and the village has
// not heard: one note by its own title and body, several by the kind's name
// and their titles. Quiet hours hold it until the morning.
func (s *Server) ringNews(ctx context.Context) (int, error) {
	if s.quietHours() {
		return 0, nil
	}
	rows, err := s.st.Rows(ctx, `SELECT key FROM settings WHERE key LIKE 'news:%'`)
	if err != nil {
		return 0, err
	}
	told := map[string]bool{}
	for _, r := range rows {
		told[strings.TrimPrefix(str(r, "key"), "news:")] = true
	}
	var fresh []newsEntry
	for _, e := range news {
		if !told[e.ID] {
			fresh = append(fresh, e)
		}
	}
	if len(fresh) == 0 {
		return 0, nil
	}
	houses, err := s.st.One(ctx, `SELECT count(*) AS n FROM houses WHERE kind='house'`)
	if err != nil {
		return 0, err
	}
	if houses["n"].(int64) > 0 {
		s.notify("news", 0, func(lang string) Payload {
			if len(fresh) == 1 {
				t := s.newsText(fresh[0], lang)
				return Payload{Title: "📰 " + t.Title, Body: snippet(t.Body, 120), URL: "#/?news"}
			}
			titles := make([]string, len(fresh))
			for i, e := range fresh {
				titles[i] = s.newsText(e, lang).Title
			}
			return Payload{Title: "📰 " + tr(lang, "Portal News"), Body: snippet(strings.Join(titles, " · "), 120), URL: "#/?news"}
		})
	}
	for _, e := range fresh {
		if _, err := s.st.Exec(ctx, `INSERT INTO settings(key, value) VALUES (?, datetime('now')) ON CONFLICT(key) DO NOTHING`, "news:"+e.ID); err != nil {
			return 0, err
		}
	}
	if houses["n"].(int64) == 0 {
		return 0, nil
	}
	return len(fresh), nil
}
