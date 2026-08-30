package utils

import (
	"net/url"
	"strings"
)

const avatarAssetPathPrefix = "/api/asset/"

func BuildAvatarURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return avatarAssetPathPrefix + url.PathEscape(value)
}

func NormalizeAvatarValue(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, avatarAssetPathPrefix) {
		return value
	}
	assetID, err := url.PathUnescape(strings.TrimPrefix(value, avatarAssetPathPrefix))
	if err != nil {
		return value
	}
	return strings.TrimSpace(assetID)
}
