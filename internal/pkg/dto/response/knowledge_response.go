package response

import (
	"agent-desk/internal/pkg/enums"
	"time"
)

type KnowledgeBaseResponse struct {
	ID                    int64        `json:"id"`
	Name                  string       `json:"name"`
	Description           string       `json:"description"`
	KnowledgeType         string       `json:"knowledgeType"`
	KnowledgeTypeName     string       `json:"knowledgeTypeName"`
	Status                enums.Status `json:"status"`
	StatusName            string       `json:"statusName"`
	DefaultTopK           int          `json:"defaultTopK"`
	DefaultScoreThreshold float64      `json:"defaultScoreThreshold"`
	DefaultRerankLimit    int          `json:"defaultRerankLimit"`
	ChunkProvider         string       `json:"chunkProvider"`
	ChunkTargetTokens     int          `json:"chunkTargetTokens"`
	ChunkMaxTokens        int          `json:"chunkMaxTokens"`
	ChunkOverlapTokens    int          `json:"chunkOverlapTokens"`
	AnswerMode            int          `json:"answerMode"`
	AnswerModeName        string       `json:"answerModeName"`
	DocumentCount         int64        `json:"documentCount"`
	FAQCount              int64        `json:"faqCount"`
	Remark                string       `json:"remark"`
	CreatedAt             time.Time    `json:"createdAt"`
	UpdatedAt             time.Time    `json:"updatedAt"`
	CreateUserName        string       `json:"createUserName"`
	UpdateUserName        string       `json:"updateUserName"`
}

type KnowledgeDocumentResponse struct {
	ID                int64                              `json:"id"`
	KnowledgeBaseID   int64                              `json:"knowledgeBaseId"`
	KnowledgeBaseName string                             `json:"knowledgeBaseName,omitempty"`
	DirectoryID       int64                              `json:"directoryId"`
	DirectoryName     string                             `json:"directoryName,omitempty"`
	DirectoryPath     string                             `json:"directoryPath,omitempty"`
	Title             string                             `json:"title"`
	ContentType       enums.KnowledgeDocumentContentType `json:"contentType"`
	Content           string                             `json:"content"`
	Status            enums.Status                       `json:"status"`
	StatusName        string                             `json:"statusName"`
	IndexStatus       enums.KnowledgeDocumentIndexStatus `json:"indexStatus"`
	IndexStatusName   string                             `json:"indexStatusName"`
	IndexedAt         *time.Time                         `json:"indexedAt"`
	IndexError        string                             `json:"indexError"`
	ContentHash       string                             `json:"contentHash"`
	CreatedAt         time.Time                          `json:"createdAt"`
	UpdatedAt         time.Time                          `json:"updatedAt"`
	CreateUserName    string                             `json:"createUserName"`
	UpdateUserName    string                             `json:"updateUserName"`
}

type KnowledgeDocumentListResponse struct {
	ID                int64                              `json:"id"`
	KnowledgeBaseID   int64                              `json:"knowledgeBaseId"`
	KnowledgeBaseName string                             `json:"knowledgeBaseName,omitempty"`
	DirectoryID       int64                              `json:"directoryId"`
	DirectoryName     string                             `json:"directoryName,omitempty"`
	DirectoryPath     string                             `json:"directoryPath,omitempty"`
	Title             string                             `json:"title"`
	ContentType       enums.KnowledgeDocumentContentType `json:"contentType"`
	Status            enums.Status                       `json:"status"`
	StatusName        string                             `json:"statusName"`
	IndexStatus       enums.KnowledgeDocumentIndexStatus `json:"indexStatus"`
	IndexStatusName   string                             `json:"indexStatusName"`
	IndexedAt         *time.Time                         `json:"indexedAt"`
	IndexError        string                             `json:"indexError"`
	ContentHash       string                             `json:"contentHash"`
	CreatedAt         time.Time                          `json:"createdAt"`
	UpdatedAt         time.Time                          `json:"updatedAt"`
	CreateUserName    string                             `json:"createUserName"`
	UpdateUserName    string                             `json:"updateUserName"`
}

