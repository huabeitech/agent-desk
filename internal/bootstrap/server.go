package bootstrap

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"agent-desk/internal/ai/mcps"
	_ "agent-desk/internal/ai/runtime"
	"agent-desk/internal/handlers/api"
	"agent-desk/internal/middleware"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/ginx"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/pkg/tracex"
	"agent-desk/internal/services"
	webspa "agent-desk/web"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"

	_ "agent-desk/internal/services/wx_callback_handlers"
)

func NewServer() (*gin.Engine, error) {
	cfg := config.Current()
	i18nx.SetDefaultLocale(cfg.LanguageOrDefault())

	gin.SetMode(gin.ReleaseMode)
	printBanner()

	app := gin.New()
	app.Use(requestIDMiddleware())
	app.Use(corsMiddleware())
	app.Use(gin.Recovery())
	app.Use(requestLogMiddleware())
	app.Use(maxBodySizeMiddleware())
	app.Use(i18nx.Middleware())

	addRouter(app)

	notFoundPrefixes := []string{"/api/"}
	if baseURL := strings.TrimRight(cfg.Storage.Local.BaseURL, "/"); baseURL != "" {
		notFoundPrefixes = append(notFoundPrefixes, baseURL+"/")
	}
	app.StaticFS(cfg.Storage.Local.BaseURL, ginx.StaticFiles(cfg.Storage.Local.Root))
	ginx.HandleSPA(app, ginx.SPAOptions{
		Root:         "./web/out",
		EmbeddedFS:   webspa.SPA,
		EmbeddedRoot: "out",
		DirOptions: ginx.DirOptions{
			ShowList:  false,
			SPA:       true,
			IndexName: "index.html",
		},
		NotFoundPrefixes: notFoundPrefixes,
		NotFoundHandler: func(ctx *gin.Context) {
			httpx.WriteHttpStatusJSON(ctx, http.StatusNotFound, web.JsonErrorCode(http.StatusNotFound, i18nx.T(ctx, "error.notFound")))
		},
	})

	return app, nil
}

func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := config.Current().Server.CORS.AllowedOrigins
	allowHeaders := "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Guest-Id, X-Channel-Id, X-External-Id, X-External-Name, X-Customer-Session-Token, X-Customer-Session-Expires-At"
	exposeHeaders := "Content-Length, Content-Type, Authorization, X-Guest-Id, X-Channel-Id, X-External-Id, X-External-Name, X-Customer-Session-Token, X-Customer-Session-Expires-At"
	allowMethods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedOriginSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" {
			continue
		}
		allowedOriginSet[origin] = struct{}{}
	}
	return func(ctx *gin.Context) {
		if isWebsocketUpgrade(ctx) {
			ctx.Next()
			return
		}
		origin := strings.TrimRight(strings.TrimSpace(ctx.GetHeader("Origin")), "/")
		if origin != "" {
			ctx.Header("Vary", "Origin")
			if _, ok := allowedOriginSet[origin]; !ok {
				if ctx.Request.Method == http.MethodOptions {
					ctx.AbortWithStatus(http.StatusForbidden)
					return
				}
				ctx.Next()
				return
			}
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Methods", allowMethods)
			ctx.Header("Access-Control-Allow-Headers", allowHeaders)
			ctx.Header("Access-Control-Expose-Headers", exposeHeaders)
			ctx.Header("Access-Control-Max-Age", "600")
		}
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := tracex.EnsureRequestID(ctx.GetHeader(tracex.RequestIDHeader))
		ctx.Set(tracex.GinRequestIDKey, requestID)
		if requestID != "" {
			ctx.Header(tracex.RequestIDHeader, requestID)
		}
		ctx.Next()
	}
}

func requestLogMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		method := ctx.Request.Method
		requestID, _ := ctx.Get(tracex.GinRequestIDKey)
		ctx.Next()

		slog.Info("http request",
			"requestId", requestID,
			"method", method,
			"path", path,
			"status", ctx.Writer.Status(),
			"elapsed", time.Since(start).Milliseconds(),
			"clientIp", ctx.ClientIP(),
		)
	}
}

func maxBodySizeMiddleware() gin.HandlerFunc {
	limit := config.Current().Storage.MaxRequestBodySizeBytes()
	return func(ctx *gin.Context) {
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, limit)
		ctx.Next()
	}
}

func isWebsocketUpgrade(ctx *gin.Context) bool {
	if !strings.EqualFold(ctx.GetHeader("Upgrade"), "websocket") {
		return false
	}
	return strings.Contains(strings.ToLower(ctx.GetHeader("Connection")), "upgrade")
}

