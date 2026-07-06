package constants

import (
	"os"
	"strings"
)

const (
	RoleCodeSuperAdmin   = "super_admin"    // 超管
	RoleCodeAdmin        = "admin"          // 管理员
	RoleCodeCsTeamLeader = "cs_team_leader" // 客服组长
	RoleCodeCsUser       = "cs_user"        // 客服
)

const (
	AuthTokenPrefix = "ak_"
)

const (
	ClientTypeAdminWeb = "admin_web"
)

const (
	BootstrapAdminUsername = "admin"
	BootstrapAdminPassword = "ChangeMe123!"
	BootstrapAdminNickname = "Super Admin"
)

const (
	EnvBootstrapAdminPassword = "AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD"
)

func BootstrapAdminInitialPassword() string {
	if password := strings.TrimSpace(os.Getenv(EnvBootstrapAdminPassword)); password != "" {
		return password
	}
	return BootstrapAdminPassword
}

// Permission 权限结构体
type Permission struct {
	Name      string
	Code      string
	Type      string
	GroupName string
	Method    string
	APIPath   string
	SortNo    int
}

// 权限常量定义
var (
	// 用户相关权限
	PermissionUserView       = Permission{Name: "查看用户", Code: "user.view", Type: "api", GroupName: "user", Method: "ANY", APIPath: "/api/dashboard/user/list", SortNo: 10}
	PermissionUserCreate     = Permission{Name: "创建用户", Code: "user.create", Type: "api", GroupName: "user", Method: "POST", APIPath: "/api/dashboard/user/create", SortNo: 20}
	PermissionUserUpdate     = Permission{Name: "更新用户", Code: "user.update", Type: "api", GroupName: "user", Method: "POST", APIPath: "/api/dashboard/user/update", SortNo: 30}
	PermissionUserDelete     = Permission{Name: "删除用户", Code: "user.delete", Type: "api", GroupName: "user", Method: "POST", APIPath: "/api/dashboard/user/delete", SortNo: 40}
	PermissionUserAssignRole = Permission{Name: "分配用户角色", Code: "user.assignRole", Type: "api", GroupName: "user", Method: "POST", APIPath: "/api/dashboard/user/assign_role", SortNo: 50}

	// 角色相关权限
	PermissionRoleView             = Permission{Name: "查看角色", Code: "role.view", Type: "api", GroupName: "role", Method: "ANY", APIPath: "/api/dashboard/role/list", SortNo: 110}
	PermissionRoleCreate           = Permission{Name: "创建角色", Code: "role.create", Type: "api", GroupName: "role", Method: "POST", APIPath: "/api/dashboard/role/create", SortNo: 120}
	PermissionRoleUpdate           = Permission{Name: "更新角色", Code: "role.update", Type: "api", GroupName: "role", Method: "POST", APIPath: "/api/dashboard/role/update", SortNo: 130}
	PermissionRoleDelete           = Permission{Name: "删除角色", Code: "role.delete", Type: "api", GroupName: "role", Method: "POST", APIPath: "/api/dashboard/role/delete", SortNo: 140}
	PermissionRoleAssignPermission = Permission{Name: "分配角色权限", Code: "role.assignPermission", Type: "api", GroupName: "role", Method: "POST", APIPath: "/api/dashboard/role/assign_permission", SortNo: 150}

	// 权限相关权限
	PermissionPermissionView = Permission{Name: "查看权限", Code: "permission.view", Type: "api", GroupName: "permission", Method: "ANY", APIPath: "/api/dashboard/permission/list", SortNo: 210}
	PermissionPermissionSync = Permission{Name: "同步权限", Code: "permission.sync", Type: "api", GroupName: "permission", Method: "POST", APIPath: "/api/dashboard/permission/sync", SortNo: 220}

	// 会话相关权限
	PermissionSessionView   = Permission{Name: "查看会话", Code: "session.view", Type: "api", GroupName: "session", Method: "ANY", APIPath: "/api/dashboard/session/list", SortNo: 310}
	PermissionSessionRevoke = Permission{Name: "踢除会话", Code: "session.revoke", Type: "api", GroupName: "session", Method: "POST", APIPath: "/api/dashboard/session/revoke", SortNo: 320}

	// 客服会话相关权限
	PermissionConversationView         = Permission{Name: "查看会话", Code: "conversation.view", Type: "api", GroupName: "conversation", Method: "ANY", APIPath: "/api/dashboard/conversation/list", SortNo: 410}
	PermissionConversationAssign       = Permission{Name: "分配会话", Code: "conversation.assign", Type: "api", GroupName: "conversation", Method: "POST", APIPath: "/api/dashboard/conversation/assign", SortNo: 430}
	PermissionConversationTransfer     = Permission{Name: "转接会话", Code: "conversation.transfer", Type: "api", GroupName: "conversation", Method: "POST", APIPath: "/api/dashboard/conversation/transfer", SortNo: 440}
	PermissionConversationClose        = Permission{Name: "关闭会话", Code: "conversation.close", Type: "api", GroupName: "conversation", Method: "POST", APIPath: "/api/dashboard/conversation/close", SortNo: 450}
	PermissionConversationSend         = Permission{Name: "发送会话消息", Code: "conversation.send", Type: "api", GroupName: "conversation", Method: "POST", APIPath: "/api/dashboard/conversation/send_message", SortNo: 460}
	PermissionConversationTag          = Permission{Name: "管理会话标签", Code: "conversation.tag", Type: "api", GroupName: "conversation", Method: "POST", APIPath: "/api/dashboard/conversation/add_tag", SortNo: 470}
	PermissionConversationHandover     = Permission{Name: "处理会话交接", Code: "conversation.handover", Type: "api", GroupName: "conversation", Method: "ANY", APIPath: "/api/dashboard/conversation/handover_list", SortNo: 480}
	PermissionConversationRecycle      = Permission{Name: "回收会话", Code: "conversation.recycle", Type: "api", GroupName: "conversation", Method: "POST", APIPath: "/api/dashboard/conversation/recycle", SortNo: 490}
	PermissionConversationLinkCustomer = Permission{Name: "关联会话客户", Code: "conversation.linkCustomer", Type: "api", GroupName: "conversation", Method: "POST", APIPath: "/api/dashboard/conversation/link_customer", SortNo: 495}

	// 工单相关权限
	PermissionTicketView         = Permission{Name: "查看工单", Code: "ticket.view", Type: "api", GroupName: "ticket", Method: "ANY", APIPath: "/api/dashboard/ticket/list", SortNo: 500}
	PermissionTicketCreate       = Permission{Name: "创建工单", Code: "ticket.create", Type: "api", GroupName: "ticket", Method: "POST", APIPath: "/api/dashboard/ticket/create", SortNo: 510}
	PermissionTicketUpdate       = Permission{Name: "更新工单", Code: "ticket.update", Type: "api", GroupName: "ticket", Method: "POST", APIPath: "/api/dashboard/ticket/update", SortNo: 520}
	PermissionTicketAssign       = Permission{Name: "指派工单", Code: "ticket.assign", Type: "api", GroupName: "ticket", Method: "POST", APIPath: "/api/dashboard/ticket/assign", SortNo: 530}
	PermissionTicketChangeStatus = Permission{Name: "变更工单状态", Code: "ticket.changeStatus", Type: "api", GroupName: "ticket", Method: "POST", APIPath: "/api/dashboard/ticket/change_status", SortNo: 540}
	PermissionTicketProgress     = Permission{Name: "更新工单进展", Code: "ticket.progress", Type: "api", GroupName: "ticket", Method: "POST", APIPath: "/api/dashboard/ticket/progress/create", SortNo: 550}

	// 通知相关权限
	PermissionNotificationView   = Permission{Name: "查看通知", Code: "notification.view", Type: "api", GroupName: "notification", Method: "ANY", APIPath: "/api/dashboard/notification/list", SortNo: 680}
	PermissionNotificationUpdate = Permission{Name: "更新通知", Code: "notification.update", Type: "api", GroupName: "notification", Method: "POST", APIPath: "/api/dashboard/notification/mark_read", SortNo: 690}

	// 快捷回复相关权限
	PermissionQuickReplyView   = Permission{Name: "查看快捷回复", Code: "quickReply.view", Type: "api", GroupName: "quickReply", Method: "ANY", APIPath: "/api/dashboard/quick-reply/list", SortNo: 610}
	PermissionQuickReplyCreate = Permission{Name: "创建快捷回复", Code: "quickReply.create", Type: "api", GroupName: "quickReply", Method: "POST", APIPath: "/api/dashboard/quick-reply/create", SortNo: 620}
	PermissionQuickReplyUpdate = Permission{Name: "更新快捷回复", Code: "quickReply.update", Type: "api", GroupName: "quickReply", Method: "POST", APIPath: "/api/dashboard/quick-reply/update", SortNo: 630}
	PermissionQuickReplyDelete = Permission{Name: "删除快捷回复", Code: "quickReply.delete", Type: "api", GroupName: "quickReply", Method: "POST", APIPath: "/api/dashboard/quick-reply/delete", SortNo: 640}

	// 标签相关权限
	PermissionTagView   = Permission{Name: "查看标签", Code: "tag.view", Type: "api", GroupName: "tag", Method: "ANY", APIPath: "/api/dashboard/tag/list", SortNo: 550}
	PermissionTagCreate = Permission{Name: "创建标签", Code: "tag.create", Type: "api", GroupName: "tag", Method: "POST", APIPath: "/api/dashboard/tag/create", SortNo: 560}
	PermissionTagUpdate = Permission{Name: "更新标签", Code: "tag.update", Type: "api", GroupName: "tag", Method: "POST", APIPath: "/api/dashboard/tag/update", SortNo: 570}
	PermissionTagDelete = Permission{Name: "删除标签", Code: "tag.delete", Type: "api", GroupName: "tag", Method: "POST", APIPath: "/api/dashboard/tag/delete", SortNo: 580}

	// 公司相关权限
	PermissionCompanyView   = Permission{Name: "查看公司", Code: "company.view", Type: "api", GroupName: "company", Method: "ANY", APIPath: "/api/dashboard/company/list", SortNo: 590}
	PermissionCompanyCreate = Permission{Name: "创建公司", Code: "company.create", Type: "api", GroupName: "company", Method: "POST", APIPath: "/api/dashboard/company/create", SortNo: 600}
	PermissionCompanyUpdate = Permission{Name: "更新公司", Code: "company.update", Type: "api", GroupName: "company", Method: "POST", APIPath: "/api/dashboard/company/update", SortNo: 610}
	PermissionCompanyDelete = Permission{Name: "删除公司", Code: "company.delete", Type: "api", GroupName: "company", Method: "POST", APIPath: "/api/dashboard/company/delete", SortNo: 620}

	// 接入渠道相关权限
	PermissionChannelView   = Permission{Name: "查看接入渠道", Code: "channel.view", Type: "api", GroupName: "channel", Method: "ANY", APIPath: "/api/dashboard/channel/list", SortNo: 625}
	PermissionChannelCreate = Permission{Name: "创建接入渠道", Code: "channel.create", Type: "api", GroupName: "channel", Method: "POST", APIPath: "/api/dashboard/channel/create", SortNo: 626}
	PermissionChannelUpdate = Permission{Name: "更新接入渠道", Code: "channel.update", Type: "api", GroupName: "channel", Method: "POST", APIPath: "/api/dashboard/channel/update", SortNo: 627}
	PermissionChannelDelete = Permission{Name: "删除接入渠道", Code: "channel.delete", Type: "api", GroupName: "channel", Method: "POST", APIPath: "/api/dashboard/channel/delete", SortNo: 628}

	// 客户相关权限
	PermissionCustomerView   = Permission{Name: "查看客户", Code: "customer.view", Type: "api", GroupName: "customer", Method: "POST", APIPath: "/api/dashboard/customer/list", SortNo: 630}
	PermissionCustomerCreate = Permission{Name: "创建客户", Code: "customer.create", Type: "api", GroupName: "customer", Method: "POST", APIPath: "/api/dashboard/customer/create", SortNo: 640}
	PermissionCustomerUpdate = Permission{Name: "更新客户", Code: "customer.update", Type: "api", GroupName: "customer", Method: "POST", APIPath: "/api/dashboard/customer/update", SortNo: 650}
	PermissionCustomerDelete = Permission{Name: "删除客户", Code: "customer.delete", Type: "api", GroupName: "customer", Method: "POST", APIPath: "/api/dashboard/customer/delete", SortNo: 660}

	// 销售线索相关权限
	PermissionSalesLeadView     = Permission{Name: "查看销售线索", Code: "salesLead.view", Type: "api", GroupName: "salesLead", Method: "POST", APIPath: "/api/dashboard/sales-lead/list", SortNo: 665}
	PermissionSalesLeadUpdate   = Permission{Name: "更新销售线索", Code: "salesLead.update", Type: "api", GroupName: "salesLead", Method: "POST", APIPath: "/api/dashboard/sales-lead/update", SortNo: 666}
	PermissionSalesLeadAssign   = Permission{Name: "分配销售线索", Code: "salesLead.assign", Type: "api", GroupName: "salesLead", Method: "POST", APIPath: "/api/dashboard/sales-lead/assign", SortNo: 667}
	PermissionSalesLeadFollowUp = Permission{Name: "跟进销售线索", Code: "salesLead.followUp", Type: "api", GroupName: "salesLead", Method: "POST", APIPath: "/api/dashboard/sales-lead/follow-up/create", SortNo: 668}

	// 产品库相关权限
	PermissionProductView    = Permission{Name: "查看产品库", Code: "product.view", Type: "api", GroupName: "product", Method: "POST", APIPath: "/api/dashboard/product/list", SortNo: 669}
	PermissionProductCreate  = Permission{Name: "创建产品", Code: "product.create", Type: "api", GroupName: "product", Method: "POST", APIPath: "/api/dashboard/product/create", SortNo: 670}
	PermissionProductUpdate  = Permission{Name: "更新产品", Code: "product.update", Type: "api", GroupName: "product", Method: "POST", APIPath: "/api/dashboard/product/update", SortNo: 671}
	PermissionProductDelete  = Permission{Name: "删除产品", Code: "product.delete", Type: "api", GroupName: "product", Method: "POST", APIPath: "/api/dashboard/product/delete", SortNo: 672}
	PermissionProductReindex = Permission{Name: "重建产品索引", Code: "product.reindex", Type: "api", GroupName: "product", Method: "POST", APIPath: "/api/dashboard/product/reindex", SortNo: 673}

	// 活动/优惠库相关权限
	PermissionPromotionView    = Permission{Name: "查看活动库", Code: "promotion.view", Type: "api", GroupName: "promotion", Method: "POST", APIPath: "/api/dashboard/promotion/list", SortNo: 674}
	PermissionPromotionCreate  = Permission{Name: "创建活动", Code: "promotion.create", Type: "api", GroupName: "promotion", Method: "POST", APIPath: "/api/dashboard/promotion/create", SortNo: 675}
	PermissionPromotionUpdate  = Permission{Name: "更新活动", Code: "promotion.update", Type: "api", GroupName: "promotion", Method: "POST", APIPath: "/api/dashboard/promotion/update", SortNo: 676}
	PermissionPromotionDelete  = Permission{Name: "删除活动", Code: "promotion.delete", Type: "api", GroupName: "promotion", Method: "POST", APIPath: "/api/dashboard/promotion/delete", SortNo: 677}
	PermissionPromotionReindex = Permission{Name: "重建活动索引", Code: "promotion.reindex", Type: "api", GroupName: "promotion", Method: "POST", APIPath: "/api/dashboard/promotion/reindex", SortNo: 678}

	// AI 数字店长配置权限
	PermissionDigitalStoreView   = Permission{Name: "查看数字店长配置", Code: "digitalStore.view", Type: "api", GroupName: "digitalStore", Method: "GET", APIPath: "/api/dashboard/digital-store/profile", SortNo: 679}
	PermissionDigitalStoreUpdate = Permission{Name: "更新数字店长配置", Code: "digitalStore.update", Type: "api", GroupName: "digitalStore", Method: "POST", APIPath: "/api/dashboard/digital-store/profile", SortNo: 680}

	// 客服相关权限
	PermissionAgentView         = Permission{Name: "查看客服", Code: "agent.view", Type: "api", GroupName: "agent", Method: "ANY", APIPath: "/api/dashboard/agent/list", SortNo: 610}
	PermissionAgentCreate       = Permission{Name: "创建客服", Code: "agent.create", Type: "api", GroupName: "agent", Method: "POST", APIPath: "/api/dashboard/agent/create", SortNo: 620}
	PermissionAgentUpdate       = Permission{Name: "更新客服", Code: "agent.update", Type: "api", GroupName: "agent", Method: "POST", APIPath: "/api/dashboard/agent/update", SortNo: 630}
	PermissionAgentDelete       = Permission{Name: "删除客服", Code: "agent.delete", Type: "api", GroupName: "agent", Method: "POST", APIPath: "/api/dashboard/agent/delete", SortNo: 640}
	PermissionAgentUpdateStatus = Permission{Name: "更新客服状态", Code: "agent.updateStatus", Type: "api", GroupName: "agent", Method: "POST", APIPath: "/api/dashboard/agent/update_status", SortNo: 650}
	PermissionAgentConfig       = Permission{Name: "配置客服服务规则", Code: "agent.config", Type: "api", GroupName: "agent", Method: "POST", APIPath: "/api/dashboard/agent/update_service_config", SortNo: 660}

	// 客服组相关权限
	PermissionAgentTeamView   = Permission{Name: "查看客服组", Code: "agentTeam.view", Type: "api", GroupName: "agentTeam", Method: "ANY", APIPath: "/api/dashboard/agent-team/list", SortNo: 710}
	PermissionAgentTeamCreate = Permission{Name: "创建客服组", Code: "agentTeam.create", Type: "api", GroupName: "agentTeam", Method: "POST", APIPath: "/api/dashboard/agent-team/create", SortNo: 720}
	PermissionAgentTeamUpdate = Permission{Name: "更新客服组", Code: "agentTeam.update", Type: "api", GroupName: "agentTeam", Method: "POST", APIPath: "/api/dashboard/agent-team/update", SortNo: 730}
	PermissionAgentTeamDelete = Permission{Name: "删除客服组", Code: "agentTeam.delete", Type: "api", GroupName: "agentTeam", Method: "POST", APIPath: "/api/dashboard/agent-team/delete", SortNo: 740}

	// 客服组排班相关权限
	PermissionAgentTeamScheduleView          = Permission{Name: "查看客服组排班", Code: "agentTeamSchedule.view", Type: "api", GroupName: "agentTeamSchedule", Method: "ANY", APIPath: "/api/dashboard/agent-team-schedule/list", SortNo: 810}
	PermissionAgentTeamScheduleCreate        = Permission{Name: "创建客服组排班", Code: "agentTeamSchedule.create", Type: "api", GroupName: "agentTeamSchedule", Method: "POST", APIPath: "/api/dashboard/agent-team-schedule/create", SortNo: 820}
	PermissionAgentTeamScheduleUpdate        = Permission{Name: "更新客服组排班", Code: "agentTeamSchedule.update", Type: "api", GroupName: "agentTeamSchedule", Method: "POST", APIPath: "/api/dashboard/agent-team-schedule/update", SortNo: 830}
	PermissionAgentTeamScheduleDelete        = Permission{Name: "删除客服组排班", Code: "agentTeamSchedule.delete", Type: "api", GroupName: "agentTeamSchedule", Method: "POST", APIPath: "/api/dashboard/agent-team-schedule/delete", SortNo: 840}
	PermissionAgentTeamScheduleBatchGenerate = Permission{Name: "批量生成客服组排班", Code: "agentTeamSchedule.batchGenerate", Type: "api", GroupName: "agentTeamSchedule", Method: "POST", APIPath: "/api/dashboard/agent-team-schedule/batch_generate", SortNo: 850}

	// 文件资源相关权限
	PermissionAssetView   = Permission{Name: "查看文件资源", Code: "asset.view", Type: "api", GroupName: "asset", Method: "ANY", APIPath: "/api/dashboard/asset/list", SortNo: 1210}
	PermissionAssetCreate = Permission{Name: "上传文件资源", Code: "asset.create", Type: "api", GroupName: "asset", Method: "POST", APIPath: "/api/dashboard/asset/create", SortNo: 1220}
	PermissionAssetDelete = Permission{Name: "删除文件资源", Code: "asset.delete", Type: "api", GroupName: "asset", Method: "POST", APIPath: "/api/dashboard/asset/delete", SortNo: 1230}

	// AI Agent 相关权限
	PermissionAIAgentView   = Permission{Name: "查看 AI Agent", Code: "aiAgent.view", Type: "api", GroupName: "aiAgent", Method: "ANY", APIPath: "/api/dashboard/ai-agent/list", SortNo: 1310}
	PermissionAIAgentCreate = Permission{Name: "创建 AI Agent", Code: "aiAgent.create", Type: "api", GroupName: "aiAgent", Method: "POST", APIPath: "/api/dashboard/ai-agent/create", SortNo: 1320}
	PermissionAIAgentUpdate = Permission{Name: "更新 AI Agent", Code: "aiAgent.update", Type: "api", GroupName: "aiAgent", Method: "POST", APIPath: "/api/dashboard/ai-agent/update", SortNo: 1330}
	PermissionAIAgentDelete = Permission{Name: "删除 AI Agent", Code: "aiAgent.delete", Type: "api", GroupName: "aiAgent", Method: "POST", APIPath: "/api/dashboard/ai-agent/delete", SortNo: 1340}

	// AI 配置相关权限
	PermissionAIConfigView   = Permission{Name: "查看 AI 配置", Code: "aiConfig.view", Type: "api", GroupName: "aiConfig", Method: "ANY", APIPath: "/api/dashboard/ai-config/list", SortNo: 1390}
	PermissionAIConfigCreate = Permission{Name: "创建 AI 配置", Code: "aiConfig.create", Type: "api", GroupName: "aiConfig", Method: "POST", APIPath: "/api/dashboard/ai-config/create", SortNo: 1400}
	PermissionAIConfigUpdate = Permission{Name: "更新 AI 配置", Code: "aiConfig.update", Type: "api", GroupName: "aiConfig", Method: "POST", APIPath: "/api/dashboard/ai-config/update", SortNo: 1410}
	PermissionAIConfigDelete = Permission{Name: "删除 AI 配置", Code: "aiConfig.delete", Type: "api", GroupName: "aiConfig", Method: "POST", APIPath: "/api/dashboard/ai-config/delete", SortNo: 1420}

	// 知识库相关权限
	PermissionKnowledgeBaseView   = Permission{Name: "查看知识库", Code: "knowledgeBase.view", Type: "api", GroupName: "knowledgeBase", Method: "ANY", APIPath: "/api/dashboard/knowledge-base/list", SortNo: 1410}
	PermissionKnowledgeBaseCreate = Permission{Name: "创建知识库", Code: "knowledgeBase.create", Type: "api", GroupName: "knowledgeBase", Method: "POST", APIPath: "/api/dashboard/knowledge-base/create", SortNo: 1420}
	PermissionKnowledgeBaseUpdate = Permission{Name: "更新知识库", Code: "knowledgeBase.update", Type: "api", GroupName: "knowledgeBase", Method: "POST", APIPath: "/api/dashboard/knowledge-base/update", SortNo: 1430}
	PermissionKnowledgeBaseDelete = Permission{Name: "删除知识库", Code: "knowledgeBase.delete", Type: "api", GroupName: "knowledgeBase", Method: "POST", APIPath: "/api/dashboard/knowledge-base/delete", SortNo: 1440}

	// 知识文档相关权限
	PermissionKnowledgeDocumentView   = Permission{Name: "查看知识文档", Code: "knowledgeDocument.view", Type: "api", GroupName: "knowledgeDocument", Method: "ANY", APIPath: "/api/dashboard/knowledge-document/list", SortNo: 1510}
	PermissionKnowledgeDocumentCreate = Permission{Name: "创建知识文档", Code: "knowledgeDocument.create", Type: "api", GroupName: "knowledgeDocument", Method: "POST", APIPath: "/api/dashboard/knowledge-document/create", SortNo: 1520}
	PermissionKnowledgeDocumentUpdate = Permission{Name: "更新知识文档", Code: "knowledgeDocument.update", Type: "api", GroupName: "knowledgeDocument", Method: "POST", APIPath: "/api/dashboard/knowledge-document/update", SortNo: 1530}
	PermissionKnowledgeDocumentDelete = Permission{Name: "删除知识文档", Code: "knowledgeDocument.delete", Type: "api", GroupName: "knowledgeDocument", Method: "POST", APIPath: "/api/dashboard/knowledge-document/delete", SortNo: 1540}
	PermissionKnowledgeFAQView        = Permission{Name: "查看知识FAQ", Code: "knowledgeFAQ.view", Type: "api", GroupName: "knowledgeFAQ", Method: "ANY", APIPath: "/api/dashboard/knowledge-faq/list", SortNo: 1550}
	PermissionKnowledgeFAQCreate      = Permission{Name: "创建知识FAQ", Code: "knowledgeFAQ.create", Type: "api", GroupName: "knowledgeFAQ", Method: "POST", APIPath: "/api/dashboard/knowledge-faq/create", SortNo: 1560}
	PermissionKnowledgeFAQUpdate      = Permission{Name: "更新知识FAQ", Code: "knowledgeFAQ.update", Type: "api", GroupName: "knowledgeFAQ", Method: "POST", APIPath: "/api/dashboard/knowledge-faq/update", SortNo: 1570}
	PermissionKnowledgeFAQDelete      = Permission{Name: "删除知识FAQ", Code: "knowledgeFAQ.delete", Type: "api", GroupName: "knowledgeFAQ", Method: "POST", APIPath: "/api/dashboard/knowledge-faq/delete", SortNo: 1580}

	// Skill 定义相关权限
	PermissionSkillDefinitionView   = Permission{Name: "查看技能定义", Code: "skillDefinition.view", Type: "api", GroupName: "skillDefinition", Method: "ANY", APIPath: "/api/dashboard/skill-definition/list", SortNo: 1610}
	PermissionSkillDefinitionCreate = Permission{Name: "创建技能定义", Code: "skillDefinition.create", Type: "api", GroupName: "skillDefinition", Method: "POST", APIPath: "/api/dashboard/skill-definition/create", SortNo: 1620}
	PermissionSkillDefinitionUpdate = Permission{Name: "更新技能定义", Code: "skillDefinition.update", Type: "api", GroupName: "skillDefinition", Method: "POST", APIPath: "/api/dashboard/skill-definition/update", SortNo: 1630}
	PermissionSkillDefinitionDelete = Permission{Name: "删除技能定义", Code: "skillDefinition.delete", Type: "api", GroupName: "skillDefinition", Method: "POST", APIPath: "/api/dashboard/skill-definition/delete", SortNo: 1640}

	// MCP 调试相关权限
	PermissionMCPView = Permission{Name: "查看MCP调试信息", Code: "mcp.view", Type: "api", GroupName: "mcp", Method: "POST", APIPath: "/api/dashboard/mcp/list_tools", SortNo: 1710}
	PermissionMCPCall = Permission{Name: "调用MCP工具", Code: "mcp.call", Type: "api", GroupName: "mcp", Method: "POST", APIPath: "/api/dashboard/mcp/call_tool", SortNo: 1720}
)

