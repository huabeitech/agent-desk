package storage

import (
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/strs"
)

func GenerateStorageKey(info UploadInfo) (assetID string, storageKey string) {
	assetID = strs.UUID()
	var (
		env      = currentAssetEnv()
		prefix   = normalizeAssetPrefix(info.Prefix)
		datePath = time.Now().Format("2006/01/02")
		ext      = getExt(info)
	)

	storageKey = filepath.Join(
		env,
		prefix,
		datePath,
		assetID+ext,
	)
	storageKey = strings.TrimLeft(filepath.ToSlash(storageKey), "/")
	return
}

func getExt(info UploadInfo) string {
	ext := normalizeExt(filepath.Ext(strings.TrimSpace(info.Filename)))
	if ext == "" || IsBlockedExtension(ext) {
		ext = normalizeExt(getExtByMimeType(info.MimeType))
	}
	if ext == "" || IsBlockedExtension(ext) {
		// The stored extension decides the Content-Type this origin serves. A
		// browser-active extension must never reach the key, and no extension at
		// all would leave the server to sniff the payload and announce whatever
		// it finds, so both cases settle on an inert binary type.
		return ".bin"
	}
	return ext
}

func getExtByMimeType(mimeType string) string {
	if strs.IsBlank(mimeType) {
		return ""
	}

	mediaType, _, _ := mime.ParseMediaType(mimeType)
	if mediaType == "" {
		return ""
	}

	return safeExtensionForMediaType(mediaType)
}

func normalizeAssetPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.Trim(prefix, "/")
	prefix = strings.ReplaceAll(prefix, "..", "")
	prefix = filepath.ToSlash(prefix)
	for strings.Contains(prefix, "//") {
		prefix = strings.ReplaceAll(prefix, "//", "/")
	}
	return strings.Trim(prefix, "/")
}

func currentAssetEnv() string {
	for _, key := range []string{"APP_ENV", "GO_ENV"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