func addRouter(app *gin.Engine) {
	app.Any("/api/mcp", gin.WrapH(mcps.NewHTTPHandler()))

	apiGroup := app.Group("/api")
	apiGroup.GET("/health", api.Health)
	apiGroup.GET("/config", api.PublicConfig)
	registerApiAuthRoutes(apiGroup.Group("/auth"))
	registerApiChannelRoutes(apiGroup.Group("/channel"))
	registerApiCustomerRoutes(apiGroup.Group("/customer"))
	registerApiConversationRoutes(apiGroup.Group("/conversation", middleware.ExternalUserMiddleware))
	registerApiMessageRoutes(apiGroup.Group("/message", middleware.ExternalUserMiddleware))

	wsGroup := app.Group("/api/ws")
	wsGroup.GET("/dashboard", middleware.AuthMiddleware, services.WsService.HandleDashboardWS)
	wsGroup.GET("/dashboard/notification", middleware.AuthMiddleware, services.WsService.HandleDashboardNotificationWS)
	wsGroup.GET("/open", services.WsService.HandleOpenWS)

	dashboardGroup := app.Group("/api/dashboard", middleware.AuthMiddleware)
	registerDashboardDashboardRoutes(dashboardGroup.Group("/dashboard"))
	registerDashboardBusinessReportRoutes(dashboardGroup.Group("/business-report"))
	registerDashboardUserRoutes(dashboardGroup.Group("/user"))
	registerDashboardCompanyRoutes(dashboardGroup.Group("/company"))
	registerDashboardCustomerRoutes(dashboardGroup.Group("/customer"))
	registerDashboardSalesLeadRoutes(dashboardGroup.Group("/sales-lead"))
	registerDashboardProductRoutes(dashboardGroup.Group("/product"))
	registerDashboardPromotionRoutes(dashboardGroup.Group("/promotion"))
	registerDashboardDigitalStoreRoutes(dashboardGroup.Group("/digital-store"))
	registerDashboardCustomerContactRoutes(dashboardGroup.Group("/customer-contact"))
	registerDashboardRoleRoutes(dashboardGroup.Group("/role"))
	registerDashboardPermissionRoutes(dashboardGroup.Group("/permission"))
	registerDashboardSessionRoutes(dashboardGroup.Group("/session"))
	registerDashboardTagRoutes(dashboardGroup.Group("/tag"))
	registerDashboardConversationRoutes(dashboardGroup.Group("/conversation"))
	registerDashboardTicketRoutes(dashboardGroup.Group("/ticket"))
	registerDashboardNotificationRoutes(dashboardGroup.Group("/notification"))
	registerDashboardQuickReplyRoutes(dashboardGroup.Group("/quick-reply"))
	registerDashboardChannelRoutes(dashboardGroup.Group("/channel"))
	registerDashboardAgentRoutes(dashboardGroup.Group("/agent"))
	registerDashboardAgentTeamRoutes(dashboardGroup.Group("/agent-team"))
	registerDashboardAgentTeamScheduleRoutes(dashboardGroup.Group("/agent-team-schedule"))
	registerDashboardAIAgentRoutes(dashboardGroup.Group("/ai-agent"))
	registerDashboardAIWorkflowRoutes(dashboardGroup.Group("/ai-workflow"))
	registerDashboardAIConfigRoutes(dashboardGroup.Group("/ai-config"))
	registerDashboardAssetRoutes(dashboardGroup.Group("/asset"))
	registerDashboardKnowledgeBaseRoutes(dashboardGroup.Group("/knowledge-base"))
	registerDashboardKnowledgeDirectoryRoutes(dashboardGroup.Group("/knowledge-directory"))
	registerDashboardKnowledgeDocumentRoutes(dashboardGroup.Group("/knowledge-document"))
	registerDashboardKnowledgeFAQRoutes(dashboardGroup.Group("/knowledge-faq"))
	registerDashboardKnowledgeRetrieveRoutes(dashboardGroup.Group("/knowledge-retrieve"))
	registerDashboardKnowledgeRetrieveLogRoutes(dashboardGroup.Group("/knowledge-retrieve-log"))
	registerDashboardSkillDefinitionRoutes(dashboardGroup.Group("/skill-definition"))
	registerDashboardMCPRoutes(dashboardGroup.Group("/mcp"))

	thirdGroup := app.Group("/api/third")
	registerThirdWechatRoutes(thirdGroup.Group("/wechat"))
}