// Permissions 内置权限列表
var Permissions = []Permission{
	PermissionUserView,
	PermissionUserCreate,
	PermissionUserUpdate,
	PermissionUserDelete,
	PermissionUserAssignRole,
	PermissionRoleView,
	PermissionRoleCreate,
	PermissionRoleUpdate,
	PermissionRoleDelete,
	PermissionRoleAssignPermission,
	PermissionPermissionView,
	PermissionPermissionSync,
	PermissionSessionView,
	PermissionSessionRevoke,
	PermissionConversationView,
	PermissionConversationAssign,
	PermissionConversationTransfer,
	PermissionConversationClose,
	PermissionConversationSend,
	PermissionConversationTag,
	PermissionConversationHandover,
	PermissionConversationRecycle,
	PermissionConversationLinkCustomer,
	PermissionTicketView,
	PermissionTicketCreate,
	PermissionTicketUpdate,
	PermissionTicketAssign,
	PermissionTicketChangeStatus,
	PermissionTicketProgress,
	PermissionNotificationView,
	PermissionNotificationUpdate,
	PermissionQuickReplyView,
	PermissionQuickReplyCreate,
	PermissionQuickReplyUpdate,
	PermissionQuickReplyDelete,
	PermissionTagView,
	PermissionTagCreate,
	PermissionTagUpdate,
	PermissionTagDelete,
	PermissionCompanyView,
	PermissionCompanyCreate,
	PermissionCompanyUpdate,
	PermissionCompanyDelete,
	PermissionChannelView,
	PermissionChannelCreate,
	PermissionChannelUpdate,
	PermissionChannelDelete,
	PermissionCustomerView,
	PermissionCustomerCreate,
	PermissionCustomerUpdate,
	PermissionCustomerDelete,
	PermissionSalesLeadView,
	PermissionSalesLeadUpdate,
	PermissionSalesLeadAssign,
	PermissionSalesLeadFollowUp,
	PermissionProductView,
	PermissionProductCreate,
	PermissionProductUpdate,
	PermissionProductDelete,
	PermissionProductReindex,
	PermissionPromotionView,
	PermissionPromotionCreate,
	PermissionPromotionUpdate,
	PermissionPromotionDelete,
	PermissionPromotionReindex,
	PermissionDigitalStoreView,
	PermissionDigitalStoreUpdate,
	PermissionAgentView,
	PermissionAgentCreate,
	PermissionAgentUpdate,
	PermissionAgentDelete,
	PermissionAgentUpdateStatus,
	PermissionAgentConfig,
	PermissionAgentTeamView,
	PermissionAgentTeamCreate,
	PermissionAgentTeamUpdate,
	PermissionAgentTeamDelete,
	PermissionAgentTeamScheduleView,
	PermissionAgentTeamScheduleCreate,
	PermissionAgentTeamScheduleUpdate,
	PermissionAgentTeamScheduleDelete,
	PermissionAgentTeamScheduleBatchGenerate,
	PermissionAssetView,
	PermissionAssetCreate,
	PermissionAssetDelete,
	PermissionAIAgentView,
	PermissionAIAgentCreate,
	PermissionAIAgentUpdate,
	PermissionAIAgentDelete,
	PermissionAIConfigView,
	PermissionAIConfigCreate,
	PermissionAIConfigUpdate,
	PermissionAIConfigDelete,
	PermissionKnowledgeBaseView,
	PermissionKnowledgeBaseCreate,
	PermissionKnowledgeBaseUpdate,
	PermissionKnowledgeBaseDelete,
	PermissionKnowledgeDocumentView,
	PermissionKnowledgeDocumentCreate,
	PermissionKnowledgeDocumentUpdate,
	PermissionKnowledgeDocumentDelete,
	PermissionKnowledgeFAQView,
	PermissionKnowledgeFAQCreate,
	PermissionKnowledgeFAQUpdate,
	PermissionKnowledgeFAQDelete,
	PermissionSkillDefinitionView,
	PermissionSkillDefinitionCreate,
	PermissionSkillDefinitionUpdate,
	PermissionSkillDefinitionDelete,
	PermissionMCPView,
	PermissionMCPCall,
}

