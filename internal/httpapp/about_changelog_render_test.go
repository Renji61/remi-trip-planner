package httpapp

import (
	"strings"
	"testing"
)

func TestRenderInlineMarkdown(t *testing.T) {
	s := string(renderInlineMarkdown("**About** page and `GET /x`"))
	if !strings.Contains(s, "<strong>About</strong>") {
		t.Fatalf("bold: %q", s)
	}
	if !strings.Contains(s, "<code>GET /x</code>") {
		t.Fatalf("code: %q", s)
	}
}

func TestRenderInlineMarkdownLink(t *testing.T) {
	s := string(renderInlineMarkdown(`see [docs](docs/foo.md) and [ext](https://example.com/x)`))
	if !strings.Contains(s, `href="docs/foo.md"`) {
		t.Fatalf("relative link: %q", s)
	}
	if !strings.Contains(s, `href="https://example.com/x"`) || !strings.Contains(s, `target="_blank"`) {
		t.Fatalf("external link: %q", s)
	}
}

func TestRenderChangelogSectionHTMLBold(t *testing.T) {
	body := "### Added\n\n- **About** page (`/about`).\n"
	h := string(renderChangelogSectionHTML(body))
	if strings.Contains(h, "**") {
		t.Fatalf("raw markdown left: %q", h)
	}
	if !strings.Contains(h, "<strong>About</strong>") {
		t.Fatalf("expected strong: %q", h)
	}
}
