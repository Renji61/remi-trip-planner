package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSectionForVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	content := `# Log

## [1.0.0] - day

### Added

- First

## [2.0.0] - later

### Fixed

- Two
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := SectionForVersion(path, "1.0.0")
	if !strings.Contains(got, "First") || strings.Contains(got, "Two") {
		t.Fatalf("unexpected section: %q", got)
	}
}

func TestTrimSelfHosterNotes(t *testing.T) {
	in := `### Added

- One

### Notes for self-hosters

- Admin note

### Changed

- Two`
	got := TrimSelfHosterNotes(in)
	if strings.Contains(got, "self-hosters") || strings.Contains(got, "Admin note") {
		t.Fatalf("expected self-hoster block removed: %q", got)
	}
	if !strings.Contains(got, "One") || !strings.Contains(got, "Changed") || !strings.Contains(got, "Two") {
		t.Fatalf("expected other sections kept: %q", got)
	}
	last := `### Fixed

- x

### Notes for self-hosters

- tail`
	got2 := TrimSelfHosterNotes(last)
	if strings.Contains(got2, "self-hosters") || strings.Contains(got2, "tail") {
		t.Fatalf("expected trailing block removed: %q", got2)
	}
	if !strings.Contains(got2, "Fixed") {
		t.Fatalf("expected prior section kept: %q", got2)
	}
}
