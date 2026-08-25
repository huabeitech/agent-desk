package bootstrap

import (
	"agent-desk/internal/handlers/api"
	"agent-desk/internal/handlers/dashboard"
	"agent-desk/internal/handlers/third"

	"github.com/gin-gonic/gin"
)

func registerApiAuthRoutes(group *gin.RouterGroup) {
	group.POST("/login", api.Login)
	group.POST("/logout", api.Logout)
	group.GET("/profile", api.Profile)
	group.POST("/profile/update", api.UpdateProfile)
	group.POST("/profile/avatar/upload", api.UploadProfileAvatar)
	group.GET("/wxwork_callback", api.WxWorkCallback)
	group.POST("/wxwork_exchange", api.WxWorkExchange)
	group.GET("/wxwork_login", api.WxWorkLogin)
	group.GET("/wxwork_qr_login", api.WxWorkQRLogin)
	group.GET("/oidc_callback", api.OIDCCallback)
	group.GET("/callback/custom", api.OIDCCallback)
	group.POST("/oidc_exchange", api.OIDCExchange)
	group.GET("/oidc_login", api.OIDCLogin)
}

func registerApiChannelRoutes(group *gin.RouterGroup) {
	group.Any("/config", api.ChannelAnyConfig)
}

func registerApiWebhookRoutes(group *gin.RouterGroup) {
	group.POST("/org-sync", api.OrgSyncWebhook)
	group.POST("/dos-org-sync", api.DOSOrgSyncWebhook)
}

func registerApiCustomerRoutes(group *gin.RouterGroup) {
	group.POST("/session_exchange", api.CustomerPostSession_exchange)
}

func registerApiConversationRoutes(group *gin.RouterGroup) {
	group.GET("/:id", api.ConversationGetBy)
	group.POST("/close", api.ConversationPostClose)
	group.POST("/create_or_match", api.ConversationPostCreate_or_match)
}

func registerApiMessageRoutes(group *gin.RouterGroup) {
	group.Any("/list", api.MessageAnyList)
	group.POST("/read", api.MessagePostRead)
	group.POST("/send", api.MessagePostSend)
	group.POST("/upload_attachment", api.MessagePostUpload_attachment)
	group.POST("/upload_image", api.MessagePostUpload_image)
}

func registerApiSupportRoutes(group *gin.RouterGroup) {
	group.GET("/config", api.SupportConfigGetConfig)
	group.GET("/ai-customer-service/user-token", api.SupportConfigGetAICustomerServiceUserToken)
	group.POST("/auth/register", api.SupportAuthPostRegister)
	group.GET("/me", api.SupportGetMe)
	group.Any("/doc-page/list", api.DocPageAnyList)
	group.GET("/doc-page/navigation", api.DocPageGetNavigation)
	group.GET("/doc-page/:id", api.DocPageGetBy)
	group.POST("/doc-page/feedback", api.DocPagePostFeedback)
	group.Any("/community/categories/list", api.CategoryAnyList)
	group.Any("/community/posts/list", api.PostAnyList)
	group.GET("/community/posts/:id", api.PostGetBy)
	group.POST("/community/posts/create", api.PostPostCreate)
	group.POST("/community/posts/update", api.PostPostUpdate)
	group.POST("/community/posts/accept_comment", api.PostPostAcceptComment)
	group.Any("/community/comments/list", api.CommentAnyList)
	group.POST("/community/comments/create", api.CommentPostCreate)
	group.POST("/community/comments/update", api.CommentPostUpdate)
	group.POST("/community/comments/delete", api.CommentPostDelete)
	group.POST("/community/comments/report", api.CommentPostReport)
	group.POST("/community/reactions/toggle", api.ReactionPostToggle)
}

func registerDashboardSupportConfigRoutes(group *gin.RouterGroup) {
	group.GET("", dashboard.SupportConfigGetConfig)
	group.POST("/save", dashboard.SupportConfigPostSave)
}

