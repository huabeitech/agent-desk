package models

import (
	"agent-desk/internal/pkg/enums"
	"time"
)

// Models 注册所有需要迁移和代码生成的模型。
var Models = []any{
	&Migration{},
	&Organization{},
	&OrganizationMember{},
	&User{},
	&UserIdentity{},
	&Company{},
	&Customer{},
	&CustomerIdentity{},
	&CustomerContact{},
	&Role{},
	&Permission{},
	&UserRole{},
	&RolePermission{},
	&UserPermission{},
	&LoginSession{},
	&LoginCredentialLog{},
	&Asset{},
	&Tag{},
	&Conversation{},
	&ConversationParticipant{},
	&ConversationReadState{},
	&Message{},
	&WxWorkKFSyncState{},
	&WxWorkKFConversation{},
	&WxWorkKFMessageRef{},
	&ChannelMessageOutbox{},
	&ConversationAssignment{},
	&ConversationTag{},
	&QuickReply{},
	&ConversationEventLog{},
	&Ticket{},
	&TicketTag{},
	&TicketProgress{},
	&TicketView{},
	&TicketNoSequence{},
	&Notification{},
	&AIAgent{},
	&Channel{},
	&AgentProfile{},
	&AgentTeam{},
	&AgentTeamSchedule{},
	&AIConfig{},
	&KnowledgeBase{},
	&KnowledgeDirectory{},
	&KnowledgeDocument{},
	&KnowledgeFAQ{},
	&KnowledgeChunk{},
	&KnowledgeRetrieveLog{},
	&KnowledgeRetrieveHit{},
	&KnowledgeFeedback{},
	&DocPage{},
	&Category{},
	&Post{},
	&Comment{},
	&Reaction{},
	&CommentReport{},
	&SkillDefinition{},
	&AgentRevision{},
	&AgentRun{},
	&AgentStep{},
	&AgentToolCall{},
	&AgentToolInvocation{},
	&AgentRunQualityFeedback{},
	&AIWorkflow{},
	&AIWorkflowVersion{},
	&AIAgentWorkflowBinding{},
	&AIWorkflowRun{},
	&AIWorkflowNodeRun{},
	&ConversationInterrupt{},
	&SystemConfig{},
}

// AgentToolInvocation persists the idempotency boundary for a business tool.
// It is intentionally independent of AgentRun audit rows so a retry after a
// process interruption cannot repeat an external write.
type AgentToolInvocation struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	ConversationID int64  `gorm:"type:bigint;not null;index;uniqueIndex:uk_agent_tool_invocation"`
	AIAgentID      int64  `gorm:"type:bigint;not null;default:0;index"`
	ToolCode       string `gorm:"type:varchar(128);not null;default:'';uniqueIndex:uk_agent_tool_invocation"`
	IdempotencyKey string `gorm:"type:varchar(160);not null;default:'';uniqueIndex:uk_agent_tool_invocation"`
	Status         string `gorm:"type:varchar(20);not null;default:'running';index"`
	ResultData     string `gorm:"type:text"`
	ErrorMessage   string `gorm:"type:text"`
	AuditFields
}

type Migration struct {
	ID         int64  `gorm:"primaryKey;autoIncrement"`
	Version    int64  `gorm:"type:bigint;not null;uniqueIndex"`
	Remark     string `gorm:"type:text"`
	Success    bool   `gorm:"not null;default:false"`
	ErrorInfo  string `gorm:"type:text"`
	RetryCount int    `gorm:"type:int;not null;default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SystemConfig 运营侧系统配置项；具体有哪些 config_key 由业务代码约定，表内一行一项。
type SystemConfig struct {
	ID          int64        `gorm:"primaryKey;autoIncrement"`
	ConfigKey   string       `gorm:"column:config_key;type:varchar(128);not null;uniqueIndex:uk_system_config_group_key"`
	ConfigValue string       `gorm:"column:config_value;type:text;not null"`
	GroupCode   string       `gorm:"column:group_code;type:varchar(64);not null;default:'';index;uniqueIndex:uk_system_config_group_key"`
	Title       string       `gorm:"type:varchar(200);not null;default:''"`
	Description string       `gorm:"type:text"`
	Status      enums.Status `gorm:"type:int;not null;default:0;index"`
	AuditFields
}

// TicketNoSequence 工单号日序列表。
//
// 每天一条记录，NextSeq 表示当日下一次可分配的序号。
type TicketNoSequence struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	DateKey   string    `gorm:"column:date_key;type:varchar(8);not null;uniqueIndex"`
	NextSeq   int64     `gorm:"column:next_seq;type:bigint;not null;default:1"`
	CreatedAt time.Time `gorm:"not null;index"`
	UpdatedAt time.Time `gorm:"not null;index"`
}

// TicketView 工单工作台个人保存视图。
type TicketView struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	UserID      int64  `gorm:"column:user_id;type:bigint;not null;index"`
	Name        string `gorm:"column:name;type:varchar(100);not null;default:'';index"`
	FiltersJSON string `gorm:"column:filters_json;type:text;not null"`
	SortNo      int    `gorm:"column:sort_no;type:int;not null;default:0;index"`
	AuditFields
}

// Notification 站内通知。
type Notification struct {
	ID               int64        `gorm:"primaryKey;autoIncrement"`
	RecipientUserID  int64        `gorm:"type:bigint;not null;default:0;index"`
	Title            string       `gorm:"type:varchar(255);not null;default:''"`
	Content          string       `gorm:"type:text"`
	NotificationType string       `gorm:"type:varchar(50);not null;default:'';index"`
	BizType          string       `gorm:"type:varchar(50);not null;default:'';index"`
	BizID            int64        `gorm:"type:bigint;not null;default:0;index"`
	ActionURL        string       `gorm:"type:varchar(255);not null;default:''"`
	ReadAt           *time.Time   `gorm:"index"`
	Status           enums.Status `gorm:"type:int;not null;default:0;index"`
	CreatedAt        time.Time    `gorm:"not null;index"`
}

// AuditFields 定义涉及用户操作数据的统一审计字段。
// 该结构记录数据创建与更新的时间、操作者ID和操作者名称。
type AuditFields struct {
	CreatedAt      time.Time `gorm:"not null;index"`                        // CreatedAt 记录数据创建时间。
	CreateUserID   int64     `gorm:"type:bigint;not null;default:0;index"`  // CreateUserID 记录创建人用户ID；系统任务写0。
	CreateUserName string    `gorm:"type:varchar(100);not null;default:''"` // CreateUserName 记录创建人名称；系统任务写system。
	UpdatedAt      time.Time `gorm:"not null;index"`                        // UpdatedAt 记录数据最近更新时间。
	UpdateUserID   int64     `gorm:"type:bigint;not null;default:0;index"`  // UpdateUserID 记录最后更新人用户ID；系统任务写0。
	UpdateUserName string    `gorm:"type:varchar(100);not null;default:''"` // UpdateUserName 记录最后更新人名称；系统任务写system。
}

// Organization 组织 / 工作区（Tenant / Workspace）。
type Organization struct {
	ID     int64        `gorm:"primaryKey;autoIncrement"`
	Code   string       `gorm:"type:varchar(100);not null;uniqueIndex:uk_org_code"` // 组织唯一编码 / Slug，如 org_123456
	Name   string       `gorm:"type:varchar(200);not null;default:''"`              // 组织名称
	Logo   string       `gorm:"type:varchar(255);not null;default:''"`              // 组织 Logo URL
	Plan   string       `gorm:"type:varchar(50);not null;default:'free'"`           // 订阅计划（free, pro, enterprise）
	Status enums.Status `gorm:"type:int;not null;default:0;index"`                  // 组织状态
	Remark string       `gorm:"type:text"`                                          // 备注
	AuditFields
}

// OrganizationMember 组织成员关联表。
type OrganizationMember struct {
	ID             int64        `gorm:"primaryKey;autoIncrement"`
	OrganizationID int64        `gorm:"type:bigint;not null;index;uniqueIndex:uk_org_member"` // 组织 ID
	UserID         int64        `gorm:"type:bigint;not null;index;uniqueIndex:uk_org_member"` // 用户 ID
	Role           string       `gorm:"type:varchar(50);not null;default:'MEMBER';index"`     // 角色：OWNER | ADMIN | MEMBER
	Status         enums.Status `gorm:"type:int;not null;default:0;index"`                    // 成员状态
	AuditFields
}

// User 后台用户账号。
type User struct {
	ID           int64          `gorm:"primaryKey;autoIncrement"`
	Username     string         `gorm:"type:varchar(100);not null;uniqueIndex"`
	Nickname     string         `gorm:"type:varchar(100);not null;default:'';index"`
	Avatar       string         `gorm:"type:varchar(255);not null;default:''"`
	Mobile       *string        `gorm:"type:varchar(32);uniqueIndex"`
	Email        *string        `gorm:"type:varchar(100);uniqueIndex"`
	Password     string         `gorm:"type:varchar(255);not null;default:''"`
	PasswordSalt string         `gorm:"type:varchar(64);not null;default:''"`
	UserType     enums.UserType `gorm:"column:user_type;type:varchar(20);not null;default:'user';index"`
	ActiveOrgID  int64          `gorm:"column:active_org_id;type:bigint;not null;default:0;index"`
	Status       enums.Status   `gorm:"type:int;not null;default:0;index"`
	LastLoginAt  *time.Time
	LastLoginIP  string     `gorm:"type:varchar(64);not null;default:''"`
	Remark       string     `gorm:"type:text"`
	DeletedAt    *time.Time `gorm:"index"`
	AuditFields
}

// UserIdentity 第三方身份绑定信息。
type UserIdentity struct {
	ID              int64               `gorm:"primaryKey;autoIncrement"`
	UserID          int64               `gorm:"type:bigint;not null;index;uniqueIndex:uk_provider_user"`
	Provider        enums.ThirdProvider `gorm:"type:varchar(50);not null;default:'';index;uniqueIndex:uk_provider_user;uniqueIndex:uk_provider_union"`
	ProviderUserID  string              `gorm:"type:varchar(128);not null;default:'';uniqueIndex:uk_provider_user"`
	ProviderUnionID *string             `gorm:"type:varchar(128);uniqueIndex:uk_provider_union"`
	ProviderCorpID  string              `gorm:"type:varchar(128);not null;default:'';index"`
	ProviderName    string              `gorm:"type:varchar(100);not null;default:''"`
	RawProfile      string              `gorm:"type:text"`
	Status          enums.Status        `gorm:"type:int;not null;default:0;index"`
	LastAuthAt      *time.Time
	AuditFields
}

// Company 客户公司（组织）表。
//
//	用于存储公司主体信息；Customer（人）可通过 CompanyID 关联到所属公司。
type Company struct {
	ID     int64        `gorm:"primaryKey;autoIncrement"`                               // ID 为公司主键。
	Name   string       `gorm:"type:varchar(200);not null;uniqueIndex:uk_company_name"` // Name 为公司名称（唯一）。
	Code   string       `gorm:"type:varchar(64);not null;index"`                        // Code 为公司编码/统一社会信用代码（可空语义用空串表示）。
	Status enums.Status `gorm:"type:int;not null;default:0"`                            // Status 为公司状态。
	Remark string       `gorm:"type:text"`                                              // Remark 为备注。
	AuditFields
}

// Customer 客户主表。
//
//	用于存储客户稳定画像信息，不包含平台身份映射和多联系方式明细。
type Customer struct {
	ID            int64        `gorm:"primaryKey;autoIncrement"`                    // ID 为客户主键。
	UserID        int64        `gorm:"type:bigint;not null;default:0;index"`        // UserID 为关联的统一登录账号ID，0表示未绑定。
	Name          string       `gorm:"type:varchar(100);not null;default:'';index"` // Name 为客户姓名或展示名称。
	Gender        enums.Gender `gorm:"type:int;not null;default:0;"`                // Gender 为性别：0未知 1男 2女。
	CompanyID     int64        `gorm:"type:bigint;not null;default:0;index"`        // CompanyID 为所属公司ID；0表示无所属公司（个人客户）。
	LastActiveAt  *time.Time   // LastActiveAt 为最近活跃时间。
	PrimaryMobile string       `gorm:"type:varchar(32);not null;default:'';index"`  // PrimaryMobile 为主手机号（冗余展示字段）。
	PrimaryEmail  string       `gorm:"type:varchar(100);not null;default:'';index"` // PrimaryEmail 为主邮箱（冗余展示字段）。
	Status        enums.Status `gorm:"type:int;not null;default:0;"`                // Status 为客户状态。
	Remark        string       `gorm:"type:text"`                                   // Remark 为备注。
	AuditFields
}

// CustomerIdentity 客户第三方身份映射表。
type CustomerIdentity struct {
	ID             int64                `gorm:"primaryKey;autoIncrement"`
	CustomerID     int64                `gorm:"type:bigint;not null;uniqueIndex:uk_customer_external"`                    // 为所属客户ID。
	ExternalSource enums.ExternalSource `gorm:"type:varchar(30);uniqueIndex:uk_customer_external"`                        // 为外部身份来源
	ExternalID     string               `gorm:"type:varchar(128);index:idx_external_id;uniqueIndex:uk_customer_external"` // 为平台侧用户唯一ID，与访客 ExternalID 对齐。
	RawProfile     string               `gorm:"type:text"`                                                                // 为第三方原始资料JSON。
	Status         enums.Status         `gorm:"type:int;not null;default:0;index"`                                        // 为映射状态。
	AuditFields
}

// CustomerContact 客户联系方式表。
//
//	用于维护客户的一对多联系方式，支持主联系方式、验证状态与失效标记。
type CustomerContact struct {
	ID           int64             `gorm:"primaryKey;autoIncrement"`
	CustomerID   int64             `gorm:"type:bigint;not null;index;uniqueIndex:uk_customer_contact"`                  // CustomerID 为所属客户ID。
	ContactType  enums.ContactType `gorm:"type:varchar(30);not null;default:'';index;uniqueIndex:uk_customer_contact"`  // ContactType 为联系方式类型：mobile/email/wechat/other。
	ContactValue string            `gorm:"type:varchar(200);not null;default:'';index;uniqueIndex:uk_customer_contact"` // ContactValue 为联系方式值。
	IsPrimary    bool              `gorm:"not null;default:false;index"`                                                // IsPrimary 表示是否主联系方式。
	IsVerified   bool              `gorm:"not null;default:false;index"`                                                // IsVerified 表示是否已验证。
	VerifiedAt   *time.Time        // VerifiedAt 为验证时间。
	Source       string            `gorm:"type:varchar(30);not null;default:'';index"` // Source 为来源：manual/import/system。
	Status       enums.Status      `gorm:"type:int;not null;default:0;index"`          // Status 为联系方式状态。
	Remark       string            `gorm:"type:varchar(255);not null;default:''"`      // Remark 为备注。
	AuditFields
}

// Role 角色定义。
type Role struct {
	ID       int64        `gorm:"primaryKey;autoIncrement"`
	Name     string       `gorm:"type:varchar(100);not null;default:'';index"`
	Code     string       `gorm:"type:varchar(100);not null;uniqueIndex"`
	Status   enums.Status `gorm:"type:int;not null;default:0;index"`
	IsSystem bool         `gorm:"not null;default:false;index"`
	SortNo   int          `gorm:"type:int;not null;default:0;index"`
	Remark   string       `gorm:"type:text"`
	AuditFields
}

// Permission 权限点定义。
type Permission struct {
	ID        int64        `gorm:"primaryKey;autoIncrement"`
	Name      string       `gorm:"type:varchar(100);not null;default:''"`
	Code      string       `gorm:"type:varchar(150);not null;uniqueIndex"`
	Type      string       `gorm:"type:varchar(20);not null;default:'';index"`
	GroupName string       `gorm:"type:varchar(100);not null;default:'';index"`
	ParentID  int64        `gorm:"type:bigint;not null;default:0;index"`
	Path      string       `gorm:"type:varchar(255);not null;default:''"`
	Method    string       `gorm:"type:varchar(20);not null;default:''"`
	APIPath   string       `gorm:"type:varchar(255);not null;default:''"`
	SortNo    int          `gorm:"type:int;not null;default:0;index"`
	Status    enums.Status `gorm:"type:int;not null;default:0;index"`
	IsBuiltin bool         `gorm:"not null;default:true;index"`
	Remark    string       `gorm:"type:text"`
	AuditFields
}

// UserRole 用户和角色关联。
type UserRole struct {
	ID     int64 `gorm:"primaryKey;autoIncrement"`
	UserID int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_user_role"`
	RoleID int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_user_role"`
	AuditFields
}

