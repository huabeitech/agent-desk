package storage

import (
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"agent-desk/internal/pkg/errorsx"
)

const (
	// sniffLimit is the number of leading bytes net/http.DetectContentType inspects.
	sniffLimit = 512
	// maxFilenameLength keeps a stored filename inside the column that holds it.
	maxFilenameLength = 255
)

// previewableExtensions are the only stored file types a browser may render
// inline. Every other extension is forced to download.
//
// Assets are served from this application's own origin, so an inline response is
// same-origin content: a browser that navigates to it runs whatever the bytes
// ask for, in the same security context as the dashboard and the public support
// widget. Forcing a download for anything we have not vetted keeps that true
// only for media, which cannot touch the DOM.
var previewableExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".bmp": true, ".ico": true, ".avif": true, ".tif": true, ".tiff": true,
	".heic": true, ".heif": true,
	".mp4": true, ".m4v": true, ".webm": true, ".mov": true, ".ogv": true,
	".mp3": true, ".m4a": true, ".wav": true, ".ogg": true, ".oga": true,
	".aac": true, ".flac": true,
	".pdf": true,
}

// blockedExtensions are rejected at upload time. Each one names a document a
// browser executes or lays out instead of displaying, so accepting it would let
// an unauthenticated visitor plant a script that runs against this origin the
// moment a staff member opens the attachment link.
var blockedExtensions = map[string]bool{
	".html": true, ".htm": true, ".shtml": true, ".xhtml": true, ".xht": true,
	".svg": true, ".svgz": true,
	".xml": true, ".xsl": true, ".xslt": true, ".xsd": true, ".wsdl": true,
	".js": true, ".mjs": true, ".cjs": true,
	".swf": true,
	".php": true, ".phtml": true, ".php3": true, ".php4": true, ".php5": true,
	".asp": true, ".aspx": true, ".jsp": true, ".jspx": true, ".cfm": true,
	".hta": true, ".htc": true, ".htaccess": true,
}

// blockedMediaTypes are the payloads a browser renders as an active document.
// They are rejected regardless of the extension the client chose, so renaming a
// page to .png does not smuggle it past the extension check.
var blockedMediaTypes = map[string]bool{
	"text/html":                true,
	"application/xhtml+xml":    true,
	"image/svg+xml":            true,
	"text/xml":                 true,
	"application/xml":          true,
	"application/xslt+xml":     true,
	"text/javascript":          true,
	"application/javascript":   true,
	"application/x-javascript": true,
}

// blockedUploadI18nKey is returned to the uploader whenever the file-safety
// policy rejects a payload. It deliberately does not name the offending type.
const blockedUploadI18nKey = "error.e0348"

// mediaTypeExtensions maps a MIME type to the extension it should be stored
// under. mime.ExtensionsByType cannot be used here: on Windows it consults the
// registry, so the same type resolves to a different extension than in
// production, and it happily hands back an extension that describes an active
// document. Types that must never become a servable document are absent, which
// leaves the caller with the inert .bin fallback.
var mediaTypeExtensions = map[string]string{
	// images
	"image/jpeg": ".jpg", "image/jfif": ".jpg", "image/pjpeg": ".jpg",
	"image/png": ".png", "image/gif": ".gif", "image/webp": ".webp",
	"image/bmp": ".bmp", "image/tiff": ".tiff", "image/avif": ".avif",
	"image/heic": ".heic", "image/heif": ".heif", "image/x-icon": ".ico",
	"image/vnd.microsoft.icon": ".ico",
	// documents
	"application/pdf": ".pdf", "text/plain": ".txt", "text/csv": ".csv",
	"text/markdown": ".md", "application/json": ".json", "application/rtf": ".rtf",
	"application/msword": ".doc",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	"application/vnd.ms-excel": ".xls",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
	"application/vnd.ms-powerpoint":                                             ".ppt",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
	"application/vnd.oasis.opendocument.text":                                   ".odt",
	"application/vnd.oasis.opendocument.spreadsheet":                            ".ods",
	"application/vnd.oasis.opendocument.presentation":                           ".odp",
	// archives
	"application/zip": ".zip", "application/x-zip-compressed": ".zip",
	"application/x-rar-compressed": ".rar", "application/vnd.rar": ".rar",
	"application/x-7z-compressed": ".7z", "application/x-tar": ".tar",
	"application/gzip": ".gz", "application/x-bzip2": ".bz2", "application/x-xz": ".xz",
	// audio
	"audio/mpeg": ".mp3", "audio/mp4": ".m4a", "audio/x-m4a": ".m4a",
	"audio/wav": ".wav", "audio/x-wav": ".wav", "audio/ogg": ".ogg",
	"audio/aac": ".aac", "audio/flac": ".flac", "audio/webm": ".webm",
	// video
	"video/mp4": ".mp4", "video/webm": ".webm", "video/quicktime": ".mov",
	"video/x-msvideo": ".avi", "video/x-matroska": ".mkv", "video/ogg": ".ogv",
	"video/mpeg": ".mpeg",
}

