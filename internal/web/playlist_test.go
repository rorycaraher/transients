package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rorycaraher/transients/internal/store"
)

func mustReadyTrack(t *testing.T, srv *Server, slug, title string) {
	t.Helper()
	if err := srv.store.CreateTrackFromDiscovery(slug, slug+".mp3", title); err != nil {
		t.Fatalf("CreateTrackFromDiscovery: %v", err)
	}
	if err := srv.store.MarkTrackReady(slug, "audio/mpeg", 1234); err != nil {
		t.Fatalf("MarkTrackReady: %v", err)
	}
}

func mustPlaylist(t *testing.T, srv *Server, slug string, expiresAt *time.Time, trackSlugs ...string) {
	t.Helper()
	if err := srv.store.CreatePlaylist(slug, "Playlist "+slug, "playlist description", expiresAt); err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	for _, ts := range trackSlugs {
		if err := srv.store.AddTrackToPlaylist(slug, ts); err != nil {
			t.Fatalf("AddTrackToPlaylist: %v", err)
		}
	}
}

func entryURL(t *testing.T, srv *Server, playlistSlug, trackSlug, action string) string {
	t.Helper()
	entries, err := srv.store.ListPlaylistEntries(playlistSlug)
	if err != nil {
		t.Fatalf("ListPlaylistEntries: %v", err)
	}
	for _, e := range entries {
		if e.Track.Slug == trackSlug {
			return "/p/" + playlistSlug + "/entries/" + strconv.FormatInt(e.ID, 10) + "/" + action
		}
	}
	t.Fatalf("track %q not in playlist %q", trackSlug, playlistSlug)
	return ""
}