// RolePermission 角色和权限关联。
type RolePermission struct {
	ID           int64 `gorm:"primaryKey;autoIncrement"`
	RoleID       int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_role_permission"`
	PermissionID int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_role_permission"`
	AuditFields
}

// UserPermission 用户级例外权限。
//
//	用于处理少量临时授权或拒绝授权场景。
type UserPermission struct {
	ID           int64 `gorm:"primaryKey;autoIncrement"`
	UserID       int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_user_permission"`
	PermissionID int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_user_permission"`
	Effect       int   `gorm:"type:int;not null;default:1;index"` // Effect 表示权限生效方式：1允许 -1拒绝。
	ExpiredAt    *time.Time
	Remark       string `gorm:"type:text"`
	AuditFields
}

// LoginSession 表示一次后台登录会话。
type LoginSession struct {
	ID         int64      `gorm:"primaryKey;autoIncrement"`                   // ID 为登录会话主键。
	UserID     int64      `gorm:"type:bigint;not null;index"`                 // UserID 为登录用户 ID。
	Token      string     `gorm:"type:varchar(128);not null;uniqueIndex"`     // Token 为随机不透明登录凭证，使用 ak_ 前缀。
	ClientType string     `gorm:"type:varchar(50);not null;default:'';index"` // ClientType 为客户端类型，后台 Web 端固定为 admin_web。
	ClientIP   string     `gorm:"type:varchar(64);not null;default:''"`       // ClientIP 为登录请求来源 IP。
	UserAgent  string     `gorm:"type:varchar(255);not null;default:''"`      // UserAgent 为登录请求浏览器或客户端 UA。
	ExpiredAt  time.Time  `gorm:"not null;index"`                             // ExpiredAt 为 token 过期时间。
	RevokedAt  *time.Time `gorm:"index"`                                      // RevokedAt 为主动注销或踢下线时间，非空表示已失效。
	LastSeenAt *time.Time // LastSeenAt 为最近一次成功鉴权时间。
	AuditFields
}