// PermissionMap 权限映射，用于通过 Code 查找 Permission
var PermissionMap = make(map[string]Permission)

// init 初始化 PermissionMap
func init() {
	normalizeBuiltinPermissionNames()
	for _, permission := range Permissions {
		PermissionMap[permission.Code] = permission
	}
}

func normalizeBuiltinPermissionNames() {
	for i := range Permissions {
		Permissions[i].Name = builtinPermissionName(Permissions[i].Code, Permissions[i].Name)
	}
}

func builtinPermissionName(code string, fallback string) string {
	if name, ok := builtinPermissionNameOverrides[code]; ok {
		return name
	}
	resourceKey, actionKey, ok := splitPermissionCode(code)
	if !ok {
		return fallback
	}
	action, ok := builtinPermissionActionLabels[actionKey]
	if !ok {
		return fallback
	}
	resource, ok := builtinPermissionResourceLabels[resourceKey]
	if !ok {
		return fallback
	}
	return action + " " + resource
}

func splitPermissionCode(code string) (string, string, bool) {
	for i := 0; i < len(code); i++ {
		if code[i] == '.' {
			return code[:i], code[i+1:], i > 0 && i < len(code)-1
		}
	}
	return "", "", false
}

var builtinPermissionActionLabels = map[string]string{
	"view":             "View",
	"create":           "Create",
	"update":           "Update",
	"delete":           "Delete",
	"assignRole":       "Assign roles to",
	"assignPermission": "Assign permissions to",
	"sync":             "Sync",
	"revoke":           "Revoke",
	"assign":           "Assign",
	"transfer":         "Transfer",
	"close":            "Close",
	"send":             "Send",
	"tag":              "Manage tags for",
	"handover":         "Handle handoffs for",
	"recycle":          "Recycle",
	"linkCustomer":     "Link customers to",
	"changeStatus":     "Change status for",
	"progress":         "Update progress for",
	"updateStatus":     "Update status for",
	"config":           "Configure service rules for",
	"batchGenerate":    "Batch generate",
	"call":             "Call",
}

