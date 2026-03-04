package srv

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"srv.exe.dev/db"
	"srv.exe.dev/db/dbgen"
)

type Server struct {
	DB           *sql.DB
	Queries      *dbgen.Queries
	TemplatesDir string
	StaticDir    string
}

func New(dbPath string) (*Server, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)
	srv := &Server{
		TemplatesDir: filepath.Join(baseDir, "templates"),
		StaticDir:    filepath.Join(baseDir, "static"),
	}
	if err := srv.setUpDatabase(dbPath); err != nil {
		return nil, err
	}
	srv.Queries = dbgen.New(srv.DB)
	return srv, nil
}

func (s *Server) setUpDatabase(dbPath string) error {
	wdb, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	s.DB = wdb
	if err := db.RunMigrations(wdb); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func (s *Server) Serve(addr string) error {
	mux := http.NewServeMux()

	// Admin routes
	mux.HandleFunc("GET /_admin", s.handleAdmin)
	mux.HandleFunc("GET /_admin/api/redirects", s.handleListRedirects)
	mux.HandleFunc("POST /_admin/api/redirects", s.handleCreateRedirect)
	mux.HandleFunc("PUT /_admin/api/redirects/{id}", s.handleUpdateRedirect)
	mux.HandleFunc("DELETE /_admin/api/redirects/{id}", s.handleDeleteRedirect)

	// Static files
	mux.Handle("/_static/", http.StripPrefix("/_static/", http.FileServer(http.Dir(s.StaticDir))))

	// Everything else: redirect handler
	mux.HandleFunc("/", s.handleRedirect)

	slog.Info("starting redirector", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

// --- Redirect Logic ---

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	host := extractHost(r)
	path := r.URL.Path
	rawQuery := r.URL.RawQuery

	hostRules, err := s.Queries.GetRedirectsByHost(r.Context(), host)
	if err != nil {
		slog.Error("get redirects by host", "error", err)
	}
	catchAll, err := s.Queries.GetCatchAllRedirects(r.Context())
	if err != nil {
		slog.Error("get catch-all redirects", "error", err)
	}

	rules := append(hostRules, catchAll...)

	for _, rule := range rules {
		if dest, ok := matchRule(rule, path, rawQuery); ok {
			go func(id int64) {
				_ = dbgen.New(s.DB).IncrementHitCount(context.Background(), id)
			}(rule.ID)
			http.Redirect(w, r, dest, int(rule.RedirectType))
			return
		}
	}

	http.NotFound(w, r)
}

func matchRule(rule dbgen.Redirect, path, rawQuery string) (string, bool) {
	if rule.IsRegex != 0 {
		return matchRegex(rule, path, rawQuery)
	}
	return matchSimple(rule, path, rawQuery)
}

func matchRegex(rule dbgen.Redirect, path, rawQuery string) (string, bool) {
	re, err := regexp.Compile(rule.SourcePath)
	if err != nil {
		slog.Warn("invalid regex", "pattern", rule.SourcePath, "error", err)
		return "", false
	}

	// Try matching path+query first for full control
	fullPath := path
	if rawQuery != "" {
		fullPath = path + "?" + rawQuery
	}

	matches := re.FindStringSubmatch(fullPath)
	if matches == nil {
		// Try path only, append query to result
		matches = re.FindStringSubmatch(path)
		if matches == nil {
			return "", false
		}
		dest := rule.Destination
		for i := 1; i < len(matches); i++ {
			dest = strings.ReplaceAll(dest, "$"+strconv.Itoa(i), matches[i])
		}
		dest = appendQuery(dest, rawQuery)
		return dest, true
	}

	// Matched full path+query: user controls everything
	dest := rule.Destination
	for i := 1; i < len(matches); i++ {
		dest = strings.ReplaceAll(dest, "$"+strconv.Itoa(i), matches[i])
	}
	return dest, true
}

func matchSimple(rule dbgen.Redirect, path, rawQuery string) (string, bool) {
	srcPath := rule.SourcePath

	// Wildcard: /p/* matches /p/anything
	if strings.HasSuffix(srcPath, "/*") {
		prefix := strings.TrimSuffix(srcPath, "*")
		if strings.HasPrefix(path, prefix) || path+"/" == prefix {
			rest := strings.TrimPrefix(path, prefix)
			dest := rule.Destination + rest
			dest = appendQuery(dest, rawQuery)
			return dest, true
		}
		return "", false
	}

	// Exact match
	if path == srcPath {
		dest := rule.Destination
		dest = appendQuery(dest, rawQuery)
		return dest, true
	}

	return "", false
}

func appendQuery(dest, rawQuery string) string {
	if rawQuery == "" {
		return dest
	}
	parsed, err := url.Parse(dest)
	if err != nil {
		return dest
	}
	if parsed.RawQuery != "" {
		return dest // destination already has query, don't append
	}
	return dest + "?" + rawQuery
}

func extractHost(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	h, _, err := net.SplitHostPort(host)
	if err == nil {
		return strings.ToLower(h)
	}
	return strings.ToLower(strings.TrimSpace(host))
}

// --- Admin UI ---

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(filepath.Join(s.TemplatesDir, "admin.html"))
	if err != nil {
		http.Error(w, "admin page not found", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// --- Admin API ---

func (s *Server) handleListRedirects(w http.ResponseWriter, r *http.Request) {
	redirects, err := s.Queries.ListRedirects(r.Context())
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, redirects, 200)
}

func (s *Server) handleCreateRedirect(w http.ResponseWriter, r *http.Request) {
	var p dbgen.CreateRedirectParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, "invalid json: "+err.Error(), 400)
		return
	}
	if p.Destination == "" {
		jsonError(w, "destination is required", 400)
		return
	}
	if p.RedirectType == 0 {
		p.RedirectType = 301
	}
	if p.SourcePath == "" {
		p.SourcePath = "/"
	}
	if p.IsRegex != 0 {
		if _, err := regexp.Compile(p.SourcePath); err != nil {
			jsonError(w, "invalid regex: "+err.Error(), 400)
			return
		}
	}
	result, err := s.Queries.CreateRedirect(r.Context(), p)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, result, 201)
}

func (s *Server) handleUpdateRedirect(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", 400)
		return
	}
	var p struct {
		SourceHost   string `json:"source_host"`
		SourcePath   string `json:"source_path"`
		Destination  string `json:"destination"`
		RedirectType int64  `json:"redirect_type"`
		IsRegex      int64  `json:"is_regex"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, "invalid json: "+err.Error(), 400)
		return
	}
	if p.IsRegex != 0 {
		if _, err := regexp.Compile(p.SourcePath); err != nil {
			jsonError(w, "invalid regex: "+err.Error(), 400)
			return
		}
	}
	result, err := s.Queries.UpdateRedirect(r.Context(), dbgen.UpdateRedirectParams{
		SourceHost:   p.SourceHost,
		SourcePath:   p.SourcePath,
		Destination:  p.Destination,
		RedirectType: p.RedirectType,
		IsRegex:      p.IsRegex,
		ID:           id,
	})
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, result, 200)
}

func (s *Server) handleDeleteRedirect(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", 400)
		return
	}
	if err := s.Queries.DeleteRedirect(r.Context(), id); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "deleted"}, 200)
}

func jsonResp(w http.ResponseWriter, data any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