// LoginCredentialLog 记录一次后台登录凭证校验结果。
type LoginCredentialLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`                    // ID 为登录凭证日志主键。
	Principal string    `gorm:"type:varchar(100);not null;default:'';index"` // Principal 为用户输入的登录名。
	UserID    int64     `gorm:"type:bigint;not null;default:0;index"`        // UserID 为匹配到的用户 ID，未匹配时为 0。
	Success   bool      `gorm:"not null;default:false;index"`                // Success 表示本次凭证校验是否成功。
	ClientIP  string    `gorm:"type:varchar(64);not null;default:''"`        // ClientIP 为登录请求来源 IP。
	UserAgent string    `gorm:"type:varchar(255);not null;default:''"`       // UserAgent 为登录请求浏览器或客户端 UA。
	Reason    string    `gorm:"type:varchar(255);not null;default:''"`       // Reason 为校验结果原因。
	CreatedAt time.Time `gorm:"not null;index"`                              // CreatedAt 为日志创建时间。
}

// Asset 存储的文件资源，如上传的附件等。
type Asset struct {
	ID         int64               `gorm:"primaryKey;autoIncrement"`
	AssetID    string              `gorm:"type:varchar(64);not null;uniqueIndex"`
	Provider   enums.AssetProvider `gorm:"type:varchar(50);not null;default:'';index"`
	StorageKey string              `gorm:"type:varchar(255);not null;default:'';uniqueIndex:uk_storage_key"`
	Filename   string              `gorm:"type:varchar(255);not null;default:''"`
	FileSize   int64               `gorm:"type:bigint;not null;default:0"`
	MimeType   string              `gorm:"type:varchar(100);not null;default:''"`
	Status     enums.AssetStatus   `gorm:"type:int;not null;default:1;index"`
	AuditFields
}

type Tag struct {
	ID       int64        `gorm:"primaryKey;autoIncrement"`
	ParentID int64        `gorm:"type:bigint;not null;index"`
	Name     string       `gorm:"type:varchar(50);not null;"`
	Remark   string       `gorm:"type:text;"`
	SortNo   int          `gorm:"type:int;not null;default:0"`
	Status   enums.Status `gorm:"type:int;not null;default:0"`
	AuditFields
}

// Conversation 客服会话。
type Conversation struct {
	ID                  int64                           `gorm:"primaryKey;autoIncrement"`                    // ID 为会话主键。
	AIAgentID           int64                           `gorm:"type:bigint;not null;default:0;index"`        // AIAgentID 为当前会话绑定的 AI Agent ID。
	ChannelID           int64                           `gorm:"type:bigint;not null;default:0;index"`        // ChannelID 为该会话来源接入渠道ID。
	CustomerID          int64                           `gorm:"type:bigint;not null;default:0;index"`        // CustomerID 为会话所属客户 ID。
	CustomerName        string                          `gorm:"type:varchar(100);not null;default:'';index"` // CustomerName 为客户名称冗余字段，用于列表展示和搜索。
	Status              enums.IMConversationStatus      `gorm:"type:int;not null;default:1;index"`           // Status 为会话状态，如待接入、处理中、已关闭。
	ServiceMode         enums.IMConversationServiceMode `gorm:"type:int;not null;default:3;index"`           // ServiceMode 为服务模式，如仅AI、仅人工、AI优先人工接管。
	Priority            int                             `gorm:"type:int;not null;default:0;index"`           // Priority 为会话优先级。
	CurrentAssigneeID   int64                           `gorm:"type:bigint;not null;default:0;index"`        // CurrentAssigneeID 为当前接待客服ID。
	CurrentTeamID       int64                           `gorm:"type:bigint;not null;default:0;index"`        // CurrentTeamID 为当前处理客服组ID。
	LastMessageID       int64                           `gorm:"type:bigint;not null;default:0;index"`        // LastMessageID 为最后一条消息ID。
	LastMessageAt       time.Time                       `gorm:"index"`                                       // LastMessageAt 为最后消息时间。
	LastActiveAt        time.Time                       `gorm:"index"`                                       // LastActiveAt 为会话最近活跃时间。
	LastMessageSummary  string                          `gorm:"type:varchar(255);not null;default:''"`       // LastMessageSummary 为最后一条消息摘要。
	CustomerUnreadCount int                             `gorm:"type:int;not null;default:0"`                 // CustomerUnreadCount 为用户侧未读数。
	AgentUnreadCount    int                             `gorm:"type:int;not null;default:0"`                 // AgentUnreadCount 为客服侧未读数。
	HandoffAt           *time.Time                      `gorm:"index"`                                       // HandoffAt 为最近一次转人工时间。
	HandoffReason       string                          `gorm:"type:varchar(255);not null;default:''"`       // HandoffReason 为最近一次转人工原因。
	AIReplyRounds       int                             `gorm:"type:int;not null;default:0"`                 // AIReplyRounds 为当前会话内 AI 已成功回复次数。
	ClosedAt            *time.Time                      `gorm:"index"`                                       // ClosedAt 为会话关闭时间。
	ClosedBy            int64                           `gorm:"type:bigint;not null;default:0;index"`        // ClosedBy 为关闭人用户ID，访客关闭时写0。
	CloseReason         string                          `gorm:"type:varchar(255);not null;default:''"`       // CloseReason 为关闭原因。
	AuditFields
}

// ConversationParticipant 会话参与方。
type ConversationParticipant struct {
	ID                    int64  `gorm:"primaryKey;autoIncrement"`
	ConversationID        int64  `gorm:"type:bigint;not null;index;uniqueIndex:uk_conversation_participant"`
	ParticipantType       string `gorm:"type:varchar(30);not null;default:'';index;uniqueIndex:uk_conversation_participant"`
	ParticipantID         int64  `gorm:"type:bigint;not null;default:0;uniqueIndex:uk_conversation_participant"`
	ExternalParticipantID string `gorm:"type:varchar(128);not null;default:''"`
	JoinedAt              *time.Time
	LeftAt                *time.Time
	Status                enums.Status `gorm:"type:int;not null;default:0;index"`
	AuditFields
}

// ConversationReadState 会话读游标。
type ConversationReadState struct {
	ID                int64              `gorm:"primaryKey;autoIncrement"`
	ConversationID    int64              `gorm:"type:bigint;not null;index;uniqueIndex:uk_conversation_reader"`
	ReaderType        enums.IMSenderType `gorm:"type:varchar(30);not null;default:'';index;uniqueIndex:uk_conversation_reader"`
	ReaderID          int64              `gorm:"type:bigint;not null;default:0;uniqueIndex:uk_conversation_reader"`
	ExternalReaderID  string             `gorm:"type:varchar(128);not null;default:'';uniqueIndex:uk_conversation_reader"`
	LastReadMessageID int64              `gorm:"type:bigint;not null;default:0;index"`
	LastReadAt        *time.Time
	AuditFields
}

// Message 会话消息。
type Message struct {
	ID              int64                 `gorm:"primaryKey;autoIncrement"`
	ConversationID  int64                 `gorm:"type:bigint;not null;index;uniqueIndex:uk_conversation_client_msg"`
	RequestID       string                `gorm:"type:varchar(128);not null;default:'';index"`
	WorkflowRunID   int64                 `gorm:"type:bigint;not null;default:0;index"`
	ClientMsgID     string                `gorm:"type:varchar(128);not null;default:'';uniqueIndex:uk_conversation_client_msg"`
	SenderType      enums.IMSenderType    `gorm:"type:varchar(30);not null;default:'';index"`
	SenderID        int64                 `gorm:"type:bigint;not null;default:0;index"`
	ReceiverType    string                `gorm:"type:varchar(30);not null;default:'';index"`
	MessageType     enums.IMMessageType   `gorm:"type:varchar(30);not null;default:'';index"`
	Content         string                `gorm:"type:text"`
	Payload         string                `gorm:"type:text"`
	SendStatus      enums.IMMessageStatus `gorm:"type:int;not null;default:2;index"`
	SentAt          *time.Time            `gorm:"index"`
	DeliveredAt     *time.Time
	ReadAt          *time.Time
	RecalledAt      *time.Time
	QuotedMessageID int64 `gorm:"type:bigint;not null;default:0;index"`
	AuditFields
}

// WxWorkKFSyncState 企业微信客服消息同步状态表。
//
//	按 open_kfid 记录企业微信客服消息同步游标，用于 SyncMsg 增量拉取。
type WxWorkKFSyncState struct {
	ID         int64        `gorm:"primaryKey;autoIncrement"`                         // ID 为同步状态主键。
	OpenKfID   string       `gorm:"type:varchar(64);not null;default:'';uniqueIndex"` // OpenKfID 为企业微信客服账号ID。
	NextCursor string       `gorm:"type:varchar(128);not null;default:''"`            // NextCursor 为下一次增量同步使用的游标。
	LastSyncAt *time.Time   `gorm:"index"`                                            // LastSyncAt 为最近一次成功同步时间。
	Status     enums.Status `gorm:"type:int;not null;default:0;index"`                // Status 为同步状态记录状态。
	Remark     string       `gorm:"type:text"`                                        // Remark 为同步异常、人工备注等补充信息。
	AuditFields
}

// WxWorkKFConversation 企业微信客服渠道会话映射表。
//
//	维护平台会话与企业微信客服会话上下文的对应关系，供入站同步和下行发送复用。
type WxWorkKFConversation struct {
	ID             int64        `gorm:"primaryKey;autoIncrement"`                                   // ID 为渠道会话映射主键。
	ConversationID int64        `gorm:"type:bigint;not null;uniqueIndex"`                           // ConversationID 为平台会话ID，一条平台会话仅对应一条当前有效渠道映射。
	ChannelID      int64        `gorm:"type:bigint;not null;default:0;index"`                       // ChannelID 为所属接入渠道ID，用于标识该会话来自哪个企业微信渠道配置。
	OpenKfID       string       `gorm:"type:varchar(64);not null;default:'';index:idx_openkf_ext"`  // OpenKfID 为企业微信客服账号ID。
	ExternalUserID string       `gorm:"type:varchar(128);not null;default:'';index:idx_openkf_ext"` // ExternalUserID 为企业微信客户ID。
	ServicerUserID string       `gorm:"type:varchar(128);not null;default:'';index"`                // ServicerUserID 为企业微信当前接待客服成员UserID。
	SessionStatus  string       `gorm:"type:varchar(30);not null;default:'';index"`                 // SessionStatus 为微信侧会话状态快照，如接入中、转接中、已结束。
	LastWxMsgID    string       `gorm:"type:varchar(64);not null;default:'';index"`                 // LastWxMsgID 为最近一次同步到的微信消息ID。
	LastWxMsgTime  *time.Time   `gorm:"index"`                                                      // LastWxMsgTime 为最近一次微信消息时间。
	RawProfile     string       `gorm:"type:text"`                                                  // RawProfile 为微信侧原始会话补充信息JSON。
	Status         enums.Status `gorm:"type:int;not null;default:0;index"`                          // Status 为渠道会话映射状态。
	AuditFields
}

// WxWorkKFMessageRef 企业微信客服消息映射表。
//
//	用于实现微信消息幂等消费，并保存平台消息与微信消息的双向映射关系。
type WxWorkKFMessageRef struct {
	ID             int64        `gorm:"primaryKey;autoIncrement"`                         // ID 为消息映射主键。
	ConversationID int64        `gorm:"type:bigint;not null;default:0;index"`             // ConversationID 为所属平台会话ID。
	MessageID      int64        `gorm:"type:bigint;not null;default:0;index"`             // MessageID 为所属平台消息ID；仅渠道消息尚未生成平台消息时可暂为0。
	WxMsgID        string       `gorm:"type:varchar(64);not null;default:'';uniqueIndex"` // WxMsgID 为企业微信消息ID，用于幂等去重。
	Direction      string       `gorm:"type:varchar(20);not null;default:'';index"`       // Direction 为消息方向，如 in/out。
	Origin         int          `gorm:"type:int;not null;default:0;index"`                // Origin 为企业微信消息来源值，如客户发送、系统事件、企微客户端发送。
	OpenKfID       string       `gorm:"type:varchar(64);not null;default:'';index"`       // OpenKfID 为发送或接收该消息的客服账号ID。
	ExternalUserID string       `gorm:"type:varchar(128);not null;default:'';index"`      // ExternalUserID 为消息对应的企业微信客户ID。
	SendStatus     string       `gorm:"type:varchar(30);not null;default:'';index"`       // SendStatus 为渠道发送状态快照，如 sent、failed。
	FailReason     string       `gorm:"type:text"`                                        // FailReason 为渠道发送失败原因或补偿说明。
	RawPayload     string       `gorm:"type:text"`                                        // RawPayload 为企业微信原始消息JSON。
	Status         enums.Status `gorm:"type:int;not null;default:0;index"`                // Status 为消息映射状态。
	AuditFields
}

// ChannelMessageOutbox 外部渠道消息投递任务表。
//
//	用于记录平台消息提交后的渠道发送任务，保证第三方发送动作与主事务解耦。
type ChannelMessageOutbox struct {
	ID             int64      `gorm:"primaryKey;autoIncrement"`                                                  // ID 为投递任务主键。
	ChannelType    string     `gorm:"type:varchar(30);not null;default:'';index;uniqueIndex:uk_channel_message"` // ChannelType 为目标渠道类型，如 wxwork_kf。
	ConversationID int64      `gorm:"type:bigint;not null;default:0;index"`                                      // ConversationID 为所属平台会话ID。
	MessageID      int64      `gorm:"type:bigint;not null;default:0;uniqueIndex:uk_channel_message"`             // MessageID 为待投递的平台消息ID。
	Payload        string     `gorm:"type:text"`                                                                 // Payload 为渠道发送所需的标准化请求数据JSON。
	SendStatus     string     `gorm:"type:varchar(30);not null;default:'';index"`                                // SendStatus 为当前投递状态，如 pending、sending、sent、failed。
	RetryCount     int        `gorm:"type:int;not null;default:0"`                                               // RetryCount 为已重试次数。
	NextRetryAt    *time.Time `gorm:"index"`                                                                     // NextRetryAt 为下一次允许重试时间。
	LastError      string     `gorm:"type:text"`                                                                 // LastError 为最近一次发送失败信息。
	SentAt         *time.Time `gorm:"index"`                                                                     // SentAt 为最终发送成功时间。
	AuditFields
}

// ConversationAssignment 会话接待关系。
type ConversationAssignment struct {
	ID             int64                    `gorm:"primaryKey;autoIncrement"`
	ConversationID int64                    `gorm:"type:bigint;not null;index"`
	FromUserID     int64                    `gorm:"type:bigint;not null;default:0;index"`
	ToUserID       int64                    `gorm:"type:bigint;not null;default:0;index"`
	AssignType     string                   `gorm:"type:varchar(30);not null;default:'';index"`
	Reason         string                   `gorm:"type:varchar(255);not null;default:''"`
	Status         enums.IMAssignmentStatus `gorm:"type:int;not null;index"`
	CreatedAt      time.Time                `gorm:"not null;index"`
	FinishedAt     *time.Time
	OperatorID     int64 `gorm:"type:bigint;not null;default:0;index"`
}

// ConversationTag 会话标签关联
type ConversationTag struct {
	ID             int64 `gorm:"primaryKey;autoIncrement"`
	ConversationID int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_conversation_tag"`
	TagID          int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_conversation_tag"`
	AuditFields
}

