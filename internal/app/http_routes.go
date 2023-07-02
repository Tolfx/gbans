package app

import (
	"context"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()

	return func(ctx *gin.Context) {
		h.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	log := logger.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			log.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}

// jsConfig contains all the variables that we inject into the frontend at runtime.
type jsConfig struct {
	SiteName        string `json:"site_name"`
	DiscordClientID string `json:"discord_client_id"`
	DiscordLinkID   string `json:"discord_link_id"`
}

//nolint:contextcheck
func createRouter(ctx context.Context, app *App) *gin.Engine {
	engine := gin.New()
	engine.Use(ErrorHandler(app.log), gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = app.conf.HTTP.CorsOrigins
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowWildcard = true
	corsConfig.AllowCredentials = false
	if app.conf.General.Mode != config.TestMode {
		engine.Use(cors.New(corsConfig))
	}

	prom := ginprom.New(func(prom *ginprom.Prometheus) {
		prom.Namespace = "gbans"
		prom.Subsystem = "http"
	})
	engine.Use(prom.Instrument())

	staticPath := app.conf.HTTP.StaticPath
	if staticPath == "" {
		staticPath = "./dist"
	}
	absStaticPath, errStaticPath := filepath.Abs(staticPath)
	if errStaticPath != nil {
		app.log.Fatal("Invalid static path", zap.Error(errStaticPath))
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))
	engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))

	// These should match routes defined in the frontend. This allows us to use the browser
	// based routing when serving the SPA.
	jsRoutes := []string{
		"/", "/servers", "/profile/:steam_id", "/bans", "/appeal", "/settings", "/report",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban", "/admin/reports", "/admin/news",
		"/admin/import", "/admin/filters", "/404", "/logout", "/login/success", "/report/:report_id", "/wiki",
		"/wiki/*slug", "/log/:match_id", "/logs", "/ban/:ban_id", "/admin/chat", "/admin/appeals", "/login",
		"/pug", "/quickplay", "/global_stats", "/stv", "/login/discord", "/notifications",
	}
	for _, rt := range jsRoutes {
		engine.GET(rt, func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", jsConfig{
				SiteName:        app.conf.General.SiteName,
				DiscordClientID: app.conf.Discord.AppID,
				DiscordLinkID:   app.conf.Discord.LinkID,
			})
		})
	}
	engine.GET("/auth/callback", onOpenIDCallback(app))
	engine.GET("/api/auth/logout", onGetLogout(app))
	engine.POST("/api/auth/refresh", onTokenRefresh(app))

	engine.GET("/export/bans/tf2bd", onAPIExportBansTF2BD(app))
	engine.GET("/export/sourcemod/admins_simple.ini", onAPIExportSourcemodSimpleAdmins(app))
	engine.GET("/export/bans/valve/steamid", onAPIExportBansValveSteamID(app))
	engine.GET("/export/bans/valve/network", onAPIExportBansValveIP(app))
	engine.GET("/metrics", prometheusHandler())

	engine.GET("/api/profile", onAPIProfile(app))
	engine.GET("/api/servers/state", onAPIGetServerStates(app))
	engine.GET("/api/stats", onAPIGetStats(app))

	engine.GET("/api/players", onAPIGetPlayers(app))
	engine.GET("/api/wiki/slug/*slug", onAPIGetWikiSlug(app))
	engine.GET("/api/log/:match_id", onAPIGetMatch(app))
	engine.POST("/api/logs", onAPIGetMatches(app))
	engine.GET("/media/:media_id", onGetMediaByID(app))
	engine.POST("/api/news_latest", onAPIGetNewsLatest(app))
	engine.POST("/api/server_query", onAPIPostServerQuery(app))
	engine.GET("/api/server_stats", onAPIGetTF2Stats(app))

	engine.POST("/api/demos", onAPIPostDemosQuery(app))
	engine.GET("/demos/name/:demo_name", onAPIGetDemoDownloadByName(app))
	engine.GET("/demos/:demo_id", onAPIGetDemoDownload(app))

	// Service discovery endpoints
	engine.GET("/api/sd/prometheus/hosts", onAPIGetPrometheusHosts(app))
	engine.GET("/api/sd/ansible/hosts", onAPIGetPrometheusHosts(app))

	// Game server plugin routes
	engine.POST("/api/server/auth", onSAPIPostServerAuth(app))
	engine.POST("/api/resolve_profile", onAPIGetResolveProfile(app))

	engine.GET("/api/patreon/campaigns", onAPIGetPatreonCampaigns(app))
	engine.GET("/api/patreon/pledges", onAPIGetPatreonPledges(app))

	srvGrp := engine.Group("/")
	{
		// Server Auth Request
		serverAuth := srvGrp.Use(authServerMiddleWare(app))
		serverAuth.GET("/api/server/admins", onAPIGetServerAdmins(app))
		serverAuth.POST("/api/ping_mod", onAPIPostPingMod(app))
		serverAuth.POST("/api/check", onAPIPostServerCheck(app))
		serverAuth.POST("/api/demo", onAPIPostDemo(app))
		serverAuth.POST("/api/log", onAPIPostLog(app))
		// Duplicated since we need to authenticate via server middleware
		serverAuth.POST("/api/sm/bans/steam/create", onAPIPostBanSteamCreate(app))
		serverAuth.POST("/api/sm/report/create", onAPIPostReportCreate(app))
	}

	cm := newWSConnectionManager(ctx, app.log)

	authedGrp := engine.Group("/")
	{
		// Basic logged-in user
		authed := authedGrp.Use(authMiddleware(app, consts.PUser))
		authed.GET("/ws", func(c *gin.Context) {
			wsConnHandler(c.Writer, c.Request, cm, currentUserProfile(c), app.log)
		})

		authed.GET("/api/auth/discord", onOAuthDiscordCallback(app))
		authed.GET("/api/current_profile", onAPICurrentProfile(app))
		authed.POST("/api/current_profile/notifications", onAPICurrentProfileNotifications(app))
		authed.POST("/api/report", onAPIPostReportCreate(app))
		authed.GET("/api/report/:report_id", onAPIGetReport(app))
		authed.POST("/api/reports", onAPIGetReports(app))
		authed.POST("/api/report_status/:report_id", onAPISetReportStatus(app))
		authed.POST("/api/media", onAPISaveMedia(app))

		authed.GET("/api/report/:report_id/messages", onAPIGetReportMessages(app))
		authed.POST("/api/report/:report_id/messages", onAPIPostReportMessage(app))
		authed.POST("/api/report/message/:report_message_id", onAPIEditReportMessage(app))
		authed.DELETE("/api/report/message/:report_message_id", onAPIDeleteReportMessage(app))

		authed.GET("/api/bans/steam/:ban_id", onAPIGetBanByID(app))
		authed.GET("/api/bans/:ban_id/messages", onAPIGetBanMessages(app))
		authed.POST("/api/bans/:ban_id/messages", onAPIPostBanMessage(app))
		authed.POST("/api/bans/message/:ban_message_id", onAPIEditBanMessage(app))
		authed.DELETE("/api/bans/message/:ban_message_id", onAPIDeleteBanMessage(app))
	}

	editorGrp := engine.Group("/")
	{
		// Editor access
		editorRoute := editorGrp.Use(authMiddleware(app, consts.PEditor))
		editorRoute.POST("/api/wiki/slug", onAPISaveWikiSlug(app))
		editorRoute.POST("/api/news", onAPIPostNewsCreate(app))
		editorRoute.POST("/api/news/:news_id", onAPIPostNewsUpdate(app))
		editorRoute.POST("/api/news_all", onAPIGetNewsAll(app))
		editorRoute.GET("/api/filters", onAPIGetWordFilters(app))
		editorRoute.POST("/api/filters", onAPIPostWordFilter(app))
		editorRoute.DELETE("/api/filters/:word_id", onAPIDeleteWordFilter(app))
		editorRoute.POST("/api/filter_match", onAPIPostWordMatch(app))
	}

	modGrp := engine.Group("/")
	{
		// Moderator access
		modRoute := modGrp.Use(authMiddleware(app, consts.PModerator))
		modRoute.POST("/api/report/:report_id/state", onAPIPostBanState(app))
		modRoute.GET("/api/connections/:steam_id", onAPIGetPersonConnections(app))
		modRoute.GET("/api/messages/:steam_id", onAPIGetPersonMessages(app))
		modRoute.GET("/api/message/:person_message_id/context", onAPIGetMessageContext(app))
		modRoute.POST("/api/messages", onAPIQueryMessages(app))
		modRoute.POST("/api/appeals", onAPIGetAppeals(app))
		modRoute.POST("/api/bans/steam", onAPIGetBansSteam(app))
		modRoute.POST("/api/bans/steam/create", onAPIPostBanSteamCreate(app))
		modRoute.DELETE("/api/bans/steam/:ban_id", onAPIPostBanDelete(app))
		modRoute.POST("/api/bans/steam/:ban_id/status", onAPIPostSetBanAppealStatus(app))
		modRoute.POST("/api/bans/cidr/create", onAPIPostBansCIDRCreate(app))
		modRoute.POST("/api/bans/cidr", onAPIGetBansCIDR(app))
		modRoute.DELETE("/api/bans/cidr/:net_id", onAPIDeleteBansCIDR(app))
		modRoute.POST("/api/bans/asn/create", onAPIPostBansASNCreate(app))
		modRoute.POST("/api/bans/asn", onAPIGetBansASN(app))
		modRoute.DELETE("/api/bans/asn/:asn_id", onAPIDeleteBansASN(app))
		modRoute.POST("/api/bans/group/create", onAPIPostBansGroupCreate(app))
		modRoute.POST("/api/bans/group", onAPIGetBansGroup(app))
		modRoute.DELETE("/api/bans/group/:ban_group_id", onAPIDeleteBansGroup(app))
	}

	adminGrp := engine.Group("/")
	{
		// Admin access
		adminRoute := adminGrp.Use(authMiddleware(app, consts.PAdmin))
		adminRoute.POST("/api/servers", onAPIPostServer(app))
		adminRoute.POST("/api/servers/:server_id", onAPIPostServerUpdate(app))
		adminRoute.DELETE("/api/servers/:server_id", onAPIPostServerDelete(app))
		adminRoute.GET("/api/servers", onAPIGetServers(app))
	}

	return engine
}