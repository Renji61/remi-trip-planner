package httpapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"remi-trip-planner/internal/trips"
)

func (a *app) profilePage(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	u, err := a.tripService.GetUserByID(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	settings, _ := a.tripService.MergedSettingsForUI(r.Context(), uid)
	verified := !u.EmailVerifiedAt.IsZero()
	data := map[string]any{
		"User": u, "Settings": settings, "CSRFToken": CSRFToken(r.Context()),
		"EmailVerified": verified, "Saved": r.URL.Query().Get("saved") == "1",
	}
	if err := a.mergeDashboardShell(r.Context(), uid, "profile", "", data); err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "profile.html", data)
}

func (a *app) profileExport(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	exp, err := a.tripService.BuildAccountExport(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fname := fmt.Sprintf("remi-export-%s.json", time.Now().UTC().Format("20060102"))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(exp); err != nil {
		return
	}
}

func (a *app) profileSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	email := r.FormValue("email")
	username := r.FormValue("username")
	displayName := r.FormValue("display_name")
	u, err := a.tripService.UpdateUserProfile(r.Context(), uid, email, username, displayName)
	if err != nil {
		settings, _ := a.tripService.MergedSettingsForUI(r.Context(), uid)
		data := map[string]any{
			"User": u, "Settings": settings, "CSRFToken": CSRFToken(r.Context()),
			"Error": err.Error(), "EmailVerified": !u.EmailVerifiedAt.IsZero(),
		}
		if err2 := a.mergeDashboardShell(r.Context(), uid, "profile", "", data); err2 != nil {
			writeInternalServerError(w, r, err2)
			return
		}
		_ = a.templates.ExecuteTemplate(w, "profile.html", data)
		return
	}
	if u.EmailVerifiedAt.IsZero() && strings.TrimSpace(email) != "" {
		_, _ = a.tripService.IssueEmailVerificationToken(r.Context(), uid)
	}
	http.Redirect(w, r, "/profile?saved=1", http.StatusSeeOther)
}

func (a *app) profilePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	cur := r.FormValue("current_password")
	nw := r.FormValue("new_password")
	confirm := r.FormValue("new_password_confirm")
	if nw != confirm {
		u, _ := a.tripService.GetUserByID(r.Context(), uid)
		settings, _ := a.tripService.MergedSettingsForUI(r.Context(), uid)
		data := map[string]any{
			"User": u, "Settings": settings, "CSRFToken": CSRFToken(r.Context()),
			"PasswordError": "Passwords do not match.", "EmailVerified": !u.EmailVerifiedAt.IsZero(),
		}
		if err := a.mergeDashboardShell(r.Context(), uid, "profile", "", data); err != nil {
			writeInternalServerError(w, r, err)
			return
		}
		_ = a.templates.ExecuteTemplate(w, "profile.html", data)
		return
	}
	if err := a.tripService.UpdateUserPassword(r.Context(), uid, cur, nw); err != nil {
		u, _ := a.tripService.GetUserByID(r.Context(), uid)
		settings, _ := a.tripService.MergedSettingsForUI(r.Context(), uid)
		data := map[string]any{
			"User": u, "Settings": settings, "CSRFToken": CSRFToken(r.Context()),
			"EmailVerified": !u.EmailVerifiedAt.IsZero(),
		}
		if errors.Is(err, trips.ErrWrongCurrentPassword) {
			data["CurrentPasswordError"] = trips.ErrWrongCurrentPassword.Error()
		} else {
			data["PasswordError"] = err.Error()
		}
		if err2 := a.mergeDashboardShell(r.Context(), uid, "profile", "", data); err2 != nil {
			writeInternalServerError(w, r, err2)
			return
		}
		_ = a.templates.ExecuteTemplate(w, "profile.html", data)
		return
	}
	ip := clientIPString(r.RemoteAddr)
	rid := RequestIDFromContext(r.Context())
	slog.InfoContext(r.Context(), "password_changed",
		slog.String("user_id", uid),
		slog.String("client_ip", ip),
		slog.String("request_id", rid),
		slog.String("user_agent", TruncatedUserAgent(r)),
	)
	http.Redirect(w, r, "/profile?saved=1", http.StatusSeeOther)
}

func (a *app) profileResendVerify(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	_, _ = a.tripService.IssueEmailVerificationToken(r.Context(), uid)
	http.Redirect(w, r, "/profile?saved=1", http.StatusSeeOther)
}

// tripCollaborationEnabled is for handlers already gated to the trip owner: returns false and writes 403 when the trip has collaboration UI disabled.
func (a *app) tripCollaborationEnabled(w http.ResponseWriter, r *http.Request, tripID string) bool {
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		http.Error(w, "trip not found", http.StatusNotFound)
		return false
	}
	if !trip.UICollaborationEnabled {
		http.Error(w, "collaboration is disabled for this trip", http.StatusForbidden)
		return false
	}
	return true
}

func (a *app) tripInviteCollaborator(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if !a.requireTripOwner(w, r, tripID) {
		return
	}
	if !a.tripCollaborationEnabled(w, r, tripID) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	uid := CurrentUserID(r.Context())
	addedExisting, err := a.tripService.InviteCollaboratorByEmail(r.Context(), tripID, uid, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	q := url.Values{}
	q.Set("invite_email", email)
	if addedExisting {
		q.Set("invite_notice", "added")
	} else {
		q.Set("invite_notice", "sent")
	}
	http.Redirect(w, r, "/trips/"+tripID+"?"+q.Encode(), http.StatusSeeOther)
}

func (a *app) tripCreateInviteLink(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if !a.requireTripOwner(w, r, tripID) {
		return
	}
	if !a.tripCollaborationEnabled(w, r, tripID) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if r.FormValue("csrf_token") != CSRFToken(r.Context()) {
		http.Error(w, "invalid csrf", http.StatusForbidden)
		return
	}
	uid := CurrentUserID(r.Context())
	raw, err := a.tripService.CreateTripInviteLink(r.Context(), tripID, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	full := absoluteOrigin(r) + "/invites/accept?token=" + url.QueryEscape(raw)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": full})
}

func (a *app) tripRemoveMember(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if !a.requireTripOwner(w, r, tripID) {
		return
	}
	if !a.tripCollaborationEnabled(w, r, tripID) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	target := strings.TrimSpace(r.FormValue("user_id"))
	if target == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	if err := a.tripService.OwnerRemoveTripMember(r.Context(), tripID, uid, target); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) tripRevokeInvite(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if !a.requireTripOwner(w, r, tripID) {
		return
	}
	if !a.tripCollaborationEnabled(w, r, tripID) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	inviteID := strings.TrimSpace(r.FormValue("invite_id"))
	if inviteID == "" {
		http.Error(w, "invite_id required", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	if err := a.tripService.OwnerRevokeTripInvite(r.Context(), tripID, uid, inviteID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) tripLeaveCollaboration(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	uid := CurrentUserID(r.Context())
	if _, ok := a.requireTripAccess(w, r, tripID); !ok {
		return
	}
	if err := a.tripService.LeaveTrip(r.Context(), tripID, uid); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) tripStopSharing(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if !a.requireTripOwner(w, r, tripID) {
		return
	}
	uid := CurrentUserID(r.Context())
	if err := a.tripService.StopSharingTrip(r.Context(), tripID, uid); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"?sharing=stopped", http.StatusSeeOther)
}

func (a *app) tripHideArchived(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	uid := CurrentUserID(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	hidden := r.FormValue("hidden") == "1" || r.FormValue("hidden") == "true"
	if err := a.tripService.SetArchivedTripHidden(r.Context(), tripID, uid, uid, hidden); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}