func do(t *testing.T, mux http.Handler, method, target string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestPlaylistShareShowsOnlyAvailableTracksAndNoTrackSlugs(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()

	mustReadyTrack(t, srv, "slug-ready-one", "Ready One")
	mustReadyTrack(t, srv, "slug-expired", "Expired Track")
	mustReadyTrack(t, srv, "slug-ready-two", "Ready Two")
	if err := srv.store.CreatePendingTrack("slug-pending", "slug-pending.mp3", "Pending Track", nil, false, ""); err != nil {
		t.Fatalf("CreatePendingTrack: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	if err := srv.store.UpdateTrack("slug-expired", "Expired Track", &past, false, ""); err != nil {
		t.Fatalf("UpdateTrack: %v", err)
	}
	if err := srv.store.UpdateTrack("slug-ready-two", "Ready Two", nil, true, "two's description"); err != nil {
		t.Fatalf("UpdateTrack: %v", err)
	}
	mustPlaylist(t, srv, "pl", nil, "slug-ready-one", "slug-expired", "slug-pending", "slug-ready-two")

	w := do(t, mux, http.MethodGet, "/p/pl", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()

	for _, want := range []string{"Playlist pl", "playlist description", "Ready One", "Ready Two", "two&#39;s description", "/download"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
	for _, unwanted := range []string{"Expired Track", "Pending Track", "slug-", "/t/"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("expected body not to contain %q", unwanted)
		}
	}
	if strings.Index(body, "Ready One") > strings.Index(body, "Ready Two") {
		t.Errorf("expected tracks in playlist order")
	}
}

func TestPlaylistShareExpiredAndEmpty(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()

	past := time.Now().Add(-time.Hour)
	mustPlaylist(t, srv, "old", &past)
	w := do(t, mux, http.MethodGet, "/p/old", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "This link has expired") {
		t.Fatalf("expected expired page, got %d: %s", w.Code, w.Body.String())
	}

	mustPlaylist(t, srv, "empty", nil)
	w = do(t, mux, http.MethodGet, "/p/empty", nil)
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, "No tracks are available") || !strings.Contains(body, "Playlist empty") {
		t.Fatalf("expected empty tracklist page, got %d: %s", w.Code, body)
	}
	if strings.Contains(body, "player.js") {
		t.Fatalf("expected no player on an empty playlist")
	}

	if w := do(t, mux, http.MethodGet, "/p/nope", nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected unknown playlist to 404, got %d", w.Code)
	}
}

func TestPlaylistEntryRoutes(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()

	mustReadyTrack(t, srv, "a", "A")
	mustReadyTrack(t, srv, "b", "B")
	if err := srv.store.UpdateTrack("b", "B", nil, true, ""); err != nil {
		t.Fatalf("UpdateTrack: %v", err)
	}
	mustPlaylist(t, srv, "pl", nil, "a", "b")
	mustPlaylist(t, srv, "other", nil)

	w := do(t, mux, http.MethodGet, entryURL(t, srv, "pl", "a", "audio"), nil)
	if w.Code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "a.mp3") {
		t.Fatalf("expected 302 to presigned audio, got %d %q", w.Code, w.Header().Get("Location"))
	}

	if w := do(t, mux, http.MethodGet, entryURL(t, srv, "pl", "a", "download"), nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected download of non-downloadable track to 404, got %d", w.Code)
	}
	w = do(t, mux, http.MethodGet, entryURL(t, srv, "pl", "b", "download"), nil)
	if w.Code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "b.mp3") {
		t.Fatalf("expected 302 to presigned download, got %d", w.Code)
	}

	foreign := strings.Replace(entryURL(t, srv, "pl", "a", "audio"), "/p/pl/", "/p/other/", 1)
	if w := do(t, mux, http.MethodGet, foreign, nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected entry via another playlist to 404, got %d", w.Code)
	}

	for i := 0; i < 2; i++ {
		if w := do(t, mux, http.MethodPost, entryURL(t, srv, "pl", "a", "play"), nil); w.Code != http.StatusNoContent {
			t.Fatalf("expected 204 from play, got %d", w.Code)
		}
	}
	track, err := srv.store.GetTrack("a")
	if err != nil {
		t.Fatalf("GetTrack: %v", err)
	}
	if track.PlayCount != 2 {
		t.Fatalf("expected Track play count 2, got %d", track.PlayCount)
	}

	past := time.Now().Add(-time.Hour)
	if err := srv.store.UpdateTrack("a", "A", &past, false, ""); err != nil {
		t.Fatalf("UpdateTrack: %v", err)
	}
	if w := do(t, mux, http.MethodGet, entryURL(t, srv, "pl", "a", "audio"), nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected expired track's audio to 404, got %d", w.Code)
	}

	bAudio := entryURL(t, srv, "pl", "b", "audio")
	if err := srv.store.UpdatePlaylist("pl", "Playlist pl", "", &past); err != nil {
		t.Fatalf("UpdatePlaylist: %v", err)
	}
	if w := do(t, mux, http.MethodGet, bAudio, nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected entry of expired playlist to 404, got %d", w.Code)
	}
}

func TestPlaylistAdminRequiresAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := srv.Mux()
	mustPlaylist(t, srv, "pl", nil)

	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/admin/playlists"},
		{http.MethodGet, "/admin/playlists/pl/edit"},
		{http.MethodPost, "/admin/playlists/pl/delete"},
	} {
		w := do(t, mux, tc.method, tc.target, nil)
		if w.Code != http.StatusSeeOther || !strings.HasPrefix(w.Header().Get("Location"), "/login") {
			t.Errorf("%s %s: expected redirect to login, got %d", tc.method, tc.target, w.Code)
		}
	}
	if _, err := srv.store.GetPlaylist("pl"); err != nil {
		t.Fatalf("expected playlist to survive unauthenticated delete, got %v", err)
	}
}