// QuickReply 快捷回复。
type QuickReply struct {
	ID        int64        `gorm:"primaryKey;autoIncrement"`
	GroupName string       `gorm:"type:varchar(50);not null;default:'';index"`
	Title     string       `gorm:"type:varchar(100);not null;default:'';index"`
	Content   string       `gorm:"type:text"`
	Status    enums.Status `gorm:"type:int;not null;index"`
	SortNo    int          `gorm:"type:int;not null;index"`
	AuditFields
}

// AIAgent AI 接待实例。
type AIAgent struct {
	ID                     int64                           `gorm:"primaryKey;autoIncrement"`                    // ID 为 AI Agent 主键。
	Name                   string                          `gorm:"type:varchar(100);not null;default:'';index"` // Name 为 AI Agent 名称。
	Description            string                          `gorm:"type:varchar(255);not null;default:''"`       // Description 为 AI Agent 描述。
	Status                 enums.Status                    `gorm:"type:int;not null;index"`                     // Status 为 AI Agent
	AIConfigID             int64                           `gorm:"type:bigint;not null;default:0;index"`        // AIConfigID 为关联的 AI 配置ID。
	MaxSteps               int                             `gorm:"type:int;not null;default:6"`                 // MaxSteps 为一次 Agent Loop 允许的最大推理步骤数。
	ContextWindow          int                             `gorm:"type:int;not null;default:0"`                 // ContextWindow 为会话上下文消息窗口，0 表示使用运行时默认值。
	ToolPolicy             string                          `gorm:"type:text"`                                   // ToolPolicy 为工具风险与确认策略JSON。
	KnowledgePolicy        string                          `gorm:"type:text"`                                   // KnowledgePolicy 为知识检索与无依据回答策略JSON。
	ServiceMode            enums.IMConversationServiceMode `gorm:"type:int;not null;default:3;index"`           // ServiceMode 为服务模式，如仅AI、仅人工、AI优先人工接管。
	SystemPrompt           string                          `gorm:"type:text"`                                   // SystemPrompt 为该 Agent 的系统提示词。
	WelcomeMessage         string                          `gorm:"type:text"`                                   // WelcomeMessage 为该 Agent 的欢迎语或首响模板。
	ReplyTimeoutSeconds    int                             `gorm:"type:int;not null;default:180"`               // ReplyTimeoutSeconds 为异步自动回复超时秒数。
	RolloutPercent         int                             `gorm:"type:int;not null;default:100"`               // RolloutPercent 为该 Agent 的会话灰度百分比，100 表示全量。
	PreviousRolloutPercent int                             `gorm:"type:int;not null;default:0"`                 // PreviousRolloutPercent 保存上一次生效的灰度比例，0 表示尚无可回滚值。
	TeamIDs                string                          `gorm:"type:varchar(500);not null;default:''"`       // TeamIDs 为转人工时可路由的客服组ID列表，多个之间使用逗号分隔。
	HandoffMode            enums.AIAgentHandoffMode        `gorm:"type:int;not null;default:1"`                 // HandoffMode 为转人工执行方式，如进入待接入池、进入默认客服组待接入池。
	FallbackMode           enums.AIAgentFallbackMode       `gorm:"type:int;not null;default:1"`                 // FallbackMode 为知识不足时的回复策略。
	FallbackMessage        string                          `gorm:"type:text"`                                   // FallbackMessage 为知识不足回复文案。
	KnowledgeIDs           string                          `gorm:"type:varchar(500);not null;default:''"`       // KnowledgeIDs 为绑定的知识库ID列表，按顺序表示优先级。
	SkillIDs               string                          `gorm:"type:varchar(500);not null;default:''"`       // SkillIDs 为绑定的技能ID列表，按顺序表示允许路由的范围。
	AllowedMCPTools        string                          `gorm:"type:text"`                                   // AllowedMCPTools 为 Agent 允许调用的 MCP 工具白名单配置 JSON。
	PublishedRevisionID    int64                           `gorm:"type:bigint;not null;default:0;index"`        // PublishedRevisionID 为当前已发布 Agent 配置快照ID。
	SortNo                 int                             `gorm:"type:int;not null;default:0;index"`           // SortNo 为后台展示排序号。
	AuditFields
}

// AgentRevision stores an immutable published Agent configuration snapshot.
type AgentRevision struct {
	ID              int64        `gorm:"primaryKey;autoIncrement"`
	AgentID         int64        `gorm:"type:bigint;not null;index;uniqueIndex:uk_agent_revision"`
	Revision        int          `gorm:"type:int;not null;uniqueIndex:uk_agent_revision"`
	Status          enums.Status `gorm:"type:int;not null;default:0;index"`
	Definition      string       `gorm:"type:text"`
	DefinitionHash  string       `gorm:"type:varchar(64);not null;default:'';index"`
	PublishedAt     *time.Time   `gorm:"index"`
	PublishedByID   int64        `gorm:"type:bigint;not null;default:0;index"`
	PublishedByName string       `gorm:"type:varchar(100);not null;default:''"`
	AuditFields
}

// AgentRun is the parent audit record for one Agent Loop execution.
type AgentRun struct {
	ID               int64      `gorm:"primaryKey;autoIncrement"`
	ConversationID   int64      `gorm:"type:bigint;not null;default:0;index"`
	AIAgentID        int64      `gorm:"type:bigint;not null;default:0;index"`
	AgentRevisionID  int64      `gorm:"type:bigint;not null;default:0;index"`
	SourceMessageID  int64      `gorm:"type:bigint;not null;default:0;index"`
	WorkflowRunID    int64      `gorm:"type:bigint;not null;default:0;index"`
	Status           string     `gorm:"type:varchar(30);not null;default:'';index"`
	PromptTokens     int        `gorm:"type:int;not null;default:0"`
	CompletionTokens int        `gorm:"type:int;not null;default:0"`
	StartedAt        time.Time  `gorm:"not null;index"`
	EndedAt          *time.Time `gorm:"index"`
	ErrorMessage     string     `gorm:"type:text"`
	TraceData        string     `gorm:"type:text"`
	CreatedAt        time.Time  `gorm:"not null;index"`
	UpdatedAt        time.Time  `gorm:"not null;index"`
}