var builtinPermissionResourceLabels = map[string]string{
	"user":              "users",
	"role":              "roles",
	"permission":        "permissions",
	"session":           "sessions",
	"conversation":      "conversations",
	"ticket":            "tickets",
	"notification":      "notifications",
	"quickReply":        "quick replies",
	"tag":               "tags",
	"company":           "companies",
	"channel":           "channels",
	"customer":          "customers",
	"salesLead":         "sales leads",
	"product":           "products",
	"promotion":         "promotions",
	"digitalStore":      "digital store profiles",
	"agent":             "agents",
	"agentTeam":         "agent teams",
	"agentTeamSchedule": "agent team schedules",
	"asset":             "file assets",
	"aiAgent":           "AI Agents",
	"aiConfig":          "AI configurations",
	"knowledgeBase":     "knowledge bases",
	"knowledgeDocument": "knowledge documents",
	"knowledgeFAQ":      "knowledge FAQs",
	"skillDefinition":   "Skill definitions",
	"mcp":               "MCP tools",
}

var builtinPermissionNameOverrides = map[string]string{
	"user.assignRole":                 "Assign user roles",
	"role.assignPermission":           "Assign role permissions",
	"session.revoke":                  "Revoke sessions",
	"conversation.send":               "Send conversation messages",
	"conversation.linkCustomer":       "Link conversation customer",
	"ticket.changeStatus":             "Change ticket status",
	"ticket.progress":                 "Update ticket progress",
	"agent.config":                    "Configure agent service rules",
	"agentTeamSchedule.batchGenerate": "Batch generate agent team schedules",
	"mcp.view":                        "View MCP debug information",
	"mcp.call":                        "Call MCP tools",
	"product.reindex":                 "Reindex products",
	"promotion.reindex":               "Reindex promotions",
}

