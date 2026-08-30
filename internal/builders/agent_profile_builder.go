package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/services"
	"strconv"
)

func BuildAgentProfileList(items []models.AgentProfile) []response.AgentProfileResponse {
	if len(items) == 0 {
		return []response.AgentProfileResponse{}
	}

	userIDs := make([]int64, 0, len(items))
	teamIDs := make([]int64, 0, len(items))
	for _, item := range items {
		if item.UserID > 0 {
			userIDs = append(userIDs, item.UserID)
		}
		if item.TeamID > 0 {
			teamIDs = append(teamIDs, item.TeamID)
		}
	}

	users := services.UserService.FindByIds(userIDs)
	teams := services.AgentTeamService.FindByIds(teamIDs)

	userMap := make(map[int64]*models.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}

	teamMap := make(map[int64]*models.AgentTeam, len(teams))
	for i := range teams {
		teamMap[teams[i].ID] = &teams[i]
	}

	results := make([]response.AgentProfileResponse, 0, len(items))
	for _, item := range items {
		if result := doBuildAgentProfileResponse(&item, userMap[item.UserID], teamMap[item.TeamID]); result != nil {
			results = append(results, *result)
		}
	}
	return results
}

func BuildAgentProfileResponse(item *models.AgentProfile) *response.AgentProfileResponse {
	user := services.UserService.Get(item.UserID)
	team := services.AgentTeamService.Get(item.TeamID)
	return doBuildAgentProfileResponse(item, user, team)
}

func doBuildAgentProfileResponse(item *models.AgentProfile, user *models.User, team *models.AgentTeam) *response.AgentProfileResponse {
	if item == nil {
		return nil
	}
	ret := &response.AgentProfileResponse{
		ID:                    item.ID,
		UserID:                item.UserID,
		TeamID:                item.TeamID,
		AgentCode:             item.AgentCode,
		DisplayName:           item.DisplayName,
		Avatar:                utils.BuildAvatarURL(item.Avatar, "/api/avatar/agent/"+strconv.FormatInt(item.ID, 10)),
		AvatarAssetID:         utils.AvatarAssetID(item.Avatar),
		ServiceStatus:         item.ServiceStatus,
		MaxConcurrentCount:    item.MaxConcurrentCount,
		PriorityLevel:         item.PriorityLevel,
		AutoAssignEnabled:     item.AutoAssignEnabled,
		ReceiveOfflineMessage: item.ReceiveOfflineMessage,
		LastOnlineAt:          utils.FormatTimePtr(item.LastOnlineAt),
		LastStatusAt:          utils.FormatTimePtr(item.LastStatusAt),
		Remark:                item.Remark,
	}
	if user != nil {
		ret.Username = user.Username
		ret.Nickname = user.Nickname
	}
	if team != nil {
		ret.TeamName = team.Name
	}
	return ret
}