// AgentStep records a normalized model, tool, workflow, or policy transition.
type AgentStep struct {
	ID            int64      `gorm:"primaryKey;autoIncrement"`
	AgentRunID    int64      `gorm:"type:bigint;not null;index"`
	WorkflowRunID int64      `gorm:"type:bigint;not null;default:0;index"`
	StepType      string     `gorm:"type:varchar(50);not null;default:'';index"`
	StepCode      string     `gorm:"type:varchar(100);not null;default:'';index"`
	Status        string     `gorm:"type:varchar(30);not null;default:'';index"`
	InputPreview  string     `gorm:"type:text"`
	OutputPreview string     `gorm:"type:text"`
	ErrorMessage  string     `gorm:"type:text"`
	StartedAt     time.Time  `gorm:"not null;index"`
	EndedAt       *time.Time `gorm:"index"`
	DurationMS    int        `gorm:"type:int;not null;default:0"`
	CreatedAt     time.Time  `gorm:"not null;index"`
}

// AgentToolCall records the safety-relevant details of a normalized tool call.
type AgentToolCall struct {
	ID               int64     `gorm:"primaryKey;autoIncrement"`
	AgentRunID       int64     `gorm:"type:bigint;not null;index"`
	AgentStepID      int64     `gorm:"type:bigint;not null;default:0;index"`
	ToolCode         string    `gorm:"type:varchar(150);not null;default:'';index"`
	RiskLevel        string    `gorm:"type:varchar(30);not null;default:'';index"`
	RequireConfirm   bool      `gorm:"not null;default:false"`
	Status           string    `gorm:"type:varchar(30);not null;default:'';index"`
	ArgumentsPreview string    `gorm:"type:text"`
	ResultPreview    string    `gorm:"type:text"`
	ErrorMessage     string    `gorm:"type:text"`
	DurationMS       int       `gorm:"type:int;not null;default:0"`
	CreatedAt        time.Time `gorm:"not null;index"`
}

// AgentRunQualityFeedback is an operator-provided quality review for one
// normalized Agent run. Runtime completion must not be treated as resolution.
type AgentRunQualityFeedback struct {
	ID               int64                          `gorm:"primaryKey;autoIncrement"`
	AgentRunID       int64                          `gorm:"type:bigint;not null;uniqueIndex"`
	ResolutionStatus enums.AgentRunResolutionStatus `gorm:"type:varchar(20);not null;default:'unknown';index"`
	EvidenceStatus   enums.AgentRunEvidenceStatus   `gorm:"type:varchar(20);not null;default:'unknown';index"`
	Comment          string                         `gorm:"type:text"`
	AuditFields
}

// AIWorkflow 表示客服 AI Agent 可编辑会话流程主表。
type AIWorkflow struct {
	ID                 int64        `gorm:"primaryKey;autoIncrement"`
	Name               string       `gorm:"type:varchar(100);not null;default:'';index"`
	Description        string       `gorm:"type:text"`
	Status             enums.Status `gorm:"type:int;not null;default:0;index"`
	DraftDefinition    string       `gorm:"type:text"`
	PublishedVersionID int64        `gorm:"type:bigint;not null;default:0;index"`
	SortNo             int          `gorm:"type:int;not null;default:0;index"`
	AuditFields
}

// AIWorkflowVersion 表示 AI 会话流程的不可变发布版本。
type AIWorkflowVersion struct {
	ID              int64        `gorm:"primaryKey;autoIncrement"`
	WorkflowID      int64        `gorm:"type:bigint;not null;default:0;index"`
	Version         int          `gorm:"type:int;not null;default:0;index"`
	Status          enums.Status `gorm:"type:int;not null;default:0;index"`
	Definition      string       `gorm:"type:text"`
	DefinitionHash  string       `gorm:"type:varchar(64);not null;default:'';index"`
	PublishedAt     *time.Time   `gorm:"index"`
	PublishedByID   int64        `gorm:"type:bigint;not null;default:0;index"`
	PublishedByName string       `gorm:"type:varchar(100);not null;default:''"`
	AuditFields
}

// AIAgentWorkflowBinding binds an Agent to one immutable, published workflow
// version. Workflows are maintained independently and can be reused by many
// Agents; Agent publication snapshots the binding set for reproducible runs.
type AIAgentWorkflowBinding struct {
	ID                 int64  `gorm:"primaryKey;autoIncrement"`
	AIAgentID          int64  `gorm:"type:bigint;not null;index;uniqueIndex:uk_agent_workflow_binding"`
	WorkflowID         int64  `gorm:"type:bigint;not null;index"`
	WorkflowVersionID  int64  `gorm:"type:bigint;not null;index;uniqueIndex:uk_agent_workflow_binding"`
	ToolName           string `gorm:"type:varchar(100);not null;default:''"`
	TriggerInstruction string `gorm:"type:text"`
	Priority           int    `gorm:"type:int;not null;default:0;index"`
	Enabled            bool   `gorm:"not null;default:true;index"`
	AuditFields
}

// AIWorkflowRun 表示一次会话 workflow 执行记录。
type AIWorkflowRun struct {
	ID                int64      `gorm:"primaryKey;autoIncrement"`
	WorkflowID        int64      `gorm:"type:bigint;not null;default:0;index"`
	WorkflowVersionID int64      `gorm:"type:bigint;not null;default:0;index"`
	ConversationID    int64      `gorm:"type:bigint;not null;default:0;index"`
	AIAgentID         int64      `gorm:"type:bigint;not null;default:0;index"`
	MessageID         int64      `gorm:"type:bigint;not null;default:0;index"`
	Status            int        `gorm:"type:int;not null;default:0;index"`
	StartedAt         time.Time  `gorm:"not null;index"`
	EndedAt           *time.Time `gorm:"index"`
	InterruptType     string     `gorm:"type:varchar(50);not null;default:'';index"`
	InterruptNodeID   string     `gorm:"type:varchar(100);not null;default:'';index"`
	ErrorMessage      string     `gorm:"type:text"`
	AuditFields
}

// AIWorkflowNodeRun 表示 workflow 执行中的单节点审计记录。
type AIWorkflowNodeRun struct {
	ID            int64      `gorm:"primaryKey;autoIncrement"`
	WorkflowRunID int64      `gorm:"type:bigint;not null;default:0;index"`
	NodeID        string     `gorm:"type:varchar(100);not null;default:'';index"`
	NodeType      string     `gorm:"type:varchar(50);not null;default:'';index"`
	Status        int        `gorm:"type:int;not null;default:0;index"`
	InputPreview  string     `gorm:"type:text"`
	OutputPreview string     `gorm:"type:text"`
	ErrorMessage  string     `gorm:"type:text"`
	StartedAt     time.Time  `gorm:"not null;index"`
	EndedAt       *time.Time `gorm:"index"`
	DurationMS    int        `gorm:"type:int;not null;default:0"`
}

// Channel 接入渠道配置。
//
//	用于统一描述系统的外部接入入口。不同渠道类型共享统一的接入配置骨架，
//	例如网页客服渠道（web）和企业微信客服渠道（wxwork_kf）。
//	渠道本身负责定义“入口如何识别、默认接入哪个 AI Agent、渠道专属配置是什么”，
//	而具体消息收发、会话映射等运行时数据由各自的渠道业务表承载。
type Channel struct {
	ID                            int64  `gorm:"primaryKey;autoIncrement"`                         // ID 为渠道主键。
	Name                          string `gorm:"type:varchar(100);not null;default:'';index"`      // Name 为渠道名称，用于后台展示和业务识别，例如“官网客服”“企业微信主客服”。
	ChannelType                   string `gorm:"type:varchar(30);not null;default:'';index"`       // ChannelType 为渠道类型，决定该渠道的接入方式和配置解释规则。当前规划的典型取值包括：web、wxwork_kf。
	ChannelID                     string `gorm:"type:varchar(64);not null;default:'';uniqueIndex"` // ChannelID 为渠道入口标识，由系统自动生成。对 web 渠道，该字段用于前端通过 X-Channel-Id 标识接入来源；对其他渠道，作为统一的系统内稳定渠道标识保留。
	AIAgentID                     int64  `gorm:"type:bigint;not null;default:0;index"`             // AIAgentID 为该渠道默认接入的 AI Agent。 当外部客户通过该渠道首次进入系统且尚未命中现有未结束会话时，系统会使用该 AI Agent 作为会话默认接待实例。
	AIAgentRolloutPercent         int    `gorm:"type:int;not null;default:100"`                    // AIAgentRolloutPercent 为该渠道对 AI 自动回复施加的灰度百分比，100 表示不额外限制。
	PreviousAIAgentRolloutPercent int    `gorm:"type:int;not null;default:0"`                      // PreviousAIAgentRolloutPercent 保存渠道上一次生效的 Agent 灰度比例，0 表示尚无可回滚值。
	// ConfigJSON 为渠道专属扩展配置，使用 JSON 存储。
	// 例如：
	// 1. web 渠道可记录允许域名、品牌配置等；
	// 2. wxwork_kf 渠道可记录 openKfId、欢迎语策略等。
	// 该字段只存储渠道类型私有配置，不承载通用主字段。
	ConfigJSON string       `gorm:"type:text"`
	Status     enums.Status `gorm:"type:int;not null;default:0;index"` // Status 为渠道状态。禁用后，该渠道不再允许新会话接入；删除时采用软删除状态保留历史关联数据。
	Remark     string       `gorm:"type:text"`                         // Remark 为渠道备注，用于记录接入说明、维护说明和内部运维信息。
	AuditFields
}

// ConversationEventLog 会话事件日志。
type ConversationEventLog struct {
	ID             int64              `gorm:"primaryKey;autoIncrement"`
	ConversationID int64              `gorm:"type:bigint;not null;index"`
	RequestID      string             `gorm:"type:varchar(128);not null;default:'';index"`
	EventType      enums.IMEventType  `gorm:"type:varchar(50);not null;default:'';index"`
	OperatorType   enums.IMSenderType `gorm:"type:varchar(30);not null;default:'';index"`
	OperatorID     int64              `gorm:"type:bigint;not null;default:0;index"`
	Content        string             `gorm:"type:text"`
	Payload        string             `gorm:"type:text"`
	CreatedAt      time.Time          `gorm:"not null;index"`
}