func TestPlaylistAdminLifecycle(t *testing.T) {
	srv, cfg := newTestServer(t)
	mux := srv.Mux()
	cookies := login(t, mux, cfg.SessionSecret)
	mustReadyTrack(t, srv, "a", "Track A")
	mustReadyTrack(t, srv, "b", "Track B")

	postForm := func(target string, form url.Values) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	w := postForm("/admin/playlists", url.Values{"title": {"Mix"}, "description": {"desc"}, "expires_at": {"2099-01-31"}})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("create: expected 303, got %d: %s", w.Code, w.Body.String())
	}
	editURL := w.Header().Get("Location")
	slug := strings.TrimSuffix(strings.TrimPrefix(editURL, "/admin/playlists/"), "/edit")

	for _, ts := range []string{"a", "b", "a"} {
		if w := postForm("/admin/playlists/"+slug+"/tracks", url.Values{"track_slug": {ts}}); w.Code != http.StatusSeeOther {
			t.Fatalf("add %s: expected 303, got %d", ts, w.Code)
		}
	}
	entries, _ := srv.store.ListPlaylistEntries(slug)
	if len(entries) != 2 {
		t.Fatalf("expected duplicate add to be ignored, got %d entries", len(entries))
	}

	moveURL := "/admin/playlists/" + slug + "/entries/" + strconv.FormatInt(entries[1].ID, 10) + "/move"
	if w := postForm(moveURL, url.Values{"dir": {"up"}}); w.Code != http.StatusSeeOther {
		t.Fatalf("move: expected 303, got %d", w.Code)
	}
	entries, _ = srv.store.ListPlaylistEntries(slug)
	if entries[0].Track.Slug != "b" {
		t.Fatalf("expected b first after move up, got %s", entries[0].Track.Slug)
	}

	orderReq := httptest.NewRequest(http.MethodPost, "/admin/playlists/"+slug+"/order",
		strings.NewReader(`{"entry_ids":[`+strconv.FormatInt(entries[1].ID, 10)+`,`+strconv.FormatInt(entries[0].ID, 10)+`]}`))
	for _, c := range cookies {
		orderReq.AddCookie(c)
	}
	orderW := httptest.NewRecorder()
	mux.ServeHTTP(orderW, orderReq)
	if orderW.Code != http.StatusNoContent {
		t.Fatalf("order: expected 204, got %d: %s", orderW.Code, orderW.Body.String())
	}

	editW := do(t, mux, http.MethodGet, editURL, cookies)
	editBody := editW.Body.String()
	if editW.Code != http.StatusOK || !strings.Contains(editBody, `value="2099-01-31"`) {
		t.Fatalf("expected edit page to round-trip the chosen expiry date, got %d", editW.Code)
	}
	if strings.Index(editBody, "Track A") > strings.Index(editBody, "Track B") {
		t.Fatalf("expected drag-drop order [a b] on edit page")
	}

	dash := do(t, mux, http.MethodGet, "/admin", cookies).Body.String()
	if !strings.Contains(dash, "1 playlist: Mix") {
		t.Fatalf("expected dashboard delete confirm to name the playlist")
	}

	w = postForm("/admin/playlists/"+slug+"/rotate", nil)
	newSlug := strings.TrimSuffix(strings.TrimPrefix(w.Header().Get("Location"), "/admin/playlists/"), "/edit")
	if newSlug == "" || newSlug == slug {
		t.Fatalf("expected rotate to redirect to a new slug, got %q", w.Header().Get("Location"))
	}
	if w := do(t, mux, http.MethodGet, "/p/"+slug, nil); w.Code != http.StatusNotFound {
		t.Fatalf("expected old playlist link to 404, got %d", w.Code)
	}
	if w := do(t, mux, http.MethodGet, "/p/"+newSlug, nil); w.Code != http.StatusOK {
		t.Fatalf("expected new playlist link to serve, got %d", w.Code)
	}

	if w := postForm("/admin/playlists/"+newSlug+"/delete", nil); w.Code != http.StatusSeeOther {
		t.Fatalf("delete: expected 303, got %d", w.Code)
	}
	if _, err := srv.store.GetPlaylist(newSlug); err != store.ErrNotFound {
		t.Fatalf("expected playlist gone, got %v", err)
	}
	if _, err := srv.store.GetTrack("a"); err != nil {
		t.Fatalf("expected tracks to survive playlist delete, got %v", err)
	}
}