func registerDashboardDashboardRoutes(group *gin.RouterGroup) {
	group.GET("/overview", dashboard.DashboardGetOverview)
}

func registerDashboardUserRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.UserGetBy)
	group.POST("/assign_role", dashboard.UserPostAssign_role)
	group.POST("/change_password", dashboard.UserPostChange_password)
	group.POST("/create", dashboard.UserPostCreate)
	group.POST("/delete", dashboard.UserPostDelete)
	group.Any("/list", dashboard.UserAnyList)
	group.Any("/list_all", dashboard.UserAnyList_all)
	group.POST("/reset_password", dashboard.UserPostReset_password)
	group.POST("/update", dashboard.UserPostUpdate)
	group.POST("/update_status", dashboard.UserPostUpdate_status)
}

func registerDashboardOrganizationRoutes(group *gin.RouterGroup) {
	group.GET("/my_list", dashboard.OrganizationUserList)
	group.POST("/switch", dashboard.OrganizationSwitch)
}

func registerDashboardCompanyRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.CompanyGetBy)
	group.POST("/create", dashboard.CompanyPostCreate)
	group.POST("/delete", dashboard.CompanyPostDelete)
	group.Any("/list", dashboard.CompanyAnyList)
	group.POST("/update", dashboard.CompanyPostUpdate)
	group.POST("/update_status", dashboard.CompanyPostUpdate_status)
}

func registerDashboardCustomerRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.CustomerGetBy)
	group.POST("/create", dashboard.CustomerPostCreate)
	group.POST("/delete", dashboard.CustomerPostDelete)
	group.POST("/list", dashboard.CustomerPostList)
	group.POST("/save_profile", dashboard.CustomerPostSave_profile)
	group.POST("/update", dashboard.CustomerPostUpdate)
	group.POST("/update_status", dashboard.CustomerPostUpdate_status)
}

func registerDashboardCustomerContactRoutes(group *gin.RouterGroup) {
	group.POST("/create", dashboard.CustomerContactPostCreate)
	group.POST("/delete", dashboard.CustomerContactPostDelete)
	group.Any("/list", dashboard.CustomerContactAnyList)
	group.POST("/update", dashboard.CustomerContactPostUpdate)
}

func registerDashboardRoleRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.RoleGetBy)
	group.POST("/assign_permission", dashboard.RolePostAssign_permission)
	group.POST("/create", dashboard.RolePostCreate)
	group.POST("/delete", dashboard.RolePostDelete)
	group.Any("/list", dashboard.RoleAnyList)
	group.GET("/list_all", dashboard.RoleGetList_all)
	group.POST("/update", dashboard.RolePostUpdate)
	group.POST("/update_sort", dashboard.RolePostUpdate_sort)
	group.POST("/update_status", dashboard.RolePostUpdate_status)
}

func registerDashboardPermissionRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.PermissionGetBy)
	group.Any("/list", dashboard.PermissionAnyList)
	group.POST("/sync", dashboard.PermissionPostSync)
}

func registerDashboardSessionRoutes(group *gin.RouterGroup) {
	group.Any("/list", dashboard.SessionAnyList)
	group.POST("/revoke", dashboard.SessionPostRevoke)
	group.POST("/revoke/by/user", dashboard.SessionPostRevokeByUser)
}

func registerDashboardTagRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.TagGetBy)
	group.POST("/create", dashboard.TagPostCreate)
	group.POST("/delete", dashboard.TagPostDelete)
	group.Any("/list", dashboard.TagAnyList)
	group.GET("/list_all", dashboard.TagGetList_all)
	group.POST("/update", dashboard.TagPostUpdate)
	group.POST("/update_sort", dashboard.TagPostUpdate_sort)
	group.POST("/update_status", dashboard.TagPostUpdate_status)
}

func registerDashboardConversationRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.ConversationGetBy)
	group.POST("/add_tag", dashboard.ConversationPostAdd_tag)
	group.POST("/assign", dashboard.ConversationPostAssign)
	group.POST("/close", dashboard.ConversationPostClose)
	group.Any("/conversations", dashboard.ConversationAnyConversations)
	group.POST("/dispatch", dashboard.ConversationPostDispatch)
	group.POST("/link_customer", dashboard.ConversationPostLink_customer)
	group.Any("/list", dashboard.ConversationAnyList)
	group.Any("/message_list", dashboard.ConversationAnyMessage_list)
	group.POST("/read", dashboard.ConversationPostRead)
	group.POST("/recall_message", dashboard.ConversationPostRecall_message)
	group.POST("/remove_tag", dashboard.ConversationPostRemove_tag)
	group.POST("/send_message", dashboard.ConversationPostSend_message)
	group.POST("/transfer", dashboard.ConversationPostTransfer)
	group.POST("/upload_attachment", dashboard.ConversationPostUpload_attachment)
	group.POST("/upload_image", dashboard.ConversationPostUpload_image)
}

func registerDashboardTicketRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.TicketGetBy)
	group.POST("/assign", dashboard.TicketPostAssign)
	group.POST("/change_status", dashboard.TicketPostChange_status)
	group.POST("/create", dashboard.TicketPostCreate)
	group.POST("/create_from_conversation", dashboard.TicketPostCreate_from_conversation)
	group.POST("/delete_view", dashboard.TicketPostDelete_view)
	group.POST("/link_customer", dashboard.TicketPostLink_customer)
	group.Any("/list", dashboard.TicketAnyList)
	group.POST("/progress/create", dashboard.TicketPostProgressCreate)
	group.Any("/progress/list", dashboard.TicketAnyProgressList)
	group.POST("/save_view", dashboard.TicketPostSave_view)
	group.Any("/summary", dashboard.TicketAnySummary)
	group.POST("/update", dashboard.TicketPostUpdate)
	group.Any("/view_list", dashboard.TicketAnyView_list)
}

func registerDashboardNotificationRoutes(group *gin.RouterGroup) {
	group.Any("/list", dashboard.NotificationAnyList)
	group.POST("/mark_all_read", dashboard.NotificationPostMark_all_read)
	group.POST("/mark_read", dashboard.NotificationPostMark_read)
	group.GET("/unread_count", dashboard.NotificationGetUnread_count)
}

func registerDashboardQuickReplyRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.QuickReplyGetBy)
	group.POST("/create", dashboard.QuickReplyPostCreate)
	group.POST("/delete", dashboard.QuickReplyPostDelete)
	group.Any("/list", dashboard.QuickReplyAnyList)
	group.GET("/list_all", dashboard.QuickReplyGetList_all)
	group.POST("/update", dashboard.QuickReplyPostUpdate)
}

func registerDashboardChannelRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.ChannelGetBy)
	group.POST("/create", dashboard.ChannelPostCreate)
	group.POST("/delete", dashboard.ChannelPostDelete)
	group.Any("/list", dashboard.ChannelAnyList)
	group.POST("/reset_user_token_secret", dashboard.ChannelPostReset_user_token_secret)
	group.POST("/rollback_ai_agent_rollout", dashboard.ChannelPostRollback_ai_agent_rollout)
	group.POST("/update", dashboard.ChannelPostUpdate)
	group.POST("/update_status", dashboard.ChannelPostUpdate_status)
	group.Any("/wxwork/kf/accounts", dashboard.ChannelAnyWxworkKfAccounts)
	group.Any("/wxwork/outbox/failed/list", dashboard.ChannelAnyWxworkOutboxFailedList)
	group.POST("/wxwork/outbox/retry", dashboard.ChannelPostWxworkOutboxRetry)
	group.POST("/wxwork/outbox/ignore", dashboard.ChannelPostWxworkOutboxIgnore)
}

func registerDashboardAgentRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.AgentGetBy)
	group.POST("/create", dashboard.AgentPostCreate)
	group.POST("/delete", dashboard.AgentPostDelete)
	group.Any("/list", dashboard.AgentAnyList)
	group.GET("/list_all", dashboard.AgentGetList_all)
	group.POST("/update", dashboard.AgentPostUpdate)
}

func registerDashboardAgentTeamRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.AgentTeamGetBy)
	group.POST("/create", dashboard.AgentTeamPostCreate)
	group.POST("/delete", dashboard.AgentTeamPostDelete)
	group.Any("/list", dashboard.AgentTeamAnyList)
	group.GET("/list_all", dashboard.AgentTeamGetList_all)
	group.POST("/update", dashboard.AgentTeamPostUpdate)
}

func registerDashboardAgentTeamScheduleRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.AgentTeamScheduleGetBy)
	group.POST("/batch_generate", dashboard.AgentTeamSchedulePostBatch_generate)
	group.POST("/batch_preview", dashboard.AgentTeamSchedulePostBatch_preview)
	group.Any("/calendar", dashboard.AgentTeamScheduleAnyCalendar)
	group.POST("/create", dashboard.AgentTeamSchedulePostCreate)
	group.POST("/delete", dashboard.AgentTeamSchedulePostDelete)
	group.Any("/list", dashboard.AgentTeamScheduleAnyList)
	group.POST("/update", dashboard.AgentTeamSchedulePostUpdate)
}

func registerDashboardAIAgentRoutes(group *gin.RouterGroup) {
	group.POST("/publish", dashboard.AIAgentPostPublish)
	group.POST("/rollback", dashboard.AIAgentPostRollback)
	group.POST("/rollback_rollout", dashboard.AIAgentPostRollback_rollout)
	group.Any("/:id/revision/list", dashboard.AIAgentAnyRevisionList)
	group.GET("/:id", dashboard.AIAgentGetBy)
	group.POST("/create", dashboard.AIAgentPostCreate)
	group.POST("/delete", dashboard.AIAgentPostDelete)
	group.Any("/list", dashboard.AIAgentAnyList)
	group.GET("/list_all", dashboard.AIAgentGetList_all)
	group.POST("/update", dashboard.AIAgentPostUpdate)
	group.POST("/update_sort", dashboard.AIAgentPostUpdate_sort)
	group.POST("/update_status", dashboard.AIAgentPostUpdate_status)
}

func registerDashboardAIWorkflowRoutes(group *gin.RouterGroup) {
	group.POST("/create", dashboard.AIWorkflowPostCreate)
	group.POST("/update", dashboard.AIWorkflowPostUpdate)
	group.POST("/delete", dashboard.AIWorkflowPostDelete)
	group.POST("/restore-version", dashboard.AIWorkflowPostRestoreVersion)
	group.Any("/list", dashboard.AIWorkflowAnyList)
	group.GET("/node-spec/list", dashboard.AIWorkflowGetNodeSpecList)
	group.GET("/default-definition", dashboard.AIWorkflowGetDefaultDefinition)
	group.GET("/template/list", dashboard.AIWorkflowGetTemplateList)
	group.POST("/validate", dashboard.AIWorkflowPostValidate)
	group.POST("/publish", dashboard.AIWorkflowPostPublish)
	group.Any("/run/list", dashboard.AIWorkflowAnyRunList)
	group.GET("/run/:id", dashboard.AIWorkflowGetRunBy)
	group.Any("/version/list", dashboard.AIWorkflowAnyVersionList)
	group.GET("/version/:id", dashboard.AIWorkflowGetVersionBy)
	group.GET("/:id/usage", dashboard.AIWorkflowGetUsage)
	group.GET("/:id", dashboard.AIWorkflowGetBy)
}

