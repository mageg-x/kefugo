package controllers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// AdminPanelController 提供管理后台看板、访客统计、导出与系统设置接口。
type AdminPanelController struct{}

func applySystemSettingsRuntimeHooks(cfg systemSettings) {
	// 预留：邮件通知能力按配置驱动；当前版本未接入 SMTP，仅在日志中输出状态。
	if cfg.EmailNotify {
		logger.Infof("system settings runtime email notify enabled email=%s", strings.TrimSpace(cfg.NotifyEmail))
	} else {
		logger.Infof("system settings runtime email notify disabled")
	}

	// 预留：会话加密开关。
	if cfg.SessionEncrypt {
		logger.Infof("system settings runtime session encryption enabled")
	} else {
		logger.Infof("system settings runtime session encryption disabled")
	}
}

type dashboardRecentSession struct {
	SID       string `json:"sid"`
	VisitorID string `json:"visitor_id"`
	AppID     string `json:"app_id"`
	AgentID   string `json:"agent_id"`
	Status    string `json:"status"`
	Time      int64  `json:"time"`
}

type visitorListItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	Sessions  int    `json:"sessions"`
	LastVisit int64  `json:"last_visit"`
	Avatar    string `json:"avatar"`
	IP        string `json:"ip,omitempty"`
	Device    string `json:"device,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Geo       string `json:"geo,omitempty"`
}

func isAdminRole(role string) bool {
	return role == "admin"
}

// Dashboard 返回管理后台首页统计数据。
func (ac *AdminPanelController) Dashboard(c *gin.Context) {
	_, role := getAuthUser(c)
	if !isAdminRole(role) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAdminForbidden)
		return
	}

	appIDFilter := strings.TrimSpace(c.Query("app_id"))
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()

	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("admin dashboard session service unavailable")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	totalSessions := 0
	todaySessions := 0
	resolvedSessions := 0
	pendingSessions := 0
	activeSessions := 0
	recent := make([]dashboardRecentSession, 0, 64)

	err := ss.IterateSessions(func(session *models.Session) bool {
		if session == nil {
			return true
		}
		visitorID, appID, _ := session.ParseSid()
		if visitorID == "" || appID == "" {
			return true
		}
		if appIDFilter != "" && appID != appIDFilter {
			return true
		}

		totalSessions++
		if session.CreatedAt >= dayStart {
			todaySessions++
		}
		status := session.Status()
		switch status {
		case models.SessionStatusClosed:
			resolvedSessions++
		case models.SessionStatusAssigned, models.SessionStatusFollowUP:
			activeSessions++
		default:
			pendingSessions++
		}

		lastActive := session.LastVisitorMsgTime
		if session.LastAgentReplyTime > lastActive {
			lastActive = session.LastAgentReplyTime
		}
		if lastActive == 0 {
			lastActive = session.CreatedAt
		}

		recent = append(recent, dashboardRecentSession{
			SID:       session.SID,
			VisitorID: visitorID,
			AppID:     appID,
			AgentID:   session.CurAgentID,
			Status:    status,
			Time:      lastActive,
		})
		return true
	})
	if err != nil {
		logger.Errorf("admin dashboard iterate sessions failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminDashboardFailed)
		return
	}

	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Time > recent[j].Time
	})
	if len(recent) > 10 {
		recent = recent[:10]
	}

	var onlineAgents int64
	_ = store.DB.Model(&models.User{}).
		Where("role = ? AND active = ? AND status = ?", "agent", true, 1).
		Count(&onlineAgents).Error

	response.ResponseSuccess(c, gin.H{
		"today_sessions":    todaySessions,
		"resolved_sessions": resolvedSessions,
		"pending_sessions":  pendingSessions,
		"active_sessions":   activeSessions,
		"total_sessions":    totalSessions,
		"online_agents":     onlineAgents,
		"recent_sessions":   recent,
	})
}

// Visitors 返回访客聚合列表（支持关键字、状态与分页筛选）。
func (ac *AdminPanelController) Visitors(c *gin.Context) {
	_, role := getAuthUser(c)
	if !isAdminRole(role) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAdminForbidden)
		return
	}

	appIDFilter := strings.TrimSpace(c.Query("app_id"))
	keyword := strings.TrimSpace(strings.ToLower(c.Query("keyword")))
	statusFilter := strings.TrimSpace(strings.ToLower(c.Query("status")))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 10
	}

	type visitorAgg struct {
		ID        string
		Sessions  int
		LastVisit int64
		IP        string
		Device    string
		UserAgent string
		Geo       string
	}
	aggs := map[string]*visitorAgg{}
	nowTS := time.Now().Unix()

	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("admin visitors session service unavailable")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}
	err := ss.IterateSessions(func(session *models.Session) bool {
		if session == nil {
			return true
		}
		visitorID, appID, _ := session.ParseSid()
		if visitorID == "" || appID == "" {
			return true
		}
		if appIDFilter != "" && appID != appIDFilter {
			return true
		}

		lastActive := session.LastVisitorMsgTime
		if session.LastAgentReplyTime > lastActive {
			lastActive = session.LastAgentReplyTime
		}
		if lastActive == 0 {
			lastActive = session.CreatedAt
		}

		item, ok := aggs[visitorID]
		if !ok {
			item = &visitorAgg{ID: visitorID}
			aggs[visitorID] = item
		}
		item.Sessions++
		if lastActive > item.LastVisit {
			item.LastVisit = lastActive
			item.IP = session.LastClientIP
			item.Device = session.LastDevice
			item.UserAgent = session.LastUserAgent
			item.Geo = session.LastGeo
		}
		return true
	})
	if err != nil {
		logger.Errorf("admin visitors iterate sessions failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminVisitorsFailed)
		return
	}

	rows := make([]visitorListItem, 0, len(aggs))
	for _, agg := range aggs {
		if keyword != "" && !strings.Contains(strings.ToLower(agg.ID), keyword) {
			continue
		}
		status := "offline"
		if nowTS-agg.LastVisit <= 5*60 {
			status = "online"
		}
		if statusFilter != "" && statusFilter != status {
			continue
		}
		rows = append(rows, visitorListItem{
			ID:        agg.ID,
			Name:      agg.ID,
			Email:     "",
			Status:    status,
			Sessions:  agg.Sessions,
			LastVisit: agg.LastVisit,
			Avatar:    "",
			IP:        agg.IP,
			Device:    agg.Device,
			UserAgent: agg.UserAgent,
			Geo:       agg.Geo,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].LastVisit > rows[j].LastVisit
	})

	total := len(rows)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	response.ResponseSuccess(c, gin.H{
		"data":      rows[start:end],
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UserStats 返回客服会话统计数据。
func (ac *AdminPanelController) UserStats(c *gin.Context) {
	_, role := getAuthUser(c)
	if !isAdminRole(role) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAdminForbidden)
		return
	}

	appID := strings.TrimSpace(c.Query("app_id"))
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("admin user stats session service unavailable")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	sessionsByAgent := map[string]int{}
	err := ss.IterateSessions(func(session *models.Session) bool {
		if session == nil || session.CurAgentID == "" {
			return true
		}
		_, sidAppID, _ := session.ParseSid()
		if appID != "" && sidAppID != appID {
			return true
		}
		sessionsByAgent[session.CurAgentID]++
		return true
	})
	if err != nil {
		logger.Errorf("admin user stats iterate sessions failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminUserStatsFailed)
		return
	}

	var users []models.User
	query := store.DB.Model(&models.User{}).Where("role IN ?", []string{"agent", "admin"})
	if err := query.Order("id DESC").Find(&users).Error; err != nil {
		logger.Errorf("admin user stats list users failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserListFailed)
		return
	}

	type statItem struct {
		Username string `json:"username"`
		Sessions int    `json:"sessions"`
	}
	result := make([]statItem, 0, len(users))
	for _, u := range users {
		result = append(result, statItem{
			Username: u.Username,
			Sessions: sessionsByAgent[u.Username],
		})
	}

	response.ResponseSuccess(c, gin.H{"data": result})
}

// GetSystemSettings 获取系统配置。
func (ac *AdminPanelController) GetSystemSettings(c *gin.Context) {
	userName, role := getAuthUser(c)
	if role != "admin" {
		logger.Errorf("system settings get forbidden user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAdminForbidden)
		return
	}
	response.ResponseSuccess(c, getSystemSettingsCached())
}

// UpdateSystemSettings 更新系统配置并刷新缓存。
func (ac *AdminPanelController) UpdateSystemSettings(c *gin.Context) {
	userName, role := getAuthUser(c)
	if role != "admin" {
		logger.Errorf("system settings update forbidden user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAdminForbidden)
		return
	}

	req := defaultSystemSettings()
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("admin settings params invalid: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAdminSettingsInvalidParams)
		return
	}
	req = normalizeSystemSettings(req)

	valueBytes, _ := json.Marshal(req)
	var row models.SystemSetting
	err := store.DB.Where("key = ?", "system_settings").First(&row).Error
	if err != nil {
		row = models.SystemSetting{
			Key:   "system_settings",
			Value: string(valueBytes),
		}
		if createErr := store.DB.Create(&row).Error; createErr != nil {
			logger.Errorf("admin settings create failed: %v", createErr)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminSettingsSaveFailed)
			return
		}
	} else {
		row.Value = string(valueBytes)
		if saveErr := store.DB.Save(&row).Error; saveErr != nil {
			logger.Errorf("admin settings save failed: %v", saveErr)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminSettingsSaveFailed)
			return
		}
	}

	RecordAudit(c, "settings.system.update", "system_setting", "system_settings", "success", userName)
	setSystemSettingsCache(req)
	applySystemSettingsRuntimeHooks(req)
	response.ResponseSuccess(c, req)
}

// GetAgentAIBotSettings 返回客服可管理的 AI 配置子集。
func (ac *AdminPanelController) GetAgentAIBotSettings(c *gin.Context) {
	cfg := getSystemSettingsCached()
	response.ResponseSuccess(c, gin.H{
		"aiBotEnabled":      cfg.AIBotEnabled,
		"aiBotName":         cfg.AIBotName,
		"aiBotModel":        cfg.AIBotModel,
		"aiBotStyle":        cfg.AIBotStyle,
		"aiBotPrompt":       cfg.AIBotPrompt,
		"aiBotTopK":         cfg.AIBotTopK,
		"aiBotWhenAssigned": cfg.AIBotWhenAssigned,
	})
}

// UpdateAgentAIBotSettings 更新客服可管理的 AI 配置子集。
func (ac *AdminPanelController) UpdateAgentAIBotSettings(c *gin.Context) {
	var req struct {
		AIBotEnabled      bool   `json:"aiBotEnabled"`
		AIBotName         string `json:"aiBotName"`
		AIBotModel        string `json:"aiBotModel"`
		AIBotStyle        string `json:"aiBotStyle"`
		AIBotPrompt       string `json:"aiBotPrompt"`
		AIBotTopK         int    `json:"aiBotTopK"`
		AIBotWhenAssigned bool   `json:"aiBotWhenAssigned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("agent ai settings params invalid: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAdminSettingsInvalidParams)
		return
	}

	cfg := loadSystemSettingsFromDB()
	cfg.AIBotEnabled = req.AIBotEnabled
	cfg.AIBotName = strings.TrimSpace(req.AIBotName)
	cfg.AIBotModel = strings.TrimSpace(req.AIBotModel)
	cfg.AIBotStyle = strings.TrimSpace(req.AIBotStyle)
	cfg.AIBotPrompt = strings.TrimSpace(req.AIBotPrompt)
	cfg.AIBotTopK = req.AIBotTopK
	cfg.AIBotWhenAssigned = req.AIBotWhenAssigned
	cfg = normalizeSystemSettings(cfg)

	if err := saveSystemSettingsRecord(cfg); err != nil {
		logger.Errorf("agent ai settings save failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminSettingsSaveFailed)
		return
	}
	setSystemSettingsCache(cfg)
	applySystemSettingsRuntimeHooks(cfg)
	response.ResponseSuccess(c, gin.H{
		"aiBotEnabled":      cfg.AIBotEnabled,
		"aiBotName":         cfg.AIBotName,
		"aiBotModel":        cfg.AIBotModel,
		"aiBotStyle":        cfg.AIBotStyle,
		"aiBotPrompt":       cfg.AIBotPrompt,
		"aiBotTopK":         cfg.AIBotTopK,
		"aiBotWhenAssigned": cfg.AIBotWhenAssigned,
	})
}

