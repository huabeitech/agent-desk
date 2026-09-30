package storage

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

var pngSignature = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func TestValidateUploadBlocksBrowserActiveExtensions(t *testing.T) {
	names := []string{
		"session-stealer.html", "page.HTM", "vector.svg", "sheet.xml",
		"transform.xsl", "bundle.min.js", "app.mjs", "shell.php",
		"report.asp", "index.jsp", "legacy.swf", "widget.htc",
	}
	for _, name := range names {
		if _, err := ValidateUpload(name, "application/octet-stream", "application/octet-stream"); err == nil {
			t.Errorf("ValidateUpload(%q) accepted a browser-active extension", name)
		}
	}
}

func TestValidateUploadBlocksStoredXSSPayload(t *testing.T) {
	payload := []byte(`<html><body><script>fetch("/api/dashboard/user/current")</script></body></html>`)
	sniffed, err := SniffContentType(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("SniffContentType() error = %v", err)
	}
	if !IsBlockedMediaType(sniffed) {
		t.Fatalf("expected the payload to sniff as a blocked media type, got %q", sniffed)
	}

	if _, err := ValidateUpload("proof.html", "text/html", sniffed); err == nil {
		t.Error("expected an honestly named HTML upload to be rejected")
	}
	// Renaming the payload is the actual attack: the extension looks like an
	// image and the declared type agrees, so only the bytes can catch it.
	if _, err := ValidateUpload("profile.png", "image/png", sniffed); err == nil {
		t.Error("expected an HTML payload disguised as a PNG to be rejected")
	}
}

func TestValidateUploadBlocksSVGRegardlessOfDeclaredType(t *testing.T) {
	payload := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	sniffed, err := SniffContentType(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("SniffContentType() error = %v", err)
	}
	if _, err := ValidateUpload("logo.svg", "image/svg+xml", sniffed); err == nil {
		t.Error("expected an SVG upload to be rejected")
	}
	// An SVG that arrives without the .svg extension is still refused, because
	// the declared type alone is enough to identify it as an active document.
	if _, err := ValidateUpload("logo.png", "image/svg+xml", sniffed); err == nil {
		t.Error("expected an SVG declared as an image to be rejected")
	}
}

func TestValidateUploadAcceptsOrdinarySupportFiles(t *testing.T) {
	cases := []struct {
		filename string
		declared string
		sniffed  string
		wantMime string
	}{
		{"screenshot.png", "image/png", http.DetectContentType(pngSignature), "image/png"},
		// The client declared a generic type; the payload identified itself.
		{"photo.jpg", "application/octet-stream", "image/jpeg", "image/jpeg"},
		// net/http cannot recognise HEIC, so the declared type has to stand in.
		{"IMG_0001.heic", "image/heic", "application/octet-stream", "image/heic"},
		{"contract.pdf", "application/pdf", "application/pdf", "application/pdf"},
		{"export.csv", "text/csv", "text/plain; charset=utf-8", "text/csv"},
		{"notes.txt", "text/plain", "text/plain; charset=utf-8", "text/plain"},
		{"logs.zip", "application/zip", "application/zip", "application/zip"},
		{"build.apk", "application/vnd.android.package-archive", "application/octet-stream", "application/vnd.android.package-archive"},
		{"data.json", "application/json", "application/json", "application/json"},
	}
	for _, tc := range cases {
		got, err := ValidateUpload(tc.filename, tc.declared, tc.sniffed)
		if err != nil {
			t.Errorf("ValidateUpload(%q) error = %v", tc.filename, err)
			continue
		}
		if got != tc.wantMime {
			t.Errorf("ValidateUpload(%q) mime = %q want %q", tc.filename, got, tc.wantMime)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"report.pdf", "report.pdf"},
		{"../../etc/passwd", "passwd"},
		{`C:\Users\victim\Desktop\evil.html`, "evil.html"},
		{"/absolute/path/notes.txt", "notes.txt"},
		{"tab\tand\nnewline.log", "tabandnewline.log"},
		{"....", ""},
		{"", ""},
		{"   spaced name.pdf   ", "spaced name.pdf"},
	}
	for _, tc := range cases {
		if got := SanitizeFilename(tc.in); got != tc.want {
			t.Errorf("SanitizeFilename(%q) = %q want %q", tc.in, got, tc.want)
		}
	}

	long := strings.Repeat("a", 300) + ".pdf"
	got := SanitizeFilename(long)
	if len(got) > 255 {
		t.Errorf("SanitizeFilename() len = %d want <= 255", len(got))
	}
	if !strings.HasSuffix(got, ".pdf") {
		t.Errorf("SanitizeFilename() = %q, expected the extension to survive truncation", got)
	}

	// A multibyte stem must not be cut in the middle of a rune.
	wide := strings.Repeat("附件", 100) + ".pdf"
	got = SanitizeFilename(wide)
	if len(got) > 255 {
		t.Errorf("SanitizeFilename() len = %d want <= 255", len(got))
	}
	if !utf8.ValidString(got) {
		t.Errorf("SanitizeFilename() = %q is not valid UTF-8", got)
	}
	if !strings.HasSuffix(got, ".pdf") {
		t.Errorf("SanitizeFilename() = %q, expected the extension to survive truncation", got)
	}

	// An extension longer than the whole budget cannot be preserved.
	got = SanitizeFilename("a." + strings.Repeat("b", 300))
	if len(got) > 255 {
		t.Errorf("SanitizeFilename() len = %d want <= 255", len(got))
	}
	if !utf8.ValidString(got) {
		t.Errorf("SanitizeFilename() = %q is not valid UTF-8", got)
	}

	// A name made entirely of dots leaves nothing worth keeping.
	if got = SanitizeFilename(strings.Repeat(".", 300)); got != "" {
		t.Errorf("SanitizeFilename() of a dot-only name = %q want empty", got)
	}
}

