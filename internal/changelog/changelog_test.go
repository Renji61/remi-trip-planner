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
