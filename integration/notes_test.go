package integration

import (
	"context"
	"encoding/json"
	"example.com/pr16/internal/core"
	"example.com/pr16/internal/httpapi"
	"example.com/pr16/internal/repo"
	"example.com/pr16/internal/service"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func setup(t *testing.T) (*httptest.Server, *repo.Postgres) {
	t.Helper()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("DB_DSN is required: make integration")
	}
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, e := repo.Open(c, dsn, 4)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { p.DB.Close() })
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	p.DB.SetMaxOpenConns(1)
	p.DB.SetMaxIdleConns(1)
	if _, e = p.DB.ExecContext(c, `CREATE SCHEMA `+schema+`;SET search_path TO `+schema+`;CREATE TABLE notes(id BIGSERIAL PRIMARY KEY,title TEXT NOT NULL,content TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now(),updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		_, e := p.DB.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		if e != nil {
			t.Error(e)
		}
	})
	h := &httpapi.Handler{Service: service.New(p)}
	s := httptest.NewServer(h.Router())
	t.Cleanup(s.Close)
	return s, p
}
func request(t *testing.T, s *httptest.Server, method, path, body string, status int) []byte {
	t.Helper()
	r, e := http.NewRequest(method, s.URL+path, strings.NewReader(body))
	if e != nil {
		t.Fatal(e)
	}
	r.Header.Set("Content-Type", "application/json")
	resp, e := s.Client().Do(r)
	if e != nil {
		t.Fatal(e)
	}
	defer resp.Body.Close()
	b, e := io.ReadAll(resp.Body)
	if e != nil {
		t.Fatal(e)
	}
	if resp.StatusCode != status {
		t.Fatalf("%s %s got %d want %d: %s", method, path, resp.StatusCode, status, b)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatal("missing JSON content type")
	}
	if status == 204 && len(b) != 0 {
		t.Fatal("204 must be empty")
	}
	t.Logf("%s %s -> %d %s", method, path, status, b)
	return b
}
func TestCRUD(t *testing.T) {
	s, p := setup(t)
	b := request(t, s, "POST", "/api/v1/notes", `{"title":"Hello","content":"World"}`, 201)
	var n core.Note
	if e := json.Unmarshal(b, &n); e != nil {
		t.Fatal(e)
	}
	if n.ID <= 0 || n.Title != "Hello" || n.Content != "World" || n.CreatedAt.IsZero() {
		t.Fatalf("bad note: %+v", n)
	}
	path := fmt.Sprintf("/api/v1/notes/%d", n.ID)
	request(t, s, "GET", path, "", 200)
	request(t, s, "PATCH", path, `{"content":"Updated"}`, 200)
	var content string
	if e := p.DB.QueryRow(`SELECT content FROM notes WHERE id=$1`, n.ID).Scan(&content); e != nil || content != "Updated" {
		t.Fatalf("DB update: %s %v", content, e)
	}
	request(t, s, "DELETE", path, "", 204)
	request(t, s, "GET", path, "", 404)
	var count int
	if e := p.DB.QueryRow(`SELECT count(*) FROM notes`).Scan(&count); e != nil || count != 0 {
		t.Fatalf("DB delete: %d %v", count, e)
	}
}
func TestValidationAndNotFound(t *testing.T) {
	s, _ := setup(t)
	for _, body := range []string{`{`, `{"title":" "}`, `{"title":"x","unknown":1}`, `{"title":"x"} {}`} {
		request(t, s, "POST", "/api/v1/notes", body, 400)
	}
	request(t, s, "GET", "/api/v1/notes/999999", "", 404)
	request(t, s, "GET", "/api/v1/notes/abc", "", 400)
	request(t, s, "PATCH", "/api/v1/notes/999999", `{"title":"x"}`, 404)
	request(t, s, "DELETE", "/api/v1/notes/999999", "", 404)
	request(t, s, "GET", "/api/v1/notes?limit=-1", "", 400)
}
func TestListFilterPagination(t *testing.T) {
	s, _ := setup(t)
	for _, title := range []string{"Go one", "Other", "Go two"} {
		request(t, s, "POST", "/api/v1/notes", fmt.Sprintf(`{"title":%q}`, title), 201)
	}
	for _, page := range []string{"1", "2"} {
		b := request(t, s, "GET", "/api/v1/notes?q=Go&limit=1&page="+page, "", 200)
		var notes []core.Note
		if e := json.Unmarshal(b, &notes); e != nil || len(notes) != 1 {
			t.Fatalf("page: %s %v", b, e)
		}
		want := "Go two"
		if page == "2" {
			want = "Go one"
		}
		if notes[0].Title != want {
			t.Fatalf("want %s got %s", want, notes[0].Title)
		}
	}
	request(t, s, "GET", "/api/v1/notes?q=missing", "", 200)
}
