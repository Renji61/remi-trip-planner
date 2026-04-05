package httpapp

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"remi-trip-planner/internal/trips"
)

func loginURLWithNext(next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return "/login"
	}
	return "/login?next=" + url.QueryEscape(next)
}

// redirectAfterAuthWithNext sends the user to `next` when it is a safe same-origin path.
// If `next` is an invite accept URL with a valid token, accepts the invite and redirects to that trip.
func (a *app) redirectAfterAuthWithNext(w http.ResponseWriter, r *http.Request, userID, next, fallback string) {
	next = strings.TrimSpace(next)
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		http.Redirect(w, r, fallback, http.StatusSeeOther)
		return
	}
	pu, err := url.Parse(next)
	if err != nil || pu.Path != "/invites/accept" {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	tok := strings.TrimSpace(pu.Query().Get("token"))
	if tok == "" {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	inv, err := a.tripService.PreviewTripInvite(r.Context(), tok)
	if err != nil {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	if err := a.tripService.AcceptTripInvite(r.Context(), userID, tok); err != nil {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+inv.TripID, http.StatusSeeOther)
}

func (a *app) setupPage(w http.ResponseWriter, r *http.Request) {
	n, err := a.tripService.CountUsers(r.Context())
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "setup.html", map[string]any{
		"Settings": trips.AppSettings{ThemePreference: "system", AppTitle: "REMI Trip Planner"},
	})
}

func (a *app) setupSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	username := r.FormValue("username")
	displayName := r.FormValue("display_name")
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")
	if password != confirm {
		_ = a.templates.ExecuteTemplate(w, "setup.html", map[string]any{
			"Settings":    trips.AppSettings{ThemePreference: "system", AppTitle: "REMI Trip Planner"},
			"Error":       "Passwords do not match.",
			"Email":       email,
			"Username":    username,
			"DisplayName": displayName,
		})
		return
	}
	u, tok, _, err := a.tripService.RegisterFirstUser(r.Context(), email, username, displayName, password)
	if err != nil {
		_ = a.templates.ExecuteTemplate(w, "setup.html", map[string]any{
			"Settings":    trips.AppSettings{ThemePreference: "system", AppTitle: "REMI Trip Planner"},
			"Error":       err.Error(),
			"Email":       email,
			"Username":    username,
			"DisplayName": displayName,
		})
		return
	}
	a.writeSessionCookie(w, tok)
	http.Redirect(w, r, "/?saved=1", http.StatusSeeOther)
	_ = u
}

