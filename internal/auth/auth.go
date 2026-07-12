// Package auth implements the single-admin login: a bcrypt password check
// and an HMAC-signed session cookie. There is no user table — this app has
// exactly one operator.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	CookieName = "session"
	SessionTTL = 30 * 24 * time.Hour
)

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// HashPassword is a helper for generating ADMIN_PASSWORD_HASH; not used at
// request time.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// sign produces "<expiryUnix>.<base64-hmac>" for the given secret.
func sign(secret string, expiry time.Time) string {
	payload := strconv.FormatInt(expiry.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig
}

func verify(secret, token string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	payload, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return false
	}

	expiryUnix, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < expiryUnix
}

func SetSessionCookie(w http.ResponseWriter, secret string) {
	expiry := time.Now().Add(SessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sign(secret, expiry),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiry,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func IsAuthenticated(r *http.Request, secret string) bool {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return false
	}
	return verify(secret, c.Value)
}

// RequireAuth redirects unauthenticated requests to /login.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsAuthenticated(r, secret) {
				http.Redirect(w, r, fmt.Sprintf("/login?next=%s", r.URL.Path), http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
