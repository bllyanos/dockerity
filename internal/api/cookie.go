package api

import (
	"net/http"
	"strings"
	"time"
)

// SessionCookie is the name of the browser session cookie.
const SessionCookie = "dockerity_session"

func setSessionCookie(w http.ResponseWriter, r *http.Request, raw string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    raw,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	// Drop a refresh queued earlier in this response so only the deletion is sent.
	h := w.Header()
	kept := h["Set-Cookie"][:0]
	for _, v := range h["Set-Cookie"] {
		if !strings.HasPrefix(v, SessionCookie+"=") {
			kept = append(kept, v)
		}
	}
	h["Set-Cookie"] = kept
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

// isHTTPS reports whether the client reached us over TLS, directly or via Traefik.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
