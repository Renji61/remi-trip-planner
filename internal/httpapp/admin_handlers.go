package httpapp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"remi-trip-planner/internal/trips"
)

type adminUserView struct {
	User            trips.User
	IsSelf          bool
	LastAdminLocked bool // cannot demote — only administrator on the instance
}

func (a *app) adminUsersPage(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	all, err := a.tripService.ListUsersForManagement(r.Context(), uid)
	if err != nil {
		if errors.Is(err, trips.ErrAdminRequired) {
			http.Error(w, "Administrator access is required.", http.StatusForbidden)
			return
		}
		writeInternalServerError(w, r, err)
		return
	}

	adminCount := 0
	verifiedCount := 0
	for _, u := range all {
		if u.IsAdmin {
			adminCount++
		}
		if !u.EmailVerifiedAt.IsZero() {
			verifiedCount++
		}
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	roleFilter := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("role")))
	if roleFilter != "admin" && roleFilter != "user" {
		roleFilter = ""
	}

	filtered := make([]trips.User, 0, len(all))
	for _, u := range all {
		if roleFilter == "admin" && !u.IsAdmin {
			continue
		}
		if roleFilter == "user" && u.IsAdmin {
			continue
		}
		if q != "" {
			hay := strings.ToLower(u.Email + " " + u.Username + " " + u.DisplayName)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		filtered = append(filtered, u)
	}

	rows := make([]adminUserView, 0, len(filtered))
	for _, u := range filtered {
		rows = append(rows, adminUserView{
			User:            u,
			IsSelf:          u.ID == uid,
			LastAdminLocked: u.IsAdmin && adminCount <= 1,
		})
	}

	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}

	errMsg := ""
	switch r.URL.Query().Get("error") {
	case "last_admin":
		errMsg = "You cannot remove the last administrator."
	case "generic":
		errMsg = "Could not update that account. Try again."
	}

	data := map[string]any{
		"Settings":      settings,
		"CSRFToken":     CSRFToken(r.Context()),
		"CurrentUser":   CurrentUser(r.Context()),
		"AdminUserRows": rows,
		"TotalUsers":    len(all),
		"AdminCount":    adminCount,
		"VerifiedCount": verifiedCount,
		"SearchQuery":   strings.TrimSpace(r.URL.Query().Get("q")),
		"RoleFilter":    roleFilter,
		"Saved":         r.URL.Query().Get("saved") == "1",
		"Error":         errMsg,
	}
	if err := a.mergeDashboardShell(r.Context(), uid, "admin-users", "", data); err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "admin_users.html", data)
}

func (a *app) adminUserSetRole(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "userID"))
	uid := CurrentUserID(r.Context())
	makeAdmin := r.FormValue("make_admin") == "1"
	if targetID == "" {
		http.Redirect(w, r, "/admin/users?error=generic", http.StatusSeeOther)
		return
	}
	err := a.tripService.SetUserAdministrator(r.Context(), uid, targetID, makeAdmin)
	if err != nil {
		if errors.Is(err, trips.ErrAdminRequired) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if strings.Contains(err.Error(), "last administrator") {
			http.Redirect(w, r, "/admin/users?error=last_admin", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/users?error=generic", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/users?saved=1", http.StatusSeeOther)
}
