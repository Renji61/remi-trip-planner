package changelog

import (
	"os"
	"strings"
)

// SectionForVersion returns the body under the first "## [ver]" heading until the next "## " heading.
func SectionForVersion(changelogPath, ver string) string {
	ver = strings.TrimPrefix(strings.TrimSpace(ver), "v")
	b, err := os.ReadFile(changelogPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(b), "\n")
	start := -1
	for i, line := range lines {
		got, ok := parseVersionHeading(strings.TrimSpace(line))
		if !ok {
			continue
		}
		if got == ver {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	var buf strings.Builder
	for i := start; i < len(lines); i++ {
		line := lines[i]
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") {
			break
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	return strings.TrimSpace(buf.String())
}

func parseVersionHeading(line string) (ver string, ok bool) {
	if !strings.HasPrefix(line, "## [") {
		return "", false
	}
	rest := strings.TrimPrefix(line, "## [")
	end := strings.Index(rest, "]")
	if end < 0 {
		return "", false
	}
	return NormalizeVersion(rest[:end]), true
}

func NormalizeVersion(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}

// isSelfHosterHeading reports a "### Notes for self-hosters" line (case-insensitive on the title).
func isSelfHosterHeading(line string) bool {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "### ") {
		return false
	}
	title := strings.TrimSpace(strings.TrimPrefix(t, "###"))
	return strings.EqualFold(title, "Notes for self-hosters")
}

// TrimSelfHosterNotes removes the "### Notes for self-hosters" subsection for in-app display
// (from that heading through the next "### " heading or end of text).
func TrimSelfHosterNotes(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	skipping := false
	for _, line := range lines {
		if !skipping && isSelfHosterHeading(line) {
			skipping = true
			continue
		}
		if skipping {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "### ") && !isSelfHosterHeading(line) {
				skipping = false
				out = append(out, line)
			}
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