type KnowledgeFAQResponse struct {
	ID                int64                              `json:"id"`
	KnowledgeBaseID   int64                              `json:"knowledgeBaseId"`
	KnowledgeBaseName string                             `json:"knowledgeBaseName,omitempty"`
	DirectoryID       int64                              `json:"directoryId"`
	DirectoryName     string                             `json:"directoryName,omitempty"`
	DirectoryPath     string                             `json:"directoryPath,omitempty"`
	Question          string                             `json:"question"`
	Answer            string                             `json:"answer"`
	SimilarQuestions  []string                           `json:"similarQuestions"`
	Status            enums.Status                       `json:"status"`
	StatusName        string                             `json:"statusName"`
	IndexStatus       enums.KnowledgeDocumentIndexStatus `json:"indexStatus"`
	IndexStatusName   string                             `json:"indexStatusName"`
	IndexedAt         *time.Time                         `json:"indexedAt"`
	IndexError        string                             `json:"indexError"`
	Remark            string                             `json:"remark"`
	CreatedAt         time.Time                          `json:"createdAt"`
	UpdatedAt         time.Time                          `json:"updatedAt"`
	CreateUserName    string                             `json:"createUserName"`
	UpdateUserName    string                             `json:"updateUserName"`
}

type KnowledgeDirectoryResponse struct {
	ID              int64                        `json:"id"`
	KnowledgeBaseID int64                        `json:"knowledgeBaseId"`
	ParentID        int64                        `json:"parentId"`
	Name            string                       `json:"name"`
	SortNo          int                          `json:"sortNo"`
	Status          enums.Status                 `json:"status"`
	StatusName      string                       `json:"statusName"`
	Remark          string                       `json:"remark"`
	CreatedAt       time.Time                    `json:"createdAt"`
	UpdatedAt       time.Time                    `json:"updatedAt"`
	CreateUserName  string                       `json:"createUserName"`
	UpdateUserName  string                       `json:"updateUserName"`
	Children        []KnowledgeDirectoryResponse `json:"children"`
}

type KnowledgeFAQExportedFile struct {
	Filename    string
	ContentType string
	Data        []byte
}

type KnowledgeFAQImportError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type KnowledgeFAQImportResult struct {
	Total   int                       `json:"total"`
	Created int                       `json:"created"`
	Updated int                       `json:"updated"`
	Skipped int                       `json:"skipped"`
	Failed  int                       `json:"failed"`
	Errors  []KnowledgeFAQImportError `json:"errors"`
}

type KnowledgeSearchResult struct {
	KnowledgeBaseID int64   `json:"knowledgeBaseId"`
	ChunkID         int64   `json:"chunkId"`
	DocumentID      int64   `json:"documentId"`
	DocumentTitle   string  `json:"documentTitle"`
	FaqID           int64   `json:"faqId"`
	FaqQuestion     string  `json:"faqQuestion"`
	ChunkNo         int     `json:"chunkNo"`
	Title           string  `json:"title"`
	SectionPath     string  `json:"sectionPath"`
	Content         string  `json:"content"`
	Score           float64 `json:"score"`
	RerankScore     float64 `json:"rerankScore"`
}

type KnowledgeSearchResponse struct {
	Question  string                  `json:"question"`
	Results   []KnowledgeSearchResult `json:"results"`
	HitCount  int                     `json:"hitCount"`
	LatencyMs int64                   `json:"latencyMs"`
}

type KnowledgeAnswerResponse struct {
	Question         string                  `json:"question"`
	RewriteQuestion  string                  `json:"rewriteQuestion,omitempty"`
	Answer           string                  `json:"answer"`
	AnswerStatus     int                     `json:"answerStatus"`
	AnswerStatusName string                  `json:"answerStatusName"`
	Citations        []KnowledgeCitation     `json:"citations"`
	Hits             []KnowledgeSearchResult `json:"hits"`
	HitCount         int                     `json:"hitCount"`
	TopScore         float64                 `json:"topScore"`
	LatencyMs        int64                   `json:"latencyMs"`
	RetrieveMs       int64                   `json:"retrieveMs"`
	GenerateMs       int64                   `json:"generateMs"`
	PromptTokens     int                     `json:"promptTokens"`
	CompletionTokens int                     `json:"completionTokens"`
	ModelName        string                  `json:"modelName"`
	RetrieveLogID    int64                   `json:"retrieveLogId"`
}

type KnowledgeCitation struct {
	DocumentID    int64   `json:"documentId"`
	DocumentTitle string  `json:"documentTitle"`
	FaqID         int64   `json:"faqId"`
	FaqQuestion   string  `json:"faqQuestion"`
	ChunkNo       int     `json:"chunkNo"`
	Title         string  `json:"title"`
	SectionPath   string  `json:"sectionPath"`
	Snippet       string  `json:"snippet"`
	Score         float64 `json:"score"`
}