func (a *app) loginPage(w http.ResponseWriter, r *http.Request) {
	n, err := a.tripService.CountUsers(r.Context())
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	if n == 0 {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	if CurrentUserID(r.Context()) != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	app, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	next := strings.TrimSpace(r.URL.Query().Get("next"))
	registerURL := ""
	if app.RegistrationEnabled {
		if next != "" {
			registerURL = "/register?next=" + url.QueryEscape(next)
		} else {
			registerURL = "/register"
		}
	}
	_ = a.templates.ExecuteTemplate(w, "login.html", map[string]any{
		"Settings":            app,
		"RegistrationEnabled": app.RegistrationEnabled,
		"RegisterURL":         registerURL,
		"Next":                next,
		"Error":               r.URL.Query().Get("err"),
	})
}

func (a *app) registerPage(w http.ResponseWriter, r *http.Request) {
	n, err := a.tripService.CountUsers(r.Context())
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	if n == 0 {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	if CurrentUserID(r.Context()) != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	app, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	if !app.RegistrationEnabled {
		http.Error(w, "Registration is disabled on this server.", http.StatusForbidden)
		return
	}
	next := strings.TrimSpace(r.URL.Query().Get("next"))
	_ = a.templates.ExecuteTemplate(w, "register.html", map[string]any{
		"Settings": app,
		"Next":     next,
		"LoginURL": loginURLWithNext(next),
	})
}

func (a *app) registerSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	app, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	if !app.RegistrationEnabled {
		http.Error(w, "Registration is disabled.", http.StatusForbidden)
		return
	}
	email := r.FormValue("email")
	username := r.FormValue("username")
	displayName := r.FormValue("display_name")
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")
	next := strings.TrimSpace(r.FormValue("next"))
	if password != confirm {
		_ = a.templates.ExecuteTemplate(w, "register.html", map[string]any{
			"Settings":    app,
			"Next":        next,
			"LoginURL":    loginURLWithNext(next),
			"Error":       "Passwords do not match.",
			"Email":       email,
			"Username":    username,
			"DisplayName": displayName,
		})
		return
	}
	u, tok, _, err := a.tripService.RegisterUser(r.Context(), email, username, displayName, password)
	if err != nil {
		msg := registerUserFacingError(err)
		_ = a.templates.ExecuteTemplate(w, "register.html", map[string]any{
			"Settings":    app,
			"Next":        next,
			"LoginURL":    loginURLWithNext(next),
			"Error":       msg,
			"Email":       email,
			"Username":    username,
			"DisplayName": displayName,
		})
		return
	}
	a.writeSessionCookie(w, tok)
	if next != "" {
		a.redirectAfterAuthWithNext(w, r, u.ID, next, "/?registered=1")
		return
	}
	_ = u
	http.Redirect(w, r, "/?registered=1", http.StatusSeeOther)
}

func (a *app) loginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	id := r.FormValue("identifier")
	pw := r.FormValue("password")
	next := strings.TrimSpace(r.FormValue("next"))
	u, tok, _, err := a.tripService.LoginWithIdentifier(r.Context(), id, pw)
	if err != nil {
		ip := clientIPString(r.RemoteAddr)
		rid := RequestIDFromContext(r.Context())
		slog.WarnContext(r.Context(), "login_failed",
			slog.String("client_ip", ip),
			slog.String("request_id", rid),
			slog.String("user_agent", TruncatedUserAgent(r)),
		)
		q := "err=1"
		if next != "" {
			q += "&next=" + url.QueryEscape(next)
		}
		http.Redirect(w, r, "/login?"+q, http.StatusSeeOther)
		return
	}
	a.writeSessionCookie(w, tok)
	ip := clientIPString(r.RemoteAddr)
	rid := RequestIDFromContext(r.Context())
	slog.InfoContext(r.Context(), "login_success",
		slog.String("user_id", u.ID),
		slog.String("client_ip", ip),
		slog.String("request_id", rid),
		slog.String("user_agent", TruncatedUserAgent(r)),
	)
	if next != "" {
		a.redirectAfterAuthWithNext(w, r, u.ID, next, "/")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	ok, err := requestHasValidCSRF(r)
	if err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, "invalid security token", http.StatusForbidden)
		return
	}
	uid := CurrentUserID(r.Context())
	ip := clientIPString(r.RemoteAddr)
	rid := RequestIDFromContext(r.Context())
	slog.InfoContext(r.Context(), "logout",
		slog.String("user_id", uid),
		slog.String("client_ip", ip),
		slog.String("request_id", rid),
		slog.String("user_agent", TruncatedUserAgent(r)),
	)
	tok := SessionTokenRaw(r.Context())
	_ = a.tripService.Logout(r.Context(), tok)
	http.SetCookie(w, a.clearSessionCookie())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *app) verifyEmailPage(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimSpace(r.URL.Query().Get("token"))
	if tok == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	if err := a.tripService.VerifyEmailToken(r.Context(), tok); err != nil {
		_ = a.templates.ExecuteTemplate(w, "verify_email.html", map[string]any{
			"Settings": trips.AppSettings{ThemePreference: "system", AppTitle: "REMI Trip Planner"},
			"OK":       false,
		})
		return
	}
	_ = a.templates.ExecuteTemplate(w, "verify_email.html", map[string]any{
		"Settings": trips.AppSettings{ThemePreference: "system", AppTitle: "REMI Trip Planner"},
		"OK":       true,
	})
}

func (a *app) inviteAcceptPage(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimSpace(r.URL.Query().Get("token"))
	if tok == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
		return
	}
	inv, err := a.tripService.PreviewTripInvite(r.Context(), tok)
	if err != nil {
		http.Error(w, "invalid or expired invite", http.StatusBadRequest)
		return
	}
	settings, _ := a.tripService.MergedSettingsForUI(r.Context(), CurrentUserID(r.Context()))
	_ = a.templates.ExecuteTemplate(w, "invite_accept.html", map[string]any{
		"Settings":     settings,
		"CSRFToken":    CSRFToken(r.Context()),
		"Token":        tok,
		"TripID":       inv.TripID,
		"Email":        inv.EmailNormalized,
		"IsLinkInvite": inv.IsLinkInvite,
	})
}

func (a *app) inviteAcceptSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	tok := strings.TrimSpace(r.FormValue("token"))
	uid := CurrentUserID(r.Context())
	if tok == "" || uid == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	inv, err := a.tripService.PreviewTripInvite(r.Context(), tok)
	if err != nil {
		http.Error(w, "invalid or expired invite", http.StatusBadRequest)
		return
	}
	tripID := inv.TripID
	if err := a.tripService.AcceptTripInvite(r.Context(), uid, tok); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

// registerUserFacingError maps errors to messages that avoid leaking whether an email/username exists.
func registerUserFacingError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "password must be at least"),
		strings.Contains(msg, "username must be at least"),
		strings.Contains(msg, "username may only contain"),
		strings.Contains(msg, "email is required"),
		strings.Contains(msg, "registration is disabled"),
		strings.Contains(msg, "complete initial setup"):
		return msg
	default:
		return "Unable to register with these details. Try signing in, or use a different email or username."
	}
}
