package httpapp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// User-facing upload errors (safe to return in HTTP bodies / toasts).
var (
	ErrUploadBlockedExecutable = errors.New("this file type is not allowed — executables and scripts (.exe, .sh, .bat, .msi, .js, .py, .php, etc.) cannot be uploaded")
	ErrUploadContentMismatch   = errors.New("file contents do not match an allowed type — upload a PDF, image, or supported document only")
	ErrUploadEmpty             = errors.New("the selected file is empty")
)

// UploadProfile selects which content types are accepted after dangerous signatures are rejected.
type UploadProfile int

const (
	// UploadProfileImageOnly: JPEG, PNG, GIF, WebP, BMP, TIFF, HEIC/HEIF.
	UploadProfileImageOnly UploadProfile = iota
	// UploadProfileBookingAttachment: images + PDF + legacy .doc (OLE) + .docx/.xlsx (ZIP OOXML).
	UploadProfileBookingAttachment
	// UploadProfileReceipt: PDF or images (group expense receipts).
	UploadProfileReceipt
)

var blockedNamePart = map[string]bool{
	"exe": true, "sh": true, "bat": true, "msi": true, "js": true, "py": true, "php": true,
	"cmd": true, "ps1": true, "com": true, "scr": true, "vbs": true, "jar": true,
	"dll": true, "app": true, "deb": true, "rpm": true, "dmg": true, "wsf": true,
	"hta": true, "lnk": true,
}

type sniffKind int

const (
	sniffUnknown sniffKind = iota
	sniffPDF
	sniffJPEG
	sniffPNG
	sniffGIF
	sniffWEBP
	sniffBMP
	sniffTIFF
	sniffHEIC
	sniffOLE // legacy MS Office .doc
	sniffZIP // OOXML / generic zip (validated with extension)
	sniffSVG
)

// filenameHasBlockedExtension checks every dotted segment so evil.php.jpg is caught.
func filenameHasBlockedExtension(filename string) bool {
	base := filepath.Base(filename)
	if base == "" || base == "." || strings.Contains(base, "..") {
		return true
	}
	for _, r := range base {
		if r == 0 || r == '/' || r == '\\' {
			return true
		}
	}
	lower := strings.ToLower(base)
	for _, part := range strings.Split(lower, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if blockedNamePart[part] {
			return true
		}
	}
	return false
}

func trimBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		return b[3:]
	}
	if len(b) >= 2 && b[0] == 0xfe && b[1] == 0xff {
		return b[2:]
	}
	if len(b) >= 2 && b[0] == 0xff && b[1] == 0xfe {
		return b[2:]
	}
	return b
}

func looksLikeDangerousBinary(head []byte) error {
	if len(head) >= 2 && head[0] == 'M' && head[1] == 'Z' {
		return ErrUploadBlockedExecutable
	}
	if len(head) >= 4 && head[0] == 0x7f && head[1] == 'E' && head[2] == 'L' && head[3] == 'F' {
		return ErrUploadBlockedExecutable
	}
	if len(head) >= 2 && head[0] == '#' && head[1] == '!' {
		return ErrUploadBlockedExecutable
	}
	t := trimBOM(head)
	if len(t) >= 5 && bytes.EqualFold(t[:5], []byte("<?php")) {
		return ErrUploadBlockedExecutable
	}
	if len(t) >= 4 && (bytes.EqualFold(t[:4], []byte("<?=")) || bytes.EqualFold(t[:4], []byte("<script"))) {
		return ErrUploadBlockedExecutable
	}
	return nil
}

func detectSniffKind(head []byte, filename string) sniffKind {
	if len(head) == 0 {
		return sniffUnknown
	}
	if len(head) >= 4 && bytes.HasPrefix(head, []byte("%PDF")) {
		return sniffPDF
	}
	if len(head) >= 3 && head[0] == 0xff && head[1] == 0xd8 && head[2] == 0xff {
		return sniffJPEG
	}
	if len(head) >= 8 && bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		return sniffPNG
	}
	if len(head) >= 6 && (bytes.HasPrefix(head, []byte("GIF87a")) || bytes.HasPrefix(head, []byte("GIF89a"))) {
		return sniffGIF
	}
	if len(head) >= 12 && bytes.HasPrefix(head, []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")) {
		return sniffWEBP
	}
	if len(head) >= 2 && head[0] == 'B' && head[1] == 'M' {
		return sniffBMP
	}
	if len(head) >= 4 && (bytes.HasPrefix(head, []byte("II*\x00")) || bytes.HasPrefix(head, []byte("MM\x00*"))) {
		return sniffTIFF
	}
	// ISO BMFF (HEIC / AVIF / MP4 family): 'ftyp' at offset 4.
	if len(head) >= 12 && string(head[4:8]) == "ftyp" {
		brand := string(head[8:12])
		if strings.Contains(strings.ToLower(brand), "heic") || strings.Contains(strings.ToLower(brand), "heix") ||
			strings.Contains(strings.ToLower(brand), "mif1") || strings.Contains(strings.ToLower(brand), "msf1") {
			return sniffHEIC
		}
	}
	// Legacy OLE Compound Document (.doc, .xls)
	if len(head) >= 8 && head[0] == 0xd0 && head[1] == 0xcf && head[2] == 0x11 && head[3] == 0xe0 {
		return sniffOLE
	}
	if len(head) >= 4 && head[0] == 'P' && head[1] == 'K' && (head[2] == 3 || head[2] == 5 || head[2] == 7) && (head[3] == 4 || head[3] == 6 || head[3] == 8) {
		return sniffZIP
	}
	t := bytes.TrimLeftFunc(head, unicode.IsSpace)
	if len(t) > 0 && t[0] == '<' {
		lt := bytes.ToLower(t)
		if bytes.HasPrefix(lt, []byte("<?xml")) || bytes.HasPrefix(lt, []byte("<!doctype")) || bytes.HasPrefix(lt, []byte("<svg")) {
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
			if ext == "svg" {
				return sniffSVG
			}
		}
	}
	return sniffUnknown
}

