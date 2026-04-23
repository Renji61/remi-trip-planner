package httpapp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (a *app) apiStopWeather(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	tripID := strings.TrimSpace(chi.URLParam(r, "tripID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	dateISO := strings.TrimSpace(r.URL.Query().Get("date"))
	if tripID == "" || itemID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	preview, err := a.tripService.GetStopWeatherPreview(r.Context(), tripID, itemID, dateISO)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = json.NewEncoder(w).Encode(preview)
}
