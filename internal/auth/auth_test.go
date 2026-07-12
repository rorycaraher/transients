package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPasswordCheck(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword(hash, "correct-horse") {
		t.Fatal("expected correct password to verify")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestSessionCookieRoundTrip(t *testing.T) {
	secret := "test-secret"

	w := httptest.NewRecorder()
	SetSessionCookie(w, secret)

	req := httptest.NewRequest("GET", "/admin", nil)
	for _, c := range w.Result().Cookies() {
		req.AddCookie(c)
	}

	if !IsAuthenticated(req, secret) {
		t.Fatal("expected freshly set session cookie to authenticate")
	}
}

func TestSessionCookieWrongSecret(t *testing.T) {
	w := httptest.NewRecorder()
	SetSessionCookie(w, "secret-a")

	req := httptest.NewRequest("GET", "/admin", nil)
	for _, c := range w.Result().Cookies() {
		req.AddCookie(c)
	}

	if IsAuthenticated(req, "secret-b") {
		t.Fatal("expected cookie signed with a different secret to fail")
	}
}

func TestSessionCookieExpired(t *testing.T) {
	secret := "test-secret"
	expired := sign(secret, time.Now().Add(-1*time.Minute))

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: expired})

	if IsAuthenticated(req, secret) {
		t.Fatal("expected expired session cookie to fail")
	}
}
