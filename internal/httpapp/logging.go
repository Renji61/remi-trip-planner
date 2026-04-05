package httpapp

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type remiCtxKey string

const ctxKeyRequestID remiCtxKey = "remi_request_id"

// RequestIDFromContext returns the request correlation id, if set.
func RequestIDFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ctxKeyRequestID).(string)
	return s
}

func remiRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, id)))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.code == 0 {
		s.code = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.code == 0 {
		s.code = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func remiAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.code
		if status == 0 {
			status = http.StatusOK
		}
		rid := RequestIDFromContext(r.Context())
		uid := CurrentUserID(r.Context())
		ip := clientIPString(r.RemoteAddr)
		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", rid),
			slog.String("client_ip", ip),
		}
		if uid != "" {
			attrs = append(attrs, slog.String("user_id", uid))
		}
		slog.LogAttrs(r.Context(), slog.LevelInfo, "http_request", attrs...)
	})
}