// safeExtensionForMediaType returns the extension a payload of this type should
// be stored under, or "" when the type is not one this origin is willing to
// serve as a document.
func safeExtensionForMediaType(mediaType string) string {
	return mediaTypeExtensions[strings.ToLower(strings.TrimSpace(mediaType))]
}

// IsPreviewableExtension reports whether a stored file may be rendered inline.
func IsPreviewableExtension(ext string) bool {
	return previewableExtensions[normalizeExt(ext)]
}

// IsBlockedExtension reports whether an extension names a browser-active document.
func IsBlockedExtension(ext string) bool {
	return blockedExtensions[normalizeExt(ext)]
}

// IsBlockedMediaType reports whether a MIME type describes a browser-active document.
func IsBlockedMediaType(mediaType string) bool {
	parsed, _, err := mime.ParseMediaType(strings.TrimSpace(mediaType))
	if err != nil || parsed == "" {
		return false
	}
	return blockedMediaTypes[strings.ToLower(parsed)]
}

// SanitizeFilename reduces a client-supplied name to a bare basename. Uploads
// arrive from browsers, mobile SDKs and channel webhooks, all of which are free
// to send a full path, control characters or nothing at all; the result is stored
// on the asset and echoed back into message payloads and download links.
func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(strings.TrimRight(name, ". "))
	if len(name) > maxFilenameLength {
		ext := path.Ext(name)
		if len(ext) > maxFilenameLength {
			ext = ""
		}
		stem := strings.TrimSuffix(name, ext)
		// Cut on a rune boundary: a partial multibyte character would not survive
		// the round trip through a utf8mb4 column.
		limit := maxFilenameLength - len(ext)
		for limit > 0 && !utf8.RuneStart(stem[limit]) {
			limit--
		}
		name = stem[:limit] + ext
	}
	return name
}

// SniffContentType reports what the leading bytes of a seekable payload actually
// are, then rewinds so the caller can still stream the whole file to storage.
// The Content-Type a client declares is a claim, not evidence.
func SniffContentType(src io.ReadSeeker) (string, error) {
	head := make([]byte, sniffLimit)
	read, err := io.ReadFull(src, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(head[:read]), nil
}

// ValidateUpload applies the file-safety policy to an upload and returns the MIME
// type that should be recorded on the asset.
//
// sniffed comes from SniffContentType and wins whenever it identified something
// specific; declared is what the client claimed and is only trusted for the
// formats net/http cannot recognise, such as HEIC and most Office documents.
func ValidateUpload(filename, declared, sniffed string) (string, error) {
	if IsBlockedExtension(path.Ext(filename)) {
		return "", errorsx.InvalidParamI18n(blockedUploadI18nKey)
	}
	if IsBlockedMediaType(sniffed) || IsBlockedMediaType(declared) {
		return "", errorsx.InvalidParamI18n(blockedUploadI18nKey)
	}

	mediaType, _, _ := mime.ParseMediaType(sniffed)
	if mediaType != "" && mediaType != "application/octet-stream" && !strings.HasPrefix(mediaType, "text/") {
		return sniffed, nil
	}
	if declared != "" {
		return declared, nil
	}
	return sniffed, nil
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}