type KnowledgeRetrieveLogResponse struct {
	ID                     int64     `json:"id"`
	KnowledgeBaseID        int64     `json:"knowledgeBaseId"`
	KnowledgeBaseName      string    `json:"knowledgeBaseName,omitempty"`
	Channel                string    `json:"channel"`
	ChannelName            string    `json:"channelName"`
	Scene                  string    `json:"scene"`
	SceneName              string    `json:"sceneName"`
	SessionID              string    `json:"sessionId"`
	ConversationID         int64     `json:"conversationId"`
	RequestID              string    `json:"requestId"`
	Question               string    `json:"question"`
	RewriteQuestion        string    `json:"rewriteQuestion"`
	Answer                 string    `json:"answer"`
	AnswerStatus           int       `json:"answerStatus"`
	AnswerStatusName       string    `json:"answerStatusName"`
	HitCount               int       `json:"hitCount"`
	TopScore               float64   `json:"topScore"`
	ChunkProvider          string    `json:"chunkProvider"`
	ChunkTargetTokens      int       `json:"chunkTargetTokens"`
	ChunkMaxTokens         int       `json:"chunkMaxTokens"`
	ChunkOverlapTokens     int       `json:"chunkOverlapTokens"`
	RerankEnabled          bool      `json:"rerankEnabled"`
	RerankLimit            int       `json:"rerankLimit"`
	CitationCount          int       `json:"citationCount"`
	UsedChunkCount         int       `json:"usedChunkCount"`
	LatencyMs              int64     `json:"latencyMs"`
	RetrieveMs             int64     `json:"retrieveMs"`
	GenerateMs             int64     `json:"generateMs"`
	PromptTokens           int       `json:"promptTokens"`
	CompletionTokens       int       `json:"completionTokens"`
	ModelName              string    `json:"modelName"`
	TraceData              string    `json:"traceData"`
	FeedbackCount          int64     `json:"feedbackCount"`
	NegativeFeedbackCount  int64     `json:"negativeFeedbackCount"`
	LatestFeedbackType     int       `json:"latestFeedbackType"`
	LatestFeedbackTypeName string    `json:"latestFeedbackTypeName"`
	LatestFeedbackReason   string    `json:"latestFeedbackReason"`
	CreatedAt              time.Time `json:"createdAt"`
}

type KnowledgeRetrieveHitResponse struct {
	ID              int64     `json:"id"`
	RetrieveLogID   int64     `json:"retrieveLogId"`
	KnowledgeBaseID int64     `json:"knowledgeBaseId"`
	ChunkID         int64     `json:"chunkId"`
	DocumentID      int64     `json:"documentId"`
	DocumentTitle   string    `json:"documentTitle"`
	FaqID           int64     `json:"faqId"`
	FaqQuestion     string    `json:"faqQuestion"`
	ChunkNo         int       `json:"chunkNo"`
	Title           string    `json:"title"`
	SectionPath     string    `json:"sectionPath"`
	ChunkType       string    `json:"chunkType"`
	ChunkTypeName   string    `json:"chunkTypeName"`
	Provider        string    `json:"provider"`
	RankNo          int       `json:"rankNo"`
	Score           float64   `json:"score"`
	RerankScore     float64   `json:"rerankScore"`
	UsedInAnswer    bool      `json:"usedInAnswer"`
	IsCitation      bool      `json:"isCitation"`
	Snippet         string    `json:"snippet"`
	CreatedAt       time.Time `json:"createdAt"`
}

type KnowledgeRetrieveLogDetailResponse struct {
	Log       KnowledgeRetrieveLogResponse   `json:"log"`
	Hits      []KnowledgeRetrieveHitResponse `json:"hits"`
	Feedbacks []KnowledgeFeedbackResponse    `json:"feedbacks"`
}

type KnowledgeFAQDraftBatchCreateResponse struct {
	TotalCandidates int64                              `json:"totalCandidates"`
	CreatedCount    int64                              `json:"createdCount"`
	ReusedCount     int64                              `json:"reusedCount"`
	SkippedCount    int64                              `json:"skippedCount"`
	DraftIDs        []int64                            `json:"draftIds"`
	Skipped         []KnowledgeFAQDraftBatchSkipReason `json:"skipped"`
}

type KnowledgeFAQDraftBatchSkipReason struct {
	RetrieveLogID int64  `json:"retrieveLogId"`
	Question      string `json:"question"`
	Reason        string `json:"reason"`
}

type KnowledgeFeedbackResponse struct {
	ID               int64     `json:"id"`
	RetrieveLogID    int64     `json:"retrieveLogId"`
	FeedbackType     int       `json:"feedbackType"`
	FeedbackTypeName string    `json:"feedbackTypeName"`
	FeedbackReason   string    `json:"feedbackReason"`
	UserID           int64     `json:"userId"`
	AgentID          int64     `json:"agentId"`
	Remark           string    `json:"remark"`
	CreatedAt        time.Time `json:"createdAt"`
}