// GetAgentSensitiveWords 返回客服可管理的敏感词配置。
func (ac *AdminPanelController) GetAgentSensitiveWords(c *gin.Context) {
	cfg := getSystemSettingsCached()
	response.ResponseSuccess(c, gin.H{
		"sensitiveWords": cfg.SensitiveWords,
	})
}

// UpdateAgentSensitiveWords 更新客服可管理的敏感词配置。
func (ac *AdminPanelController) UpdateAgentSensitiveWords(c *gin.Context) {
	var req struct {
		SensitiveWords string `json:"sensitiveWords"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("agent sensitive words params invalid: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAdminSettingsInvalidParams)
		return
	}

	cfg := loadSystemSettingsFromDB()
	cfg.SensitiveWords = strings.TrimSpace(req.SensitiveWords)
	cfg = normalizeSystemSettings(cfg)
	if err := saveSystemSettingsRecord(cfg); err != nil {
		logger.Errorf("agent sensitive words save failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminSettingsSaveFailed)
		return
	}
	setSystemSettingsCache(cfg)
	applySystemSettingsRuntimeHooks(cfg)
	response.ResponseSuccess(c, gin.H{
		"sensitiveWords": cfg.SensitiveWords,
	})
}

// ProfileSummary 返回当前登录用户在个人中心需要的摘要统计。
func (ac *AdminPanelController) ProfileSummary(c *gin.Context) {
	userName, _ := getAuthUser(c)
	if userName == "" {
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAdminProfileSummaryUnauthorized)
		return
	}

	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("admin profile summary session service unavailable user=%s", userName)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	sessionsToday := 0
	totalAssigned := 0

	err := ss.IterateSessions(func(session *models.Session) bool {
		if session == nil {
			return true
		}
		if session.CurAgentID != userName {
			return true
		}
		totalAssigned++
		if session.CreatedAt >= dayStart {
			sessionsToday++
		}
		return true
	})
	if err != nil {
		logger.Errorf("admin profile summary iterate sessions failed user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAdminProfileSummaryFailed)
		return
	}

	totalScore := 0
	ratedCount := 0
	_ = ss.IterateSessions(func(session *models.Session) bool {
		if session == nil || session.CurAgentID != userName {
			return true
		}
		if session.RatingScore >= 1 && session.RatingScore <= 5 {
			totalScore += session.RatingScore
			ratedCount++
		}
		return true
	})
	rating := 0.0
	if ratedCount > 0 {
		rating = float64(totalScore) / float64(ratedCount)
	}

	response.ResponseSuccess(c, gin.H{
		"sessions_today": sessionsToday,
		"total_assigned": totalAssigned,
		"rating":         rating,
		"rated_count":    ratedCount,
	})
}

// ExportSessions 导出会话数据为 CSV。
func (ac *AdminPanelController) ExportSessions(c *gin.Context) {
	_, role := getAuthUser(c)
	if !isAdminRole(role) {
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeAdminForbidden)
		return
	}

	appIDFilter := strings.TrimSpace(c.Query("app_id"))
	statusFilter := strings.TrimSpace(c.Query("status"))
	startTime, _ := strconv.ParseInt(strings.TrimSpace(c.Query("start_time")), 10, 64)
	endTime, _ := strconv.ParseInt(strings.TrimSpace(c.Query("end_time")), 10, 64)

	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("admin export sessions service unavailable")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	var rows [][]string
	rows = append(rows, []string{
		"sid", "visitor_id", "app_id", "status", "cur_agent_id", "created_at",
		"last_active_at", "unread_count", "last_message_type", "last_message",
		"last_client_ip", "last_device", "last_geo", "rating_score", "rating_comment",
	})

	_ = ss.IterateSessions(func(session *models.Session) bool {
		if session == nil {
			return true
		}
		visitorID, appID, _ := session.ParseSid()
		if visitorID == "" || appID == "" {
			return true
		}
		if appIDFilter != "" && appID != appIDFilter {
			return true
		}
		if statusFilter != "" && session.Status() != statusFilter {
			return true
		}
		activity := session.LastActiveAt
		if activity == 0 {
			activity = session.CreatedAt
		}
		if startTime > 0 && activity < startTime {
			return true
		}
		if endTime > 0 && activity > endTime {
			return true
		}
		rows = append(rows, []string{
			session.SID,
			visitorID,
			appID,
			session.Status(),
			session.CurAgentID,
			strconv.FormatInt(session.CreatedAt, 10),
			strconv.FormatInt(activity, 10),
			strconv.Itoa(session.UnreadCount),
			session.LastMessageType,
			session.LastMessage,
			session.LastClientIP,
			session.LastDevice,
			session.LastGeo,
			strconv.Itoa(session.RatingScore),
			session.RatingComment,
		})
		return true
	})

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.WriteAll(rows)
	w.Flush()

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="sessions_export.csv"`)
	c.String(http.StatusOK, buf.String())
}
