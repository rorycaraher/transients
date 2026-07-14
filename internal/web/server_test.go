package web

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rorycaraher/transients/internal/auth"
	"github.com/rorycaraher/transients/internal/config"
	"github.com/rorycaraher/transients/internal/db"
	"github.com/rorycaraher/transients/internal/r2"
	"github.com/rorycaraher/transients/internal/store"
)

func newTestServer(t *testing.T) (*Server, *config.Config) {
	t.Helper()

	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	hash, err := auth.HashPassword("s3cret")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	cfg := &config.Config{
		BaseURL:           "https://example.test",
		AdminPasswordHash: hash,
		SessionSecret:     "test-session-secret",
		PresignedGetTTL:   0, // presigning is local, doesn't need to be realistic here
	}

	// Fake credentials: presign operations are purely local crypto and never
	// hit the network, so this is safe to use without a real R2 account.
	r2c := r2.New("fake-account", "fake-key", "fake-secret", "fake-bucket")

	st := store.New(conn)
	srv, err := NewServer(cfg, st, r2c, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv, cfg
}

func login(t *testing.T, mux http.Handler, secret string) []*http.Cookie {
	t.Helper()
	form := url.Values{"password": {"s3cret"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("login failed: status %d, body %s", w.Code, w.Body.String())
	}
	return w.Result().Cookies()
}

func TestAdminRequiresAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect for unauthenticated /admin, got %d", w.Code)
	}
	loc := w.Result().Header.Get("Location")
	if !strings.HasPrefix(loc, "/login") {
		t.Fatalf("expected redirect to /login, got %q", loc)
	}
}

func TestLoginThenDashboard(t *testing.T) {
	srv, cfg := newTestServer(t)
	mux := srv.Mux()

	cookies := login(t, mux, cfg.SessionSecret)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated dashboard, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Nothing uploaded yet") {
		t.Fatalf("expected empty-state message in dashboard body")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()

	form := url.Values{"password": {"wrong"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering login form, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Incorrect password") {
		t.Fatalf("expected error message in response body")
	}
}

func TestUploadRequestAndStatusLifecycle(t *testing.T) {
	srv, cfg := newTestServer(t)
	mux := srv.Mux()
	cookies := login(t, mux, cfg.SessionSecret)

	reqBody := `{"title":"My Track","filename":"song.mp3","content_type":"audio/mpeg"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/upload/request", strings.NewReader(reqBody))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload request failed: %d %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, `"put_url"`) || !strings.Contains(body, `"slug"`) {
		t.Fatalf("expected slug and put_url in response, got %s", body)
	}

	// pull the slug out crudely (avoids importing encoding/json just for the test)
	slug := extractJSONString(t, body, "slug")

	statusReq := httptest.NewRequest(http.MethodGet, "/admin/upload/status/"+slug, nil)
	for _, c := range cookies {
		statusReq.AddCookie(c)
	}
	statusW := httptest.NewRecorder()
	mux.ServeHTTP(statusW, statusReq)

	if statusW.Code != http.StatusOK {
		t.Fatalf("status check failed: %d %s", statusW.Code, statusW.Body.String())
	}
	if !strings.Contains(statusW.Body.String(), `"pending"`) {
		t.Fatalf("expected pending status before ingest runs, got %s", statusW.Body.String())
	}
}

func TestPublicShareRouteNoAuthRequired(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()

	req := httptest.NewRequest(http.MethodGet, "/t/does-not-exist", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown slug, got %d", w.Code)
	}
}

func TestUnmatchedRouteRendersNotFoundPage(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()

	req := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched route, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Not found") {
		t.Fatalf("expected not-found page body, got %s", w.Body.String())
	}
}

func extractJSONString(t *testing.T, body, key string) string {
	t.Helper()
	marker := `"` + key + `":"`
	i := strings.Index(body, marker)
	if i == -1 {
		t.Fatalf("key %q not found in %s", key, body)
	}
	rest := body[i+len(marker):]
	end := strings.Index(rest, `"`)
	if end == -1 {
		t.Fatalf("malformed JSON value for key %q in %s", key, body)
	}
	return rest[:end]
}