// Ticket 客服问题记录。
type Ticket struct {
	ID                int64              `gorm:"primaryKey;autoIncrement"`
	TicketNo          string             `gorm:"type:varchar(64);not null;default:'';uniqueIndex"`
	Title             string             `gorm:"type:varchar(255);not null;default:'';index"`
	Description       string             `gorm:"type:text"`
	Source            enums.TicketSource `gorm:"type:varchar(50);not null;default:'';index"`
	Channel           string             `gorm:"type:varchar(50);not null;default:'';index"`
	CustomerID        int64              `gorm:"type:bigint;not null;default:0;index"`
	ConversationID    int64              `gorm:"type:bigint;not null;default:0;index"`
	Status            enums.TicketStatus `gorm:"type:varchar(50);not null;default:'pending';index"`
	CurrentAssigneeID int64              `gorm:"type:bigint;not null;default:0;index"`
	HandledAt         *time.Time         `gorm:"index"`
	AuditFields
}

// TicketTag 工单标签关联。
type TicketTag struct {
	ID       int64 `gorm:"primaryKey;autoIncrement"`
	TicketID int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_ticket_tag"`
	TagID    int64 `gorm:"type:bigint;not null;index;uniqueIndex:uk_ticket_tag"`
	AuditFields
}

// TicketProgress 工单处理进展。
type TicketProgress struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	TicketID  int64     `gorm:"type:bigint;not null;index"`
	Content   string    `gorm:"type:text"`
	AuthorID  int64     `gorm:"type:bigint;not null;default:0;index"`
	CreatedAt time.Time `gorm:"not null;index"`
}

// AgentProfile 客服档案。
type AgentProfile struct {
	ID                    int64               `gorm:"primaryKey;autoIncrement"`                         // ID 为客服档案主键。
	UserID                int64               `gorm:"type:bigint;not null;uniqueIndex"`                 // UserID 关联后台用户，一名用户只允许一份客服档案。
	TeamID                int64               `gorm:"type:bigint;not null;default:0;index"`             // TeamID 为客服所属客服组。
	AgentCode             string              `gorm:"type:varchar(64);not null;default:'';uniqueIndex"` // AgentCode 为客服工号，用于业务侧识别客服。
	DisplayName           string              `gorm:"type:varchar(100);not null;default:'';index"`      // DisplayName 为客服展示名，可区别于后台昵称。
	Avatar                string              `gorm:"type:varchar(1024);not null;default:''"`           // Avatar 为客服头像 URL。
	ServiceStatus         enums.ServiceStatus `gorm:"type:int;not null;default:0;index"`                // ServiceStatus 表示客服服务状态：0空闲 1忙碌。
	MaxConcurrentCount    int                 `gorm:"type:int;not null;default:0"`                      // MaxConcurrentCount 表示客服最大并发接待数。
	PriorityLevel         int                 `gorm:"type:int;not null;default:0;index"`                // PriorityLevel 表示自动分配优先级，值越大越优先。
	AutoAssignEnabled     bool                `gorm:"not null;default:true;index"`                      // AutoAssignEnabled 表示是否参与自动分配。
	ReceiveOfflineMessage bool                `gorm:"not null;default:false"`                           // ReceiveOfflineMessage 表示离线时是否仍接收离线消息或转接消息。
	LastOnlineAt          *time.Time          `gorm:"index"`                                            // LastOnlineAt 记录最近一次在线时间。
	LastStatusAt          *time.Time          `gorm:"index"`                                            // LastStatusAt 记录最近一次状态变更时间。
	Status                enums.Status        `gorm:"type:int;not null;default:0;index"`                // Status 表示客服档案状态
	Remark                string              `gorm:"type:text"`                                        // Remark 记录客服备注信息。
	AuditFields
}

// AgentTeam 客服组。
type AgentTeam struct {
	ID           int64        `gorm:"primaryKey;autoIncrement"`                    // ID 为客服组主键。
	Name         string       `gorm:"type:varchar(100);not null;default:'';index"` // Name 为客服组名称。
	LeaderUserID int64        `gorm:"type:bigint;not null;default:0;index"`        // LeaderUserID 为组长用户ID，0 表示暂未设置。
	Status       enums.Status `gorm:"type:int;not null;default:0;index"`           // Status 表示客服组状态
	Description  string       `gorm:"type:varchar(255);not null;default:''"`       // Description 为客服组简介，用于说明职责边界。
	Remark       string       `gorm:"type:text"`                                   // Remark 记录客服组内部备注。
	AuditFields
}

// AgentTeamSchedule 客服组排班。
type AgentTeamSchedule struct {
	ID      int64        `gorm:"primaryKey;autoIncrement"`              // ID 为组排班主键。
	TeamID  int64        `gorm:"type:bigint;not null;index"`            // TeamID 为被排班的客服组ID。
	StartAt time.Time    `gorm:"not null;index"`                        // StartAt 为班次开始时间。
	EndAt   time.Time    `gorm:"not null;index"`                        // EndAt 为班次结束时间。
	Remark  string       `gorm:"type:varchar(255);not null;default:''"` // Remark 记录排班备注。
	Status  enums.Status `gorm:"type:int;not null;default:0;index"`     // Status 表示组排班记录状态。
	AuditFields
}

// AIConfig AI 统一配置。
// 一条记录表示一个可直接调用的 AI 配置实例，
// 同时包含厂商接入信息、模型信息和调用参数，不再拆分 endpoint/model 两层概念。
type AIConfig struct {
	ID               int64             `gorm:"primaryKey;autoIncrement"`                    // ID 为配置主键。
	Name             string            `gorm:"type:varchar(100);not null;default:'';index"` // Name 为配置名称，用于后台识别和展示。
	Provider         enums.AIProvider  `gorm:"type:varchar(50);not null;default:'';index"`  // Provider 为供应商标识，例如 openai、azure_openai、dashscope。
	BaseURL          string            `gorm:"type:varchar(255);not null;default:''"`       // BaseURL 为模型服务基础地址，例如 https://api.openai.com/v1。
	APIKey           string            `gorm:"type:varchar(255);not null;default:''"`       // APIKey 为服务端请求模型接口所需密钥。
	ModelType        enums.AIModelType `gorm:"type:varchar(30);not null;default:'';index"`  // ModelType 为模型类型，例如 llm、embedding、rerank。
	ModelName        string            `gorm:"type:varchar(100);not null;default:'';index"` // ModelName 为实际请求时传给上游的模型名。
	Dimension        int               `gorm:"type:int;not null;default:0"`                 // Dimension 为向量维度，仅 embedding 模型通常需要填写。
	MaxContextTokens int               `gorm:"type:int;not null;default:0"`                 // MaxContextTokens 为模型支持的最大上下文 token 数。
	MaxOutputTokens  int               `gorm:"type:int;not null;default:0"`                 // MaxOutputTokens 为模型建议的最大输出 token 数。
	TimeoutMS        int               `gorm:"type:int;not null;default:30000"`             // TimeoutMS 为调用该配置的默认超时时间，单位毫秒。
	MaxRetryCount    int               `gorm:"type:int;not null;default:0"`                 // MaxRetryCount 为默认最大重试次数。
	RPMLimit         int               `gorm:"type:int;not null;default:0"`                 // RPMLimit 为每分钟请求数限制，0 表示未显式配置。
	TPMLimit         int               `gorm:"type:int;not null;default:0"`                 // TPMLimit 为每分钟 token 数限制，0 表示未显式配置。
	Status           enums.Status      `gorm:"type:int;not null;index"`                     // Status 状态；同一 modelType 仅允许一条启用记录。
	SortNo           int               `gorm:"type:int;not null;index"`                     // SortNo 为排序号，用于后台展示和人工调整顺序。
	Remark           string            `gorm:"type:text"`                                   // Remark 为备注，用于记录用途、成本、限制和切换说明等补充信息。
	AuditFields
}

// KnowledgeBase 知识库主表。
type KnowledgeBase struct {
	ID                    int64        `gorm:"primaryKey;autoIncrement"`                           // ID 为知识库主键。
	Name                  string       `gorm:"type:varchar(100);not null;default:'';index"`        // Name 为知识库名称。
	Description           string       `gorm:"type:text"`                                          // Description 为知识库描述。
	KnowledgeType         string       `gorm:"type:varchar(20);not null;default:'document';index"` // KnowledgeType 为知识库类型：document/faq。
	Status                enums.Status `gorm:"type:int;not null;index"`                            // Status 为状态
	DefaultTopK           int          `gorm:"type:int;not null;default:10"`                       // DefaultTopK 为默认召回数量。
	DefaultScoreThreshold float64      `gorm:"type:decimal(5,4);not null;default:0.5"`             // DefaultScoreThreshold 为默认相似度阈值。
	DefaultRerankLimit    int          `gorm:"type:int;not null;default:5"`                        // DefaultRerankLimit 为默认重排后保留数量。
	ChunkProvider         string       `gorm:"type:varchar(30);not null;default:'structured'"`     // ChunkProvider 为知识库分块策略 provider。
	ChunkTargetTokens     int          `gorm:"type:int;not null;default:300"`                      // ChunkTargetTokens 为目标 chunk token 数。
	ChunkMaxTokens        int          `gorm:"type:int;not null;default:400"`                      // ChunkMaxTokens 为单 chunk 最大 token 数。
	ChunkOverlapTokens    int          `gorm:"type:int;not null;default:40"`                       // ChunkOverlapTokens 为相邻 chunk 重叠 token 数。
	AnswerMode            int          `gorm:"type:int;not null;default:1"`                        // AnswerMode 为回答模式：1严格知识库模式 2辅助解释模式。
	SortNo                int          `gorm:"type:int;not null;default:0;index"`                  // SortNo 为排序号，用于后台展示和知识库的人工排序管理。
	Remark                string       `gorm:"type:text"`                                          // Remark 为备注。
	AuditFields
}