func registerDashboardAgentRunRoutes(group *gin.RouterGroup) {
	group.Any("/metrics", dashboard.AgentRunAnyMetrics)
	group.POST("/evaluate", dashboard.AgentRunPostEvaluate)
	group.Any("/list", dashboard.AgentRunAnyList)
	group.POST("/quality_feedback", dashboard.AgentRunPostSave_quality_feedback)
	group.GET("/:id", dashboard.AgentRunGetBy)
}

func registerDashboardAIConfigRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.AIConfigGetBy)
	group.POST("/create", dashboard.AIConfigPostCreate)
	group.POST("/delete", dashboard.AIConfigPostDelete)
	group.Any("/list", dashboard.AIConfigAnyList)
	group.Any("/list_all", dashboard.AIConfigAnyList_all)
	group.POST("/update", dashboard.AIConfigPostUpdate)
	group.POST("/update_sort", dashboard.AIConfigPostUpdateSort)
	group.POST("/update_status", dashboard.AIConfigPostUpdate_status)
}

func registerDashboardAssetRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.AssetGetBy)
	group.POST("/create", dashboard.AssetPostCreate)
	group.POST("/delete", dashboard.AssetPostDelete)
	group.Any("/list", dashboard.AssetAnyList)
}

func registerDashboardKnowledgeBaseRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.KnowledgeBaseGetBy)
	group.POST("/create", dashboard.KnowledgeBasePostCreate)
	group.POST("/delete", dashboard.KnowledgeBasePostDelete)
	group.Any("/list", dashboard.KnowledgeBaseAnyList)
	group.Any("/list_all", dashboard.KnowledgeBaseAnyList_all)
	group.POST("/rebuild_index", dashboard.KnowledgeBasePostRebuild_index)
	group.POST("/update", dashboard.KnowledgeBasePostUpdate)
	group.POST("/update_sort", dashboard.KnowledgeBasePostUpdate_sort)
}

func registerDashboardKnowledgeDirectoryRoutes(group *gin.RouterGroup) {
	group.GET("/list_all", dashboard.KnowledgeDirectoryGetList_all)
	group.POST("/create", dashboard.KnowledgeDirectoryPostCreate)
	group.POST("/delete", dashboard.KnowledgeDirectoryPostDelete)
	group.POST("/update", dashboard.KnowledgeDirectoryPostUpdate)
	group.POST("/update_sort", dashboard.KnowledgeDirectoryPostUpdate_sort)
}

func registerDashboardKnowledgeDocumentRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.KnowledgeDocumentGetBy)
	group.POST("/batch_delete", dashboard.KnowledgeDocumentPostBatch_delete)
	group.POST("/batch_move", dashboard.KnowledgeDocumentPostBatch_move)
	group.POST("/create", dashboard.KnowledgeDocumentPostCreate)
	group.POST("/delete", dashboard.KnowledgeDocumentPostDelete)
	group.Any("/list", dashboard.KnowledgeDocumentAnyList)
	group.POST("/update", dashboard.KnowledgeDocumentPostUpdate)
}

func registerDashboardKnowledgeFAQRoutes(group *gin.RouterGroup) {
	group.GET("/import_template", dashboard.KnowledgeFAQGetImport_template)
	group.GET("/export", dashboard.KnowledgeFAQGetExport)
	group.GET("/:id", dashboard.KnowledgeFAQGetBy)
	group.POST("/batch_delete", dashboard.KnowledgeFAQPostBatch_delete)
	group.POST("/batch_move", dashboard.KnowledgeFAQPostBatch_move)
	group.POST("/create", dashboard.KnowledgeFAQPostCreate)
	group.POST("/delete", dashboard.KnowledgeFAQPostDelete)
	group.POST("/import", dashboard.KnowledgeFAQPostImport)
	group.Any("/list", dashboard.KnowledgeFAQAnyList)
	group.POST("/update", dashboard.KnowledgeFAQPostUpdate)
}

