package httpapp

import (
	"context"
	"encoding/json"
	ht "html"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"remi-trip-planner/internal/changelog"
	"remi-trip-planner/internal/trips"
	"remi-trip-planner/internal/version"
)

// sidebarInProgressTrip is shown under My Trip on the dashboard shell (desktop + mobile bottom nav).
// In-progress trips are listed first; upcoming trips fill remaining slots up to max (see filterDashboardSidebarTrips).
type sidebarInProgressTrip struct {
	ID        string
	Name      string
	DateRange string
}

func (a *app) mergeDashboardShell(ctx context.Context, userID, navActive, sidebarTripID string, data map[string]any) error {
	tr, err := a.sidebarInProgressTrips(ctx, userID)
	if err != nil {
		return err
	}
	data["SidebarNavActive"] = navActive
	data["SidebarInProgressTrips"] = tr
	data["SidebarTripID"] = sidebarTripID
	data["TripID"] = sidebarTripID
	if strings.TrimSpace(userID) != "" {
		data["CurrentUser"] = CurrentUser(ctx)
		n, _ := a.tripService.CountUnreadNotifications(ctx, userID)
		data["NotificationUnreadCount"] = n
	} else {
		data["NotificationUnreadCount"] = 0
	}
	return nil
}

func (a *app) sidebarInProgressTrips(ctx context.Context, userID string) ([]sidebarInProgressTrip, error) {
	list, err := a.tripService.ListVisibleTrips(ctx, userID)
	if err != nil {
		return nil, err
	}
	return filterDashboardSidebarTrips(list, time.Now(), 2), nil
}

func sortTripsForDashboardSidebar(a, b trips.Trip) bool {
	ti, okI := parseTripStartForSort(a)
	tj, okJ := parseTripStartForSort(b)
	if okI != okJ {
		return okI
	}
	if !okI {
		return strings.ToLower(strings.TrimSpace(a.Name)) < strings.ToLower(strings.TrimSpace(b.Name))
	}
	if !ti.Equal(tj) {
		return ti.Before(tj)
	}
	return strings.ToLower(strings.TrimSpace(a.Name)) < strings.ToLower(strings.TrimSpace(b.Name))
}

// filterDashboardSidebarTrips returns up to max trips for the dashboard sidebar and mobile bottom nav:
// all in-progress trips first (earliest start first), then upcoming trips to fill the remainder.
func filterDashboardSidebarTrips(list []trips.Trip, now time.Time, max int) []sidebarInProgressTrip {
	if max <= 0 {
		return nil
	}
	var inProg, upcoming []trips.Trip
	for _, t := range list {
		_, slug := tripDashboardStatus(t, now)
		switch slug {
		case "in-progress":
			inProg = append(inProg, t)
		case "upcoming":
			upcoming = append(upcoming, t)
		}
	}
	sort.Slice(inProg, func(i, j int) bool { return sortTripsForDashboardSidebar(inProg[i], inProg[j]) })
	sort.Slice(upcoming, func(i, j int) bool { return sortTripsForDashboardSidebar(upcoming[i], upcoming[j]) })
	picked := make([]trips.Trip, 0, max)
	for _, t := range inProg {
		if len(picked) >= max {
			break
		}
		picked = append(picked, t)
	}
	for _, t := range upcoming {
		if len(picked) >= max {
			break
		}
		picked = append(picked, t)
	}
	out := make([]sidebarInProgressTrip, 0, len(picked))
	for _, t := range picked {
		out = append(out, sidebarInProgressTrip{
			ID:        t.ID,
			Name:      t.Name,
			DateRange: formatTripDateShortForTrip(t, t.StartDate, t.EndDate),
		})
	}
	return out
}

func changelogPath() string {
	if p := strings.TrimSpace(os.Getenv("REMI_CHANGELOG_PATH")); p != "" {
		return p
	}
	return filepath.Clean("CHANGELOG.md")
}

// isSafeChangelogHref allows https/http URLs or same-site-style relative paths (no scheme, no script).
func isSafeChangelogHref(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" || len(u) > 800 {
		return false
	}
	if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		return true
	}
	for _, r := range u {
		if r <= ' ' || r == '"' || r == '\'' || r == '<' || r == '>' || r == '\\' {
			return false
		}
		if r == ':' {
			return false
		}
	}
	return true
}

