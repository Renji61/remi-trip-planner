package httpapp

import (
	"testing"

	"remi-trip-planner/internal/version"
)

func TestHighestStableReleaseTag(t *testing.T) {
	releases := []ghReleaseListItem{
		{TagName: "v1.40.0", Draft: false, Prerelease: false},
		{TagName: "v1.47.0", Draft: false, Prerelease: false},
		{TagName: "v1.49.4", Draft: false, Prerelease: false},
		{TagName: "v1.45.0", Draft: false, Prerelease: false},
	}
	got := highestStableReleaseTag(releases)
	if got != "1.49.4" {
		t.Fatalf("got %q want 1.49.4", got)
	}
	if version.Compare(got, "1.49.4") != 0 {
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

func TestHighestSemverFromGitTagNames(t *testing.T) {
	names := []string{"v1.40.0", "v1.49.4", "v1.45.0", "not-a-version", "v2.0.0-rc1"}
	got := highestSemverFromGitTagNames(names)
	if got != "1.49.4" {
		t.Fatalf("got %q want 1.49.4", got)
	}
}

func TestLatestUpstreamSemverTagsWinWhenReleasesStale(t *testing.T) {
	releases := []ghReleaseListItem{
		{TagName: "v1.40.0", Draft: false, Prerelease: false},
	}
	tags := []string{"v1.49.4"}
	got := latestUpstreamSemver(releases, tags)
	if got != "1.49.4" {
		t.Fatalf("got %q want 1.49.4 (tags must lift above old release)", got)
	}
}

func TestLatestUpstreamSemverReleasesWinWhenHigher(t *testing.T) {
	releases := []ghReleaseListItem{
		{TagName: "v1.50.0", Draft: false, Prerelease: false},
	}
	tags := []string{"v1.49.1"}
	got := latestUpstreamSemver(releases, tags)
	if got != "1.50.0" {
		t.Fatalf("got %q want 1.50.0", got)
	}
}
