package httpapp

import (
	"testing"

	"remi-trip-planner/internal/version"
)

func TestHighestStableReleaseTag(t *testing.T) {
	releases := []ghReleaseListItem{
		{TagName: "v1.40.0", Draft: false, Prerelease: false},
		{TagName: "v1.47.0", Draft: false, Prerelease: false},
		{TagName: "v1.45.0", Draft: false, Prerelease: false},
	}
	got := highestStableReleaseTag(releases)
	if got != "1.47.0" {
		t.Fatalf("got %q want 1.47.0", got)
	}
	if version.Compare(got, "1.47.0") != 0 {
		t.Fatalf("compare")
	}
}

func TestHighestStableReleaseTagSkipsDraftAndPrerelease(t *testing.T) {
	releases := []ghReleaseListItem{
		{TagName: "v2.0.0-rc1", Draft: false, Prerelease: true},
		{TagName: "v9.9.9", Draft: true, Prerelease: false},
		{TagName: "v1.0.0", Draft: false, Prerelease: false},
	}
	got := highestStableReleaseTag(releases)
	if got != "1.0.0" {
		t.Fatalf("got %q want 1.0.0", got)
	}
}
