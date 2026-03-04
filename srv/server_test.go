package srv

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"srv.exe.dev/db/dbgen"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	tempDB := filepath.Join(t.TempDir(), "test.sqlite3")
	t.Cleanup(func() { os.Remove(tempDB) })
	s, err := New(tempDB)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return s
}

func createRule(t *testing.T, s *Server, host, path, dest string, rtype int, regex bool) dbgen.Redirect {
	t.Helper()
	isRegex := int64(0)
	if regex {
		isRegex = 1
	}
	body, _ := json.Marshal(map[string]any{
		"source_host":   host,
		"source_path":   path,
		"destination":   dest,
		"redirect_type": rtype,
		"is_regex":      isRegex,
	})
	req := httptest.NewRequest(http.MethodPost, "/_admin/api/redirects", bytes.NewReader(body))
	req.Header.Set("X-ExeDev-UserID", "admin")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleCreateRedirect(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create redirect: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var r dbgen.Redirect
	json.NewDecoder(w.Body).Decode(&r)
	return r
}

func TestAdminAuth(t *testing.T) {
	s := testServer(t)

	// Without auth header
	req := httptest.NewRequest(http.MethodGet, "/_admin/api/redirects", nil)
	w := httptest.NewRecorder()
	s.handleListRedirects(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// With auth header
	req = httptest.NewRequest(http.MethodGet, "/_admin/api/redirects", nil)
	req.Header.Set("X-ExeDev-UserID", "admin")
	w = httptest.NewRecorder()
	s.handleListRedirects(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCRUD(t *testing.T) {
	s := testServer(t)

	// Create
	r := createRule(t, s, "ggxy.co", "/discord", "https://discord.com/xyz", 302, false)
	if r.SourceHost != "ggxy.co" || r.SourcePath != "/discord" {
		t.Errorf("unexpected create result: %+v", r)
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/_admin/api/redirects", nil)
	req.Header.Set("X-ExeDev-UserID", "admin")
	w := httptest.NewRecorder()
	s.handleListRedirects(w, req)
	var list []dbgen.Redirect
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(list))
	}

	// Update
	updateBody, _ := json.Marshal(map[string]any{
		"source_host":   "ggxy.co",
		"source_path":   "/discord",
		"destination":   "https://discord.com/abc",
		"redirect_type": 301,
		"is_regex":      0,
	})
	req = httptest.NewRequest(http.MethodPut, "/_admin/api/redirects/1", bytes.NewReader(updateBody))
	req.Header.Set("X-ExeDev-UserID", "admin")
	req.SetPathValue("id", "1")
	w = httptest.NewRecorder()
	s.handleUpdateRedirect(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated dbgen.Redirect
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Destination != "https://discord.com/abc" {
		t.Errorf("expected updated destination, got %s", updated.Destination)
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/_admin/api/redirects/1", nil)
	req.Header.Set("X-ExeDev-UserID", "admin")
	req.SetPathValue("id", "1")
	w = httptest.NewRecorder()
	s.handleDeleteRedirect(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("delete: expected 204, got %d", w.Code)
	}
}

func TestExactRedirect(t *testing.T) {
	s := testServer(t)
	createRule(t, s, "ggxy.co", "/discord", "https://discord.com/xyz", 302, false)

	req := httptest.NewRequest(http.MethodGet, "/discord", nil)
	req.Host = "ggxy.co"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "https://discord.com/xyz" {
		t.Errorf("expected discord URL, got %s", loc)
	}
}

func TestExactRedirectWithQueryPreservation(t *testing.T) {
	s := testServer(t)
	createRule(t, s, "ggxy.co", "/", "https://ggxy.co", 301, false)

	req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
	req.Host = "ggxy.co"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "https://ggxy.co?foo=bar" {
		t.Errorf("expected query preserved, got %s", loc)
	}
}

func TestQueryNotAppendedWhenDestHasQuery(t *testing.T) {
	s := testServer(t)
	createRule(t, s, "outfrag.com", "/", "https://gamergalaxy.net/?utm_medium=outfrag", 301, false)

	req := httptest.NewRequest(http.MethodGet, "/?extra=1", nil)
	req.Host = "outfrag.com"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	loc := w.Header().Get("Location")
	if loc != "https://gamergalaxy.net/?utm_medium=outfrag" {
		t.Errorf("expected dest query untouched, got %s", loc)
	}
}

func TestRegexRedirect(t *testing.T) {
	s := testServer(t)
	createRule(t, s, "ggxy.co", `^/p/(.*)$`, "https://gamergalaxy.net/players/$1", 301, true)

	req := httptest.NewRequest(http.MethodGet, "/p/foobar", nil)
	req.Host = "ggxy.co"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "https://gamergalaxy.net/players/foobar" {
		t.Errorf("expected regex substitution, got %s", loc)
	}
}

func TestWildcardRedirect(t *testing.T) {
	s := testServer(t)
	createRule(t, s, "ggxy.co", "/p/*", "https://example.com/players/", 301, false)

	req := httptest.NewRequest(http.MethodGet, "/p/foo?bar=1", nil)
	req.Host = "ggxy.co"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "https://example.com/players/foo?bar=1" {
		t.Errorf("expected wildcard redirect, got %s", loc)
	}
}

func TestNoMatchReturns404(t *testing.T) {
	s := testServer(t)
	createRule(t, s, "ggxy.co", "/discord", "https://discord.com/xyz", 302, false)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	req.Host = "ggxy.co"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCatchAllHost(t *testing.T) {
	s := testServer(t)
	// Catch-all (empty host)
	createRule(t, s, "", "/health", "https://example.com/ok", 302, false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Host = "anything.com"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
}

func TestHostSpecificTakesPriority(t *testing.T) {
	s := testServer(t)
	createRule(t, s, "", "/test", "https://catchall.com", 302, false)
	createRule(t, s, "specific.com", "/test", "https://specific.com/result", 302, false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "specific.com"
	w := httptest.NewRecorder()
	s.handleRedirect(w, req)

	loc := w.Header().Get("Location")
	if loc != "https://specific.com/result" {
		t.Errorf("expected host-specific rule to win, got %s", loc)
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"ggxy.co", "ggxy.co"},
		{"ggxy.co:443", "ggxy.co"},
		{"localhost:8080", "localhost"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = tt.host
		got := extractHost(req)
		if got != tt.want {
			t.Errorf("extractHost(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}