// KnowledgeDirectory 知识库内部目录表。
type KnowledgeDirectory struct {
	ID              int64        `gorm:"primaryKey;autoIncrement"`                                // ID 为目录主键。
	KnowledgeBaseID int64        `gorm:"type:bigint;not null;index:idx_kb_parent_sort"`           // KnowledgeBaseID 为所属知识库 ID。
	ParentID        int64        `gorm:"type:bigint;not null;default:0;index:idx_kb_parent_sort"` // ParentID 为父目录 ID，0 表示一级目录。
	Name            string       `gorm:"type:varchar(100);not null;default:'';index"`             // Name 为目录名称。
	SortNo          int          `gorm:"type:int;not null;default:0;index:idx_kb_parent_sort"`    // SortNo 为同级排序号。
	Status          enums.Status `gorm:"type:int;not null;default:0;index"`                       // Status 为状态。
	Remark          string       `gorm:"type:text"`                                               // Remark 为备注。
	AuditFields
}

// KnowledgeDocument 知识文档主表。
type KnowledgeDocument struct {
	ID              int64                              `gorm:"primaryKey;autoIncrement"`                          // ID 为文档主键。
	KnowledgeBaseID int64                              `gorm:"type:bigint;not null;index"`                        // KnowledgeBaseID 为所属知识库ID。
	DirectoryID     int64                              `gorm:"type:bigint;not null;default:0;index"`              // DirectoryID 为所属知识库内部目录 ID，0 表示根目录。
	Title           string                             `gorm:"type:varchar(255);not null;default:'';index"`       // Title 为文档标题。
	ContentType     enums.KnowledgeDocumentContentType `gorm:"type:varchar(20);not null;default:'html'"`          // ContentType 为内容类型：html/markdown。
	Content         string                             `gorm:"type:text"`                                         // Content 为文档内容。
	Status          enums.Status                       `gorm:"type:int;not null;default:0;index"`                 // Status 为状态
	IndexStatus     enums.KnowledgeDocumentIndexStatus `gorm:"type:varchar(20);not null;default:'pending';index"` // IndexStatus 为索引状态：pending/indexed/failed。
	IndexedAt       *time.Time                         `gorm:"index"`                                             // IndexedAt 为最近一次索引成功时间。
	IndexError      string                             `gorm:"type:text"`                                         // IndexError 为最近一次索引失败信息。
	ContentHash     string                             `gorm:"type:varchar(64);not null;default:'';index"`        // ContentHash 为内容哈希，用于变更检测。
	AuditFields
}

// KnowledgeFAQ FAQ 条目主表。
type KnowledgeFAQ struct {
	ID               int64                              `gorm:"primaryKey;autoIncrement"`                          // ID 为 FAQ 主键。
	KnowledgeBaseID  int64                              `gorm:"type:bigint;not null;index"`                        // KnowledgeBaseID 为所属 FAQ 知识库 ID。
	DirectoryID      int64                              `gorm:"type:bigint;not null;default:0;index"`              // DirectoryID 为所属知识库内部目录 ID，0 表示根目录。
	Question         string                             `gorm:"type:varchar(500);not null;default:'';index"`       // Question 为标准问题。
	Answer           string                             `gorm:"type:text"`                                         // Answer 为标准答案。
	SimilarQuestions string                             `gorm:"type:text"`                                         // SimilarQuestions 为相似问 JSON 数组。
	Status           enums.Status                       `gorm:"type:int;not null;default:0;index"`                 // Status 为状态。
	IndexStatus      enums.KnowledgeDocumentIndexStatus `gorm:"type:varchar(20);not null;default:'pending';index"` // IndexStatus 为索引状态：pending/indexed/failed。
	IndexedAt        *time.Time                         `gorm:"index"`                                             // IndexedAt 为最近一次索引成功时间。
	IndexError       string                             `gorm:"type:text"`                                         // IndexError 为最近一次索引失败信息。
	Remark           string                             `gorm:"type:text"`                                         // Remark 为备注。
	AuditFields
}

// KnowledgeChunk 切片元数据表。
type KnowledgeChunk struct {
	ID              int64        `gorm:"primaryKey;autoIncrement"`                    // ID 为切片主键。
	KnowledgeBaseID int64        `gorm:"type:bigint;not null;index"`                  // KnowledgeBaseID 为知识库ID。
	DocumentID      int64        `gorm:"type:bigint;not null;default:0;index"`        // DocumentID 为文档ID。
	FaqID           int64        `gorm:"type:bigint;not null;default:0;index"`        // FaqID 为 FAQ ID。
	ChunkNo         int          `gorm:"type:int;not null;default:0;index"`           // ChunkNo 为切片序号。
	Title           string       `gorm:"type:varchar(255);not null;default:''"`       // Title 为切片标题。
	Content         string       `gorm:"type:text"`                                   // Content 为切片内容。
	ContentHash     string       `gorm:"type:varchar(64);not null;default:'';index"`  // ContentHash 为内容哈希。
	CharCount       int          `gorm:"type:int;not null;default:0"`                 // CharCount 为字符数。
	TokenCount      int          `gorm:"type:int;not null;default:0"`                 // TokenCount 为token数。
	ChunkType       string       `gorm:"type:varchar(30);not null;default:''"`        // ChunkType 为切片类型。
	SectionPath     string       `gorm:"type:text"`                                   // SectionPath 为章节路径。
	Provider        string       `gorm:"type:varchar(30);not null;default:''"`        // Provider 为分块 provider。
	Status          enums.Status `gorm:"type:int;not null;default:0;index"`           // Status 为状态：1有效 2已删除。
	VectorID        string       `gorm:"type:varchar(100);not null;default:'';index"` // VectorID 为向量库中的point ID。
	CreatedAt       time.Time    `gorm:"not null;index"`
	UpdatedAt       time.Time    `gorm:"not null;index"`
}

// DocPage is both a document-center navigation node and a content page.
type DocPage struct {
	ID                        int64               `gorm:"primaryKey;autoIncrement"`
	ParentID                  int64               `gorm:"type:bigint;not null;default:0;index;uniqueIndex:idx_doc_page_parent_slug"`
	Title                     string              `gorm:"type:varchar(255);not null;default:'';index"`
	Slug                      string              `gorm:"type:varchar(160);not null;default:'';uniqueIndex:idx_doc_page_parent_slug"`
	Summary                   string              `gorm:"type:text"`
	ContentType               string              `gorm:"type:varchar(20);not null;default:'markdown'"`
	Content                   string              `gorm:"type:text"`
	CoverURL                  string              `gorm:"type:varchar(255);not null;default:''"`
	TagsJSON                  string              `gorm:"type:text"`
	Status                    enums.DocPageStatus `gorm:"type:varchar(20);not null;default:'draft';index"`
	SortNo                    int                 `gorm:"type:int;not null;default:0;index"`
	ViewCount                 int64               `gorm:"type:bigint;not null;default:0"`
	HelpfulCount              int64               `gorm:"type:bigint;not null;default:0"`
	UnhelpfulCount            int64               `gorm:"type:bigint;not null;default:0"`
	PublishedAt               *time.Time          `gorm:"index"`
	SyncedKnowledgeDocumentID int64               `gorm:"type:bigint;not null;default:0;index"`
	Remark                    string              `gorm:"type:text"`
	AuditFields
}

// Category groups public FAQ community posts.
type Category struct {
	ID          int64        `gorm:"primaryKey;autoIncrement"`
	Name        string       `gorm:"type:varchar(100);not null;default:'';index"`
	Slug        string       `gorm:"type:varchar(120);not null;default:'';uniqueIndex"`
	Description string       `gorm:"type:text"`
	SortNo      int          `gorm:"type:int;not null;default:0;index"`
	Status      enums.Status `gorm:"type:int;not null;default:0;index"`
	Remark      string       `gorm:"type:text"`
	AuditFields
}

// Post is a logged-in-customer post in the public FAQ community.
type Post struct {
	ID                  int64                   `gorm:"primaryKey;autoIncrement"`
	CategoryID          int64                   `gorm:"type:bigint;not null;default:0;index"`
	UserID              int64                   `gorm:"type:bigint;not null;index"`
	Title               string                  `gorm:"type:varchar(255);not null;default:'';index"`
	ContentType         string                  `gorm:"type:varchar(20);not null;default:'markdown'"`
	Content             string                  `gorm:"type:text"`
	TagsJSON            string                  `gorm:"type:text"`
	Status              enums.PostStatus        `gorm:"type:varchar(20);not null;default:'normal';index"`
	AcceptedCommentID   int64                   `gorm:"type:bigint;not null;default:0;index"`
	CommentCount        int64                   `gorm:"type:bigint;not null;default:0"`
	ReactionCount       int64                   `gorm:"type:bigint;not null;default:0"`
	ViewCount           int64                   `gorm:"type:bigint;not null;default:0"`
	LastCommentedAt     *time.Time              `gorm:"index"`
	LastCommentUserType enums.CommentAuthorType `gorm:"type:varchar(20);not null;default:''"`
	LastCommentUserID   int64                   `gorm:"type:bigint;not null;default:0"`
	AuditFields
}

// Comment is a comment from either a customer or a backend user.
type Comment struct {
	ID            int64                   `gorm:"primaryKey;autoIncrement"`
	PostID        int64                   `gorm:"type:bigint;not null;index"`
	ParentID      int64                   `gorm:"type:bigint;not null;default:0;index"`
	AuthorType    enums.CommentAuthorType `gorm:"type:varchar(20);not null;default:'customer';index"`
	AuthorID      int64                   `gorm:"type:bigint;not null;index"`
	ContentType   string                  `gorm:"type:varchar(20);not null;default:'markdown'"`
	Content       string                  `gorm:"type:text"`
	Status        enums.CommentStatus     `gorm:"type:varchar(20);not null;default:'normal';index"`
	ReactionCount int64                   `gorm:"type:bigint;not null;default:0"`
	ReplyCount    int64                   `gorm:"type:bigint;not null;default:0"`
	ReportCount   int64                   `gorm:"type:bigint;not null;default:0"`
	IsAccepted    bool                    `gorm:"not null;default:false;index"`
	AuditFields
}

type Reaction struct {
	ID           int64                `gorm:"primaryKey;autoIncrement"`
	TargetType   enums.ReactionTarget `gorm:"type:varchar(20);not null;index;uniqueIndex:uk_reaction"`
	TargetID     int64                `gorm:"type:bigint;not null;index;uniqueIndex:uk_reaction"`
	UserID       int64                `gorm:"type:bigint;not null;index;uniqueIndex:uk_reaction"`
	ReactionType enums.ReactionType   `gorm:"type:varchar(20);not null;default:'like';uniqueIndex:uk_reaction"`
	CreatedAt    time.Time            `gorm:"not null;index"`
	UpdatedAt    time.Time            `gorm:"not null;index"`
}