func registerDashboardKnowledgeRetrieveRoutes(group *gin.RouterGroup) {
	group.POST("/build", dashboard.KnowledgeRetrievePostBuild)
	group.POST("/debug/answer", dashboard.KnowledgeRetrievePostDebugAnswer)
	group.POST("/debug/search", dashboard.KnowledgeRetrievePostDebugSearch)
}

func registerDashboardKnowledgeRetrieveLogRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.KnowledgeRetrieveLogGetBy)
	group.Any("/list", dashboard.KnowledgeRetrieveLogAnyList)
}

func registerDashboardDocPageRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.DocPageGetBy)
	group.POST("/create", dashboard.DocPagePostCreate)
	group.POST("/delete", dashboard.DocPagePostDelete)
	group.Any("/list", dashboard.DocPageAnyList)
	group.GET("/list_all", dashboard.DocPageGetList_all)
	group.POST("/update", dashboard.DocPagePostUpdate)
	group.POST("/update_settings", dashboard.DocPagePostUpdate_settings)
	group.POST("/update_sort", dashboard.DocPagePostUpdate_sort)
	group.POST("/change_status", dashboard.DocPagePostChange_status)
}

func registerDashboardCommunityCategoryRoutes(group *gin.RouterGroup) {
	group.POST("/create", dashboard.CategoryPostCreate)
	group.POST("/delete", dashboard.CategoryPostDelete)
	group.Any("/list", dashboard.CategoryAnyList)
	group.GET("/list_all", dashboard.CategoryGetList_all)
	group.POST("/update", dashboard.CategoryPostUpdate)
	group.POST("/update_sort", dashboard.CategoryPostUpdateSort)
}

func registerDashboardCommunityPostRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.PostGetBy)
	group.POST("/accept_comment", dashboard.PostPostAcceptComment)
	group.POST("/comment/create", dashboard.CommentPostCreate)
	group.POST("/comment/moderate", dashboard.CommentPostModerate)
	group.Any("/list", dashboard.PostAnyList)
	group.POST("/moderate", dashboard.PostPostModerate)
}

func registerDashboardSkillDefinitionRoutes(group *gin.RouterGroup) {
	group.GET("/:id", dashboard.SkillDefinitionGetBy)
	group.POST("/create", dashboard.SkillDefinitionPostCreate)
	group.POST("/debug_resume", dashboard.SkillDefinitionPostDebug_resume)
	group.POST("/debug_run", dashboard.SkillDefinitionPostDebug_run)
	group.POST("/delete", dashboard.SkillDefinitionPostDelete)
	group.Any("/list", dashboard.SkillDefinitionAnyList)
	group.GET("/list_all", dashboard.SkillDefinitionGetList_all)
	group.POST("/restore", dashboard.SkillDefinitionPostRestore)
	group.POST("/update", dashboard.SkillDefinitionPostUpdate)
	group.POST("/update_status", dashboard.SkillDefinitionPostUpdate_status)
}

func registerDashboardMCPRoutes(group *gin.RouterGroup) {
	group.POST("/call_tool", dashboard.MCPPostCall_tool)
	group.Any("/catalog", dashboard.MCPAnyCatalog)
	group.Any("/list_servers", dashboard.MCPAnyList_servers)
	group.POST("/list_tools", dashboard.MCPPostList_tools)
	group.POST("/test_connection", dashboard.MCPPostTest_connection)
}

func registerThirdWechatRoutes(group *gin.RouterGroup) {
	group.GET("/callback", third.WechatGetCallback)
	group.POST("/callback", third.WechatPostCallback)
}

func registerThirdTelegramRoutes(group *gin.RouterGroup) {
	group.POST("/webhook", third.TelegramPostWebhook)
	group.POST("/webhook/:channel_id", third.TelegramPostWebhook)
}

func registerThirdZaloRoutes(group *gin.RouterGroup) {
	group.POST("/webhook", third.ZaloPostWebhook)
	group.POST("/webhook/:channel_id", third.ZaloPostWebhook)
}