type RoleSpec struct {
	Name   string
	Code   string
	SortNo int
}

var Roles = []RoleSpec{
	{Name: "Super Admin", Code: RoleCodeSuperAdmin, SortNo: 1},
	{Name: "Admin", Code: RoleCodeAdmin, SortNo: 2},
	{Name: "Support Team Lead", Code: RoleCodeCsTeamLeader, SortNo: 3},
	{Name: "Support Agent", Code: RoleCodeCsUser, SortNo: 4},
}

var RolePermissions = map[string][]Permission{
	RoleCodeSuperAdmin: Permissions,
	RoleCodeAdmin: {
		PermissionUserView, PermissionUserCreate, PermissionUserUpdate, PermissionUserAssignRole,
		PermissionRoleView, PermissionRoleCreate, PermissionRoleUpdate, PermissionRoleAssignPermission,
		PermissionPermissionView, PermissionPermissionSync,
		PermissionSessionView, PermissionSessionRevoke,
		PermissionConversationView, PermissionConversationAssign, PermissionConversationTransfer, PermissionConversationClose, PermissionConversationSend, PermissionConversationTag, PermissionConversationHandover, PermissionConversationRecycle, PermissionConversationLinkCustomer,
		PermissionTicketView, PermissionTicketCreate, PermissionTicketUpdate, PermissionTicketAssign, PermissionTicketChangeStatus, PermissionTicketProgress,
		PermissionNotificationView, PermissionNotificationUpdate,
		PermissionQuickReplyView, PermissionQuickReplyCreate, PermissionQuickReplyUpdate, PermissionQuickReplyDelete,
		PermissionTagView, PermissionTagCreate, PermissionTagUpdate, PermissionTagDelete,
		PermissionCompanyView, PermissionCompanyCreate, PermissionCompanyUpdate, PermissionCompanyDelete,
		PermissionChannelView, PermissionChannelCreate, PermissionChannelUpdate, PermissionChannelDelete,
		PermissionCustomerView, PermissionCustomerCreate, PermissionCustomerUpdate, PermissionCustomerDelete,
		PermissionSalesLeadView, PermissionSalesLeadUpdate, PermissionSalesLeadAssign, PermissionSalesLeadFollowUp,
		PermissionProductView, PermissionProductCreate, PermissionProductUpdate, PermissionProductDelete, PermissionProductReindex,
		PermissionPromotionView, PermissionPromotionCreate, PermissionPromotionUpdate, PermissionPromotionDelete, PermissionPromotionReindex,
		PermissionDigitalStoreView, PermissionDigitalStoreUpdate,
		PermissionAgentView, PermissionAgentCreate, PermissionAgentUpdate, PermissionAgentDelete, PermissionAgentUpdateStatus, PermissionAgentConfig,
		PermissionAgentTeamView, PermissionAgentTeamCreate, PermissionAgentTeamUpdate, PermissionAgentTeamDelete,
		PermissionAgentTeamScheduleView, PermissionAgentTeamScheduleCreate, PermissionAgentTeamScheduleUpdate, PermissionAgentTeamScheduleDelete, PermissionAgentTeamScheduleBatchGenerate,
		PermissionAssetView, PermissionAssetCreate, PermissionAssetDelete,
		PermissionAIAgentView, PermissionAIAgentCreate, PermissionAIAgentUpdate, PermissionAIAgentDelete,
		PermissionAIConfigView, PermissionAIConfigCreate, PermissionAIConfigUpdate, PermissionAIConfigDelete,
		PermissionSkillDefinitionView, PermissionSkillDefinitionCreate, PermissionSkillDefinitionUpdate, PermissionSkillDefinitionDelete,
	},
	RoleCodeCsTeamLeader: {
		PermissionUserView,
		PermissionRoleView,
		PermissionPermissionView,
		PermissionSessionView,
		PermissionConversationView, PermissionConversationClose, PermissionConversationSend, PermissionConversationTag, PermissionConversationHandover, PermissionConversationRecycle, PermissionConversationLinkCustomer,
		PermissionTicketView, PermissionTicketCreate, PermissionTicketUpdate, PermissionTicketAssign, PermissionTicketChangeStatus, PermissionTicketProgress,
		PermissionNotificationView, PermissionNotificationUpdate,
		PermissionQuickReplyView, PermissionQuickReplyCreate, PermissionQuickReplyUpdate, PermissionQuickReplyDelete,
		PermissionTagView, PermissionTagCreate, PermissionTagUpdate, PermissionTagDelete,
		PermissionCompanyView,
		PermissionChannelView, PermissionChannelCreate, PermissionChannelUpdate,
		PermissionCustomerView, PermissionCustomerCreate, PermissionCustomerUpdate,
		PermissionSalesLeadView, PermissionSalesLeadUpdate, PermissionSalesLeadAssign, PermissionSalesLeadFollowUp,
		PermissionProductView, PermissionProductCreate, PermissionProductUpdate, PermissionProductReindex,
		PermissionPromotionView, PermissionPromotionCreate, PermissionPromotionUpdate, PermissionPromotionReindex,
		PermissionDigitalStoreView, PermissionDigitalStoreUpdate,
		PermissionAgentView, PermissionAgentUpdate,
		PermissionAgentTeamView,
		PermissionAgentTeamScheduleView, PermissionAgentTeamScheduleCreate, PermissionAgentTeamScheduleUpdate, PermissionAgentTeamScheduleDelete, PermissionAgentTeamScheduleBatchGenerate,
		PermissionAssetView, PermissionAssetCreate, PermissionAssetDelete,
		PermissionAIAgentView, PermissionAIAgentCreate, PermissionAIAgentUpdate,
		PermissionAIConfigView,
		PermissionSkillDefinitionView, PermissionSkillDefinitionCreate, PermissionSkillDefinitionUpdate,
	},
	RoleCodeCsUser: {
		PermissionUserView,
		PermissionRoleView,
		PermissionPermissionView,
		PermissionConversationView,
		PermissionTicketView, PermissionTicketCreate, PermissionTicketAssign, PermissionTicketChangeStatus, PermissionTicketProgress,
		PermissionNotificationView, PermissionNotificationUpdate,
		PermissionQuickReplyView,
		PermissionTagView,
		PermissionCompanyView,
		PermissionChannelView,
		PermissionCustomerView,
		PermissionSalesLeadView, PermissionSalesLeadUpdate, PermissionSalesLeadFollowUp,
		PermissionProductView,
		PermissionPromotionView,
		PermissionDigitalStoreView,
		PermissionAssetView,
		PermissionAgentView,
		PermissionAgentTeamView,
		PermissionAgentTeamScheduleView,
		PermissionAIAgentView,
		PermissionAIConfigView,
		PermissionSkillDefinitionView,
	},
}

func PermissionCodes() []string {
	ret := make([]string, 0, len(Permissions))
	for _, permission := range Permissions {
		ret = append(ret, permission.Code)
	}
	return ret
}