type CommentReport struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	CommentID int64     `gorm:"type:bigint;not null;index;uniqueIndex:uk_comment_report"`
	UserID    int64     `gorm:"type:bigint;not null;index;uniqueIndex:uk_comment_report"`
	Reason    string    `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt time.Time `gorm:"not null;index"`
}

// KnowledgeRetrieveLog 检索日志表。
type KnowledgeRetrieveLog struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement"`                   // ID 为日志主键。
	KnowledgeBaseID    int64     `gorm:"type:bigint;not null;index"`                 // KnowledgeBaseID 为知识库ID。
	Channel            string    `gorm:"type:varchar(30);not null;default:'';index"` // Channel 为渠道：im会话, agent_assist坐席辅助, api开放接口, debug调试。
	Scene              string    `gorm:"type:varchar(50);not null;default:'';index"` // Scene 为场景：first_response首响, assist辅助, qa问答。
	SessionID          string    `gorm:"type:varchar(64);not null;default:'';index"` // SessionID 为会话ID。
	ConversationID     int64     `gorm:"type:bigint;not null;default:0;index"`       // ConversationID 为会话ID。
	RequestID          string    `gorm:"type:varchar(64);not null;default:'';index"` // RequestID 为请求ID。
	Question           string    `gorm:"type:text"`                                  // Question 为原始问题。
	RewriteQuestion    string    `gorm:"type:text"`                                  // RewriteQuestion 为改写后问题。
	Answer             string    `gorm:"type:text"`                                  // Answer 为生成的答案。
	AnswerStatus       int       `gorm:"type:int;not null;default:1;index"`          // AnswerStatus 为答案状态：1正常 2无答案 3兜底 4风控拦截。
	HitCount           int       `gorm:"type:int;not null;default:0"`                // HitCount 为命中数量。
	TopScore           float64   `gorm:"type:decimal(5,4);not null;default:0"`       // TopScore 为最高相似度分数。
	ChunkProvider      string    `gorm:"type:varchar(30);not null;default:'';index"` // ChunkProvider 为分块 provider。
	ChunkTargetTokens  int       `gorm:"type:int;not null;default:0"`                // ChunkTargetTokens 为目标 token 数。
	ChunkMaxTokens     int       `gorm:"type:int;not null;default:0"`                // ChunkMaxTokens 为最大 token 数。
	ChunkOverlapTokens int       `gorm:"type:int;not null;default:0"`                // ChunkOverlapTokens 为重叠 token 数。
	RerankEnabled      bool      `gorm:"not null;default:false;index"`               // RerankEnabled 是否启用 rerank。
	RerankLimit        int       `gorm:"type:int;not null;default:0"`                // RerankLimit 为 rerank 条数。
	CitationCount      int       `gorm:"type:int;not null;default:0"`                // CitationCount 为最终引用条数。
	UsedChunkCount     int       `gorm:"type:int;not null;default:0"`                // UsedChunkCount 为进入上下文的 chunk 数。
	LatencyMs          int64     `gorm:"type:bigint;not null;default:0"`             // LatencyMs 为总耗时毫秒。
	RetrieveMs         int64     `gorm:"type:bigint;not null;default:0"`             // RetrieveMs 为检索耗时毫秒。
	GenerateMs         int64     `gorm:"type:bigint;not null;default:0"`             // GenerateMs 为生成耗时毫秒。
	PromptTokens       int       `gorm:"type:int;not null;default:0"`                // PromptTokens 为prompt token数。
	CompletionTokens   int       `gorm:"type:int;not null;default:0"`                // CompletionTokens 为completion token数。
	ModelName          string    `gorm:"type:varchar(100);not null;default:''"`      // ModelName 为使用的模型名称。
	TraceData          string    `gorm:"type:text"`                                  // TraceData 为链路追踪数据JSON。
	CreatedAt          time.Time `gorm:"not null;index"`
}

// KnowledgeRetrieveHit 检索命中详情表。
type KnowledgeRetrieveHit struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`              // ID 为命中记录主键。
	RetrieveLogID   int64     `gorm:"type:bigint;not null;index"`            // RetrieveLogID 为检索日志ID。
	KnowledgeBaseID int64     `gorm:"type:bigint;not null;default:0;index"`  // KnowledgeBaseID 为命中来源知识库ID。
	ChunkID         int64     `gorm:"type:bigint;not null;index"`            // ChunkID 为切片ID。
	DocumentID      int64     `gorm:"type:bigint;not null;index"`            // DocumentID 为文档ID。
	DocumentTitle   string    `gorm:"type:varchar(255);not null;default:''"` // DocumentTitle 为文档标题。
	FaqID           int64     `gorm:"type:bigint;not null;default:0;index"`  // FaqID 为 FAQ ID。
	FaqQuestion     string    `gorm:"type:varchar(500);not null;default:''"` // FaqQuestion 为 FAQ 问题。
	ChunkNo         int       `gorm:"type:int;not null;default:0"`           // ChunkNo 为切片序号。
	Title           string    `gorm:"type:varchar(255);not null;default:''"` // Title 为切片标题。
	SectionPath     string    `gorm:"type:text"`                             // SectionPath 为章节路径。
	ChunkType       string    `gorm:"type:varchar(30);not null;default:''"`  // ChunkType 为切片类型。
	Provider        string    `gorm:"type:varchar(30);not null;default:''"`  // Provider 为分块 provider。
	RankNo          int       `gorm:"type:int;not null;default:0"`           // RankNo 为排名。
	Score           float64   `gorm:"type:decimal(5,4);not null;default:0"`  // Score 为相似度分数。
	RerankScore     float64   `gorm:"type:decimal(5,4);not null;default:0"`  // RerankScore 为重排分数。
	UsedInAnswer    bool      `gorm:"not null;default:false"`                // UsedInAnswer 是否用于生成答案。
	IsCitation      bool      `gorm:"not null;default:false"`                // IsCitation 是否作为引用返回。
	Snippet         string    `gorm:"type:text"`                             // Snippet 为内容片段。
	CreatedAt       time.Time `gorm:"not null;index"`
}

// KnowledgeFeedback 问答反馈表。
type KnowledgeFeedback struct {
	ID             int64     `gorm:"primaryKey;autoIncrement"`              // ID 为反馈主键。
	RetrieveLogID  int64     `gorm:"type:bigint;not null;index"`            // RetrieveLogID 为检索日志ID。
	FeedbackType   int       `gorm:"type:int;not null;default:1;index"`     // FeedbackType 为反馈类型：1点赞 2点踩 3无帮助 4引用错误 5其他。
	FeedbackReason string    `gorm:"type:varchar(500);not null;default:''"` // FeedbackReason 为反馈原因。
	UserID         int64     `gorm:"type:bigint;not null;default:0;index"`  // UserID 为用户ID。
	AgentID        int64     `gorm:"type:bigint;not null;default:0;index"`  // AgentID 为坐席ID。
	Remark         string    `gorm:"type:text"`                             // Remark 为备注。
	CreatedAt      time.Time `gorm:"not null;index"`
}

// SkillDefinition 表示可由后台配置并参与运行时路由的 Skill 定义。
type SkillDefinition struct {
	ID            int64        `gorm:"primaryKey;autoIncrement"`                    // ID 为 Skill 主键。
	Name          string       `gorm:"type:varchar(100);not null;default:'';index"` // Name 为 Skill 的展示名称，用于后台列表、配置页和人工选择场景。
	Description   string       `gorm:"type:varchar(255);not null;default:''"`       // Description 为 Skill 的简要说明，用于描述该 Skill 的适用场景和职责边界。
	Instruction   string       `gorm:"type:text"`                                   // Instruction 为 Skill 的主体说明文档存储字段，使用 Markdown 编写，供 Agent 理解任务目标、步骤和工具使用要求。
	Examples      string       `gorm:"type:text"`                                   // Examples 为示例问法 JSON 数组字符串。
	ToolWhitelist string       `gorm:"type:text"`                                   // ToolWhitelist 为允许使用的工具编码 JSON 数组字符串。
	Status        enums.Status `gorm:"type:int;not null;default:0;index"`           // Status 为 Skill 当前状态，使用全局通用状态：0启用 1禁用 2删除。
	Remark        string       `gorm:"type:text"`                                   // Remark 为后台备注，用于记录配置说明、维护信息或内部协作信息。
	AuditFields
}

// ConversationInterrupt 表示会话级待恢复中断记录。
type ConversationInterrupt struct {
	ID                  int64      `gorm:"primaryKey;autoIncrement"`
	ConversationID      int64      `gorm:"type:bigint;not null;default:0;index"`
	AIAgentID           int64      `gorm:"type:bigint;not null;default:0;index"`
	AgentRunID          int64      `gorm:"type:bigint;not null;default:0;index"`
	AgentStepID         int64      `gorm:"type:bigint;not null;default:0;index"`
	SourceMessageID     int64      `gorm:"type:bigint;not null;default:0;index"`
	LastResumeMessageID int64      `gorm:"type:bigint;not null;default:0;index"`
	WorkflowRunID       int64      `gorm:"type:bigint;not null;default:0;index"`
	WorkflowNodeID      string     `gorm:"type:varchar(100);not null;default:'';index"`
	CheckPointID        string     `gorm:"type:varchar(128);not null;default:'';uniqueIndex"`
	InterruptID         string     `gorm:"type:varchar(255);not null;default:'';index"`
	InterruptType       string     `gorm:"type:varchar(50);not null;default:'';index"`
	Status              string     `gorm:"type:varchar(30);not null;default:'';index"`
	PromptText          string     `gorm:"type:text"`
	RequestData         string     `gorm:"type:text"`
	CheckPointData      string     `gorm:"type:text"`
	ResumeCount         int        `gorm:"type:int;not null;default:0"`
	ExpiresAt           *time.Time `gorm:"index"`
	CreatedAt           time.Time  `gorm:"not null;index"`
	UpdatedAt           time.Time  `gorm:"not null;index"`
}