func extFromDeclared(filename string) string {
	e := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if e == "" {
		return ""
	}
	return e
}

func canonicalExtForKind(k sniffKind, declaredExt string) string {
	d := declaredExt
	switch k {
	case sniffPDF:
		return ".pdf"
	case sniffJPEG:
		if d == ".jpeg" {
			return ".jpeg"
		}
		return ".jpg"
	case sniffPNG:
		return ".png"
	case sniffGIF:
		return ".gif"
	case sniffWEBP:
		return ".webp"
	case sniffBMP:
		return ".bmp"
	case sniffTIFF:
		if d == ".tif" {
			return ".tif"
		}
		return ".tiff"
	case sniffHEIC:
		if d == ".heif" {
			return ".heif"
		}
		return ".heic"
	case sniffOLE:
		return ".doc"
	case sniffZIP:
		switch d {
		case ".docx", ".xlsx", ".pptx":
			return d
		default:
			return ""
		}
	case sniffSVG:
		return ".svg"
	default:
		return ""
	}
}

func validateProfile(k sniffKind, profile UploadProfile, declaredExt string) error {
	de := strings.ToLower(declaredExt)
	switch profile {
	case UploadProfileImageOnly:
		switch k {
		case sniffJPEG, sniffPNG, sniffGIF, sniffWEBP, sniffBMP, sniffTIFF, sniffHEIC, sniffSVG:
			return nil
		default:
			return ErrUploadContentMismatch
		}
	case UploadProfileReceipt:
		switch k {
		case sniffPDF, sniffJPEG, sniffPNG, sniffGIF, sniffWEBP, sniffBMP, sniffTIFF, sniffHEIC:
			return nil
		default:
			return ErrUploadContentMismatch
		}
	case UploadProfileBookingAttachment:
		switch k {
		case sniffPDF, sniffJPEG, sniffPNG, sniffGIF, sniffWEBP, sniffBMP, sniffTIFF, sniffHEIC, sniffOLE:
			return nil
		case sniffZIP:
			if de == ".docx" || de == ".xlsx" || de == ".pptx" {
				return nil
			}
			return ErrUploadContentMismatch
		case sniffSVG:
			if de == ".svg" {
				return nil
			}
			return ErrUploadContentMismatch
		default:
			return ErrUploadContentMismatch
		}
	default:
		return ErrUploadContentMismatch
	}
}

const uploadSniffLen = 512

// SaveValidatedUpload reads and stores a multipart file under web/static/uploads/{relDir...}/<uuid>.<ext>.
// relDir is path segments, e.g. []string{"bookings"} or []string{"expenses", tripID}.
func SaveValidatedUpload(file multipart.File, header *multipart.FileHeader, maxBytes int64, relDir []string, profile UploadProfile) (webPath string, err error) {
	if header == nil {
		return "", errors.New("missing upload")
	}
	name := strings.TrimSpace(header.Filename)
	if name == "" {
		return "", errors.New("missing filename")
	}
	if filenameHasBlockedExtension(name) {
		return "", ErrUploadBlockedExecutable
	}
	if maxBytes > 0 && header.Size > maxBytes {
		return "", fmt.Errorf("file exceeds upload limit (%d MB)", maxBytes/(1024*1024))
	}

	sniff := make([]byte, uploadSniffLen)
	n, readErr := io.ReadFull(file, sniff)
	if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
		readErr = nil
	}
	if readErr != nil {
		return "", readErr
	}
	if n == 0 {
		return "", ErrUploadEmpty
	}
	head := sniff[:n]

	if err := looksLikeDangerousBinary(head); err != nil {
		return "", err
	}

	k := detectSniffKind(head, name)
	de := extFromDeclared(name)
	ext := canonicalExtForKind(k, de)
	if ext == "" {
		return "", ErrUploadContentMismatch
	}
	if err := validateProfile(k, profile, de); err != nil {
		return "", err
	}

	storedName := uuid.NewString() + ext
	targetDir := filepath.Join(append([]string{"web", "static", "uploads"}, relDir...)...)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, storedName)
	dst, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}

	body := io.MultiReader(bytes.NewReader(head), file)
	reader := io.Reader(body)
	if maxBytes > 0 {
		reader = io.LimitReader(body, maxBytes+1)
	}
	written, copyErr := io.Copy(dst, reader)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return "", closeErr
	}
	if maxBytes > 0 && written > maxBytes {
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("file exceeds upload limit (%d MB)", maxBytes/(1024*1024))
	}

	webRel := filepath.Join(append(relDir, storedName)...)
	return "/static/uploads/" + filepath.ToSlash(webRel), nil
}

// SaveValidatedUploadFromHeader opens multipart.FileHeader and saves (for trip document batch uploads).
func SaveValidatedUploadFromHeader(fh *multipart.FileHeader, maxBytes int64, relDir []string, profile UploadProfile) (webPath string, err error) {
	if fh == nil {
		return "", errors.New("missing upload")
	}
	file, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	return SaveValidatedUpload(file, fh, maxBytes, relDir, profile)
}