func TestSniffContentTypeRewindsTheReader(t *testing.T) {
	src := bytes.NewReader(pngSignature)
	got, err := SniffContentType(src)
	if err != nil {
		t.Fatalf("SniffContentType() error = %v", err)
	}
	if got != "image/png" {
		t.Fatalf("SniffContentType() = %q want image/png", got)
	}

	// The caller still streams the whole payload to storage afterwards, so the
	// reader must be back at the start.
	rest, err := io.ReadAll(src)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(rest, pngSignature) {
		t.Fatalf("reader was not rewound: got %d bytes want %d", len(rest), len(pngSignature))
	}
}

func TestSniffContentTypeHandlesShortAndEmptyPayloads(t *testing.T) {
	short := bytes.NewReader([]byte("hi"))
	got, err := SniffContentType(short)
	if err != nil {
		t.Fatalf("SniffContentType() error = %v", err)
	}
	if !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("SniffContentType() = %q want a text/plain result", got)
	}
	rest, err := io.ReadAll(short)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(rest) != "hi" {
		t.Fatalf("reader was not rewound: got %q", rest)
	}

	empty := bytes.NewReader(nil)
	if _, err := SniffContentType(empty); err != nil {
		t.Fatalf("SniffContentType() on an empty payload error = %v", err)
	}
}

func TestGenerateStorageKeyNeverUsesBlockedExtension(t *testing.T) {
	cases := []struct {
		info UploadInfo
		want string
	}{
		{UploadInfo{Filename: "evil.html", MimeType: "text/html"}, ".bin"},
		{UploadInfo{Filename: "evil.svg", MimeType: "image/svg+xml"}, ".bin"},
		// The extension is gone but the MIME type still names an active
		// document, so the derived extension must be refused too.
		{UploadInfo{Filename: "noext", MimeType: "text/html"}, ".bin"},
		{UploadInfo{Filename: "notes.txt", MimeType: "text/plain"}, ".txt"},
		{UploadInfo{Filename: "photo.png", MimeType: "image/png"}, ".png"},
		{UploadInfo{Filename: "unknown", MimeType: "application/octet-stream"}, ".bin"},
	}
	for _, tc := range cases {
		_, key := GenerateStorageKey(tc.info)
		if !strings.HasSuffix(key, tc.want) {
			t.Errorf("GenerateStorageKey(%+v) = %q, expected it to end in %q", tc.info, key, tc.want)
		}
		if strings.Contains(key, "..") {
			t.Errorf("GenerateStorageKey(%+v) = %q contains a traversal segment", tc.info, key)
		}
	}
}

func TestGetExtByMimeTypeIsPlatformIndependent(t *testing.T) {
	// mime.ExtensionsByType reads the Windows registry, where text/html resolved
	// to ".ehtml" and slipped past the block list. The mapping has to come from
	// our own table so a storage key is the same on every platform, and so a
	// media type that describes an active document maps to nothing at all.
	cases := map[string]string{
		"text/html":              "",
		"application/xhtml+xml":  "",
		"image/svg+xml":          "",
		"text/xml":               "",
		"application/xml":        "",
		"application/javascript": "",
		"text/javascript":        "",
		"image/jpeg":             ".jpg",
		"image/jfif":             ".jpg",
		"image/pjpeg":            ".jpg",
		"image/png":              ".png",
		"application/pdf":        ".pdf",
		"text/csv":               ".csv",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": ".xlsx",
	}
	for mediaType, want := range cases {
		if got := getExtByMimeType(mediaType); got != want {
			t.Errorf("getExtByMimeType(%q) = %q want %q", mediaType, got, want)
		}
	}

	for mediaType := range blockedMediaTypes {
		if got := getExtByMimeType(mediaType); got != "" {
			t.Errorf("getExtByMimeType(%q) = %q, a blocked media type must not yield an extension", mediaType, got)
		}
	}
}

func TestIsPreviewableExtension(t *testing.T) {
	inline := []string{".png", ".JPG", ".jpeg", ".gif", ".webp", ".avif", ".mp4", ".webm", ".mp3", ".pdf"}
	for _, ext := range inline {
		if !IsPreviewableExtension(ext) {
			t.Errorf("IsPreviewableExtension(%q) = false want true", ext)
		}
	}

	download := []string{".html", ".svg", ".xml", ".js", ".zip", ".txt", ".csv", ".docx", ".apk", ""}
	for _, ext := range download {
		if IsPreviewableExtension(ext) {
			t.Errorf("IsPreviewableExtension(%q) = true want false", ext)
		}
	}
}
