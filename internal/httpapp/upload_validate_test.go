package httpapp

import (
	"bytes"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilenameHasBlockedExtension(t *testing.T) {
	cases := []struct {
		name string
		fn   string
		want bool
	}{
		{"clean pdf", "invoice.pdf", false},
		{"double ext trap", "evil.php.jpg", true},
		{"exe", "setup.exe", true},
		{"case", "Run.BAT", true},
		{"path traversal name", "..\\x.exe", true},
		{"double dot in name", "file..pdf", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filenameHasBlockedExtension(tc.fn); got != tc.want {
				t.Fatalf("filenameHasBlockedExtension(%q) = %v, want %v", tc.fn, got, tc.want)
			}
		})
	}
}

func TestLooksLikeDangerousBinary(t *testing.T) {
	if err := looksLikeDangerousBinary([]byte{'M', 'Z', 0, 0}); err == nil {
		t.Fatal("expected PE header to be blocked")
	}
	if err := looksLikeDangerousBinary([]byte{0x7f, 'E', 'L', 'F'}); err == nil {
		t.Fatal("expected ELF to be blocked")
	}
	if err := looksLikeDangerousBinary([]byte("#!/bin/sh\n")); err == nil {
		t.Fatal("expected shebang to be blocked")
	}
	if err := looksLikeDangerousBinary([]byte("<?php echo 1;")); err == nil {
		t.Fatal("expected php open tag to be blocked")
	}
	if err := looksLikeDangerousBinary([]byte("%PDF-1.4")); err != nil {
		t.Fatalf("pdf prefix should be allowed: %v", err)
	}
}

func TestDetectSniffKindJPEG(t *testing.T) {
	head := []byte{0xff, 0xd8, 0xff, 0xe0}
	if k := detectSniffKind(head, "x.jpg"); k != sniffJPEG {
		t.Fatalf("got %v, want sniffJPEG", k)
	}
}

func TestSaveValidatedUploadRejectsBlockedName(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.MkdirAll(filepath.Join("web", "static", "uploads", "t"), 0o755); err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader([]byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0})
	h := &multipart.FileHeader{Filename: "photo.exe.jpg", Size: int64(body.Len())}
	_, err := SaveValidatedUpload(ioReaderAsMultipartFile(body), h, 1<<20, []string{"t"}, UploadProfileImageOnly)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected blocked extension error, got %v", err)
	}
}

func TestSaveValidatedUploadRejectsSVGForImageOnly(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.MkdirAll(filepath.Join("web", "static", "uploads", "t"), 0o755); err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
	h := &multipart.FileHeader{Filename: "image.svg", Size: int64(body.Len())}
	_, err := SaveValidatedUpload(ioReaderAsMultipartFile(body), h, 1<<20, []string{"t"}, UploadProfileImageOnly)
	if !errors.Is(err, ErrUploadContentMismatch) {
		t.Fatalf("expected svg to be rejected for image uploads, got %v", err)
	}
}

func TestSaveValidatedUploadRejectsSVGForBookingAttachment(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.MkdirAll(filepath.Join("web", "static", "uploads", "t"), 0o755); err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
	h := &multipart.FileHeader{Filename: "document.svg", Size: int64(body.Len())}
	_, err := SaveValidatedUpload(ioReaderAsMultipartFile(body), h, 1<<20, []string{"t"}, UploadProfileBookingAttachment)
	if !errors.Is(err, ErrUploadContentMismatch) {
		t.Fatalf("expected svg to be rejected for booking attachments, got %v", err)
	}
}

// ioReaderAsMultipartFile wraps *bytes.Reader as multipart.File (Read + Seek + Close).
type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func ioReaderAsMultipartFile(r *bytes.Reader) multipart.File {
	return nopCloser{r}
}