// renderInlineMarkdown converts a small Keep-a-Changelog–friendly subset: **bold**, `code`,
// and [label](url) where url is http(s) or a safe relative path.
func renderInlineMarkdown(raw string) template.HTML {
	var b strings.Builder
	r := raw
	for len(r) > 0 {
		switch {
		case strings.HasPrefix(r, "**"):
			r = r[2:]
			end := strings.Index(r, "**")
			if end < 0 {
				b.WriteString(ht.EscapeString("**" + r))
				return template.HTML(b.String())
			}
			b.WriteString("<strong>")
			b.WriteString(ht.EscapeString(r[:end]))
			b.WriteString("</strong>")
			r = r[end+2:]
		case strings.HasPrefix(r, "`"):
			r = r[1:]
			end := strings.Index(r, "`")
			if end < 0 {
				b.WriteString(ht.EscapeString("`" + r))
				return template.HTML(b.String())
			}
			b.WriteString("<code>")
			b.WriteString(ht.EscapeString(r[:end]))
			b.WriteString("</code>")
			r = r[end+1:]
		case strings.HasPrefix(r, "["):
			closeBracket := strings.Index(r, "]")
			if closeBracket <= 1 || closeBracket+1 >= len(r) || r[closeBracket+1] != '(' {
				b.WriteString(ht.EscapeString(r[:1]))
				r = r[1:]
				continue
			}
			closeParen := strings.Index(r[closeBracket+2:], ")")
			if closeParen < 0 {
				b.WriteString(ht.EscapeString(r[:1]))
				r = r[1:]
				continue
			}
			closeParen += closeBracket + 2
			label := r[1:closeBracket]
			url := strings.TrimSpace(r[closeBracket+2 : closeParen])
			if !isSafeChangelogHref(url) {
				b.WriteString(ht.EscapeString(r[:1]))
				r = r[1:]
				continue
			}
			ext := strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
			b.WriteString(`<a href="`)
			b.WriteString(ht.EscapeString(url))
			b.WriteString(`"`)
			if ext {
				b.WriteString(` rel="noopener noreferrer" target="_blank"`)
			}
			b.WriteString(`>`)
			b.WriteString(ht.EscapeString(label))
			b.WriteString(`</a>`)
			r = r[closeParen+1:]
		default:
			next := len(r)
			for _, marker := range []string{"**", "`", "["} {
				if i := strings.Index(r, marker); i >= 0 && i < next {
					next = i
				}
			}
			b.WriteString(ht.EscapeString(r[:next]))
			r = r[next:]
		}
	}
	return template.HTML(b.String())
}

func renderChangelogSectionHTML(body string) template.HTML {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	var buf strings.Builder
	lines := strings.Split(body, "\n")
	inList := false
	closeList := func() {
		if inList {
			buf.WriteString("</ul>\n")
			inList = false
		}
	}
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		t := strings.TrimSpace(line)
		if t == "" {
			closeList()
			continue
		}
		if strings.HasPrefix(t, "### ") {
			closeList()
			buf.WriteString("<h4 class=\"about-changelog-subhead\">")
			buf.WriteString(string(renderInlineMarkdown(strings.TrimPrefix(t, "### "))))
			buf.WriteString("</h4>\n")
			continue
		}
		if strings.HasPrefix(t, "- ") {
			if !inList {
				buf.WriteString("<ul class=\"about-changelog-list\">\n")
				inList = true
			}
			buf.WriteString("<li>")
			buf.WriteString(string(renderInlineMarkdown(strings.TrimPrefix(t, "- "))))
			buf.WriteString("</li>\n")
			continue
		}
		closeList()
		buf.WriteString("<p class=\"about-changelog-p\">")
		buf.WriteString(string(renderInlineMarkdown(t)))
		buf.WriteString("</p>\n")
	}
	closeList()
	return template.HTML(buf.String())
}

func (a *app) aboutPage(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	section := changelog.TrimSelfHosterNotes(changelog.SectionForVersion(changelogPath(), version.Version))
	data := map[string]any{
		"Settings":      settings,
		"CSRFToken":     CSRFToken(r.Context()),
		"CurrentUser":   CurrentUser(r.Context()),
		"AppVersion":    version.Version,
		"ChangelogHTML": renderChangelogSectionHTML(section),
	}
	if err := a.mergeDashboardShell(r.Context(), uid, "about", "", data); err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "about.html", data)
}

type aboutUpdateCheckResponse struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	CheckOK         bool   `json:"check_ok"`
	Message         string `json:"message,omitempty"`
}

type ghReleaseListItem struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// highestStableReleaseTag returns the greatest semver among published, non-draft, non-prerelease releases.
func highestStableReleaseTag(releases []ghReleaseListItem) string {
	var best string
	for _, rel := range releases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		tag := changelog.NormalizeVersion(rel.TagName)
		if tag == "" {
			continue
		}
		if best == "" || version.Compare(tag, best) > 0 {
			best = tag
		}
	}
	return best
}

func (a *app) aboutUpdateCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	cur := version.Version
	out := aboutUpdateCheckResponse{
		Current: cur,
		Latest:  cur,
		CheckOK: false,
	}
	// GitHub's /releases/latest follows the repo "latest release" flag, which may lag behind
	// newer tags. List releases and take the highest stable semver instead.
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
		"https://api.github.com/repos/"+version.GitHubRepo+"/releases?per_page=100", nil)
	if err != nil {
		out.Message = "Could not prepare request."
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "REMI-Trip-Planner/"+cur)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		out.Message = "Could not reach GitHub."
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.ReadAll(io.LimitReader(resp.Body, 2048))
		out.Message = "Release check failed."
		if resp.StatusCode == http.StatusNotFound {
			out.Message = "No releases found for this repository."
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	var releases []ghReleaseListItem
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&releases); err != nil {
		out.Message = "Could not read releases."
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	latest := highestStableReleaseTag(releases)
	if latest == "" {
		out.Message = "No stable releases found for this repository."
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	out.Latest = latest
	out.CheckOK = true
	out.UpdateAvailable = version.Compare(latest, cur) > 0
	_ = json.NewEncoder(w).Encode(out)
}
