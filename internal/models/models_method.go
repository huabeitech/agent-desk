package models

import (
	"strconv"
	"strings"
)

func (u User) UserAvatarURL() string {
	if isExternalAvatarURL(u.Avatar) {
		return strings.TrimSpace(u.Avatar)
	}
	if u.UserAvatarAssetID() == "" {
		return ""
	}
	return "/api/avatar/user/" + strconv.FormatInt(u.ID, 10)
}

func (u User) UserAvatarAssetID() string {
	if isExternalAvatarURL(u.Avatar) {
		return ""
	}
	return strings.TrimSpace(u.Avatar)
}

func (u AgentProfile) AgentAvatar() string {
	if isExternalAvatarURL(u.Avatar) {
		return strings.TrimSpace(u.Avatar)
	}
	if u.AgentAvatarAssetID() == "" {
		return ""
	}
	return "/api/avatar/agent/" + strconv.FormatInt(u.ID, 10)
}

func (u AgentProfile) AgentAvatarAssetID() string {
	if isExternalAvatarURL(u.Avatar) {
		return ""
	}
	return strings.TrimSpace(u.Avatar)
}

func isExternalAvatarURL(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
