package httpapp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
)

// newPublicErrorID returns an 8-character hex id safe to show to end users.
func newPublicErrorID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func wantsJSONResponse(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "application/json")
}

// writeInternalServerError logs a server-side error with a public correlation id and responds 500.
// Use this for unexpected handler failures so clients get a stable error_id (JSON/HTML per wantsJSONResponse).
func writeInternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		err = fmt.Errorf("unknown error")
	}
	id := newPublicErrorID()
	uid := CurrentUserID(r.Context())
	rid := RequestIDFromContext(r.Context())
	ip := clientIPString(r.RemoteAddr)
	slog.ErrorContext(r.Context(), "internal_server_error",
		slog.String("error_id", id),
		slog.String("request_id", rid),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("client_ip", ip),
		slog.String("user_id", uid),
		slog.String("err", err.Error()),
	)
	writePublicError500(w, r, id)
}

func writePublicError500(w http.ResponseWriter, r *http.Request, errorID string) {
	if w == nil || r == nil {
		return
	}
	if wantsJSONResponse(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":     "internal_error",
			"error_id":  errorID,
			"message":   "We couldn't finish that just now. Your last change may not have reached the server yet, so please refresh and try again.",
			"retryable": true,
		})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Server error</title></head><body><h1>We couldn't finish that just now</h1><p>Your information is still safe. Refresh the page and try again in a moment.</p><p>If the problem keeps happening, include this reference when you contact support:</p><p><code>%s</code></p></body></html>`, errorID)
}

// remiRecoverer recovers panics, logs stack + public error id, and returns 500.
func remiRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}
				err := fmt.Errorf("%v", rvr)
				stack := string(debug.Stack())
				id := newPublicErrorID()
				uid := CurrentUserID(r.Context())
				rid := RequestIDFromContext(r.Context())
				ip := clientIPString(r.RemoteAddr)
				slog.ErrorContext(r.Context(), "panic_recovered",
					slog.String("error_id", id),
					slog.String("request_id", rid),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("client_ip", ip),
					slog.String("user_id", uid),
					slog.String("err", err.Error()),
					slog.String("stack", stack),
				)
				if r.Header.Get("Connection") != "Upgrade" {
					writePublicError500(w, r, id)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
