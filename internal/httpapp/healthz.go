package httpapp

import (
	"context"
	"net/http"
	"time"
)

func (a *app) healthz(w http.ResponseWriter, r *http.Request) {
	if a.env.HealthzCheckDB && a.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := a.db.PingContext(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}
