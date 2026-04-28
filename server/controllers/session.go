package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"

	"github.com/gin-gonic/gin"
)

// SessionController 提供会话列表、消息查询、接待、转接、关闭、已读、评分等接口
// 这些接口主要供客服工作台和管理后台使用
type SessionController struct{}

var (
	errSessionAccessDenied   = errors.New("session access denied")
	errSessionClosedState    = errors.New("session closed")
	errSessionAlreadyTaken   = errors.New("session already assigned")
	errSessionOwnerMismatch  = errors.New("session owner mismatch")
	errSessionAlreadyRated   = errors.New("session already rated")
	errSessionInvalidRating  = errors.New("session rating invalid")
)

// sessionListItem 会话列表项数据结构
// 用于API响应返回，包含会话的核心信息和状态
type sessionListItem struct {
	SID                string `json:"sid"`                         // 会话唯一标识
	VisitorID          string `json:"visitor_id"`                  // 访客唯一标识
	AppID              string `json:"app_id"`                      // 应用唯一标识
	Status             string `json:"status"`                      // 会话状态
	CurAgentID         string `json:"cur_agent_id"`                // 当前负责客服ID
	LastClientIP       string `json:"last_client_ip,omitempty"`    // 访客最后使用的IP
	LastUserAgent      string `json:"last_user_agent,omitempty"`   // 访客最后使用的User-Agent
	LastDevice         string `json:"last_device,omitempty"`       // 访客最后使用的设备类型
	LastGeo            string `json:"last_geo,omitempty"`          // 访客地理位置
	RatingScore        int    `json:"rating_score,omitempty"`      // 访客评分（1-5分）
	RatingComment      string `json:"rating_comment,omitempty"`    // 访客评价内容
	RatedAt            int64  `json:"rated_at,omitempty"`          // 评分时间
	LastVisitorMsgTime int64  `json:"last_visitor_msg_time"`       // 访客最后发送消息时间
	LastAgentReplyTime int64  `json:"last_agent_reply_time"`       // 客服最后回复时间
	LastAgentReadTime  int64  `json:"last_agent_read_time"`        // 客服最后已读时间
	CreatedAt          int64  `json:"created_at"`                  // 会话创建时间
	UnreadCount        int    `json:"unread_count"`                // 未读消息数量
	LastMessage        string `json:"last_message,omitempty"`      // 最后一条消息内容摘要
	LastMessageType    string `json:"last_message_type,omitempty"` // 最后一条消息类型
}

// 会话列表缓存配置
const sessionListCacheTTL = 8 * time.Second // 缓存有效期8秒

// sessionListCacheEntry 缓存条目结构
type sessionListCacheEntry struct {
	ExpireAt int64             // 过期时间（纳秒）
	Data     []sessionListItem // 缓存的会话列表数据
	Total    int               // 总记录数
	Page     int               // 当前页码
	PageSize int               // 每页大小
}

const sessionListCacheMaxEntries = 256 // 缓存最大条目数

// sessionListCache 会话列表缓存
// 使用简单的内存map存储，带有过期淘汰机制
var sessionListCache = struct {
	mu      sync.RWMutex
	entries map[string]sessionListCacheEntry
}{
	entries: make(map[string]sessionListCacheEntry),
}

// makeSessionListCacheKey 生成分页列表缓存的key
// 将所有查询参数组合成一个唯一的key，用于缓存查找
func makeSessionListCacheKey(userName, role, appIDFilter, statusFilter, assignedFilter string, startTime, endTime int64, page, pageSize int) string {
	return strings.Join([]string{
		userName,
		role,
		appIDFilter,
		statusFilter,
		assignedFilter,
		strconv.FormatInt(startTime, 10),
		strconv.FormatInt(endTime, 10),
		strconv.Itoa(page),
		strconv.Itoa(pageSize),
	}, "|")
}

// getSessionListCache 获取会话列表缓存
// key: 缓存key
// 返回值：(缓存条目, 是否命中)
func getSessionListCache(key string) (sessionListCacheEntry, bool) {
	now := time.Now().UnixNano()
	sessionListCache.mu.RLock()
	entry, ok := sessionListCache.entries[key]
	sessionListCache.mu.RUnlock()
	if !ok || entry.ExpireAt < now {
		// 缓存不存在或已过期
		return sessionListCacheEntry{}, false
	}
	return entry, true
}

// setSessionListCache 设置会话列表缓存
// 如果缓存满了，会清理过期的条目，如果还有空间则淘汰最旧的条目
func setSessionListCache(key string, entry sessionListCacheEntry) {
	sessionListCache.mu.Lock()
	defer sessionListCache.mu.Unlock()

	// 如果缓存已满，先清理过期条目
	if len(sessionListCache.entries) >= sessionListCacheMaxEntries {
		now := time.Now().UnixNano()
		for k, v := range sessionListCache.entries {
			if v.ExpireAt < now {
				delete(sessionListCache.entries, k)
			}
		}
		// 如果清理后仍然满，清空整个缓存
		if len(sessionListCache.entries) >= sessionListCacheMaxEntries {
			sessionListCache.entries = make(map[string]sessionListCacheEntry)
		}
	}
	sessionListCache.entries[key] = entry
}

// invalidateSessionListCache 使会话列表缓存失效
// 在会话状态变更（接待/转接/关闭等）后调用，确保下次查询获取最新数据
func invalidateSessionListCache() {
	sessionListCache.mu.Lock()
	sessionListCache.entries = make(map[string]sessionListCacheEntry)
	sessionListCache.mu.Unlock()
	logger.Debugf("session list cache invalidated")
}

// getAgentForAuth 根据用户名获取客服信息
// 用于API权限验证，检查用户是否为合法客服
func (sc *SessionController) getAgentForAuth(userName string) (*models.User, error) {
	us := service.GetUserService()
	if us == nil {
		logger.Errorf("session auth user service unavailable user=%s", userName)
		return nil, fmt.Errorf("user service not initialized")
	}
	agent, err := us.GetUser(userName)
	if err != nil || agent == nil {
		logger.Errorf("session auth user not found user=%s err=%v", userName, err)
		return nil, fmt.Errorf("agent not found")
	}
	return agent, nil
}

// ensureAgentCanAccessSession 确保客服有权限访问指定会话
// 基于应用的访问控制：客服只能操作其所属应用下的会话
func (sc *SessionController) ensureAgentCanAccessSession(userName string, session *models.Session) bool {
	if session == nil {
		return false
	}
	agent, err := sc.getAgentForAuth(userName)
	if err != nil || agent == nil {
		return false
	}
	return isAgentForApp(agent.Apps, session.AppID())
}

// List 获取会话列表
// HTTP GET /api/v1/sessions
// 支持分页、筛选（按应用、状态、时间范围、分配状态）
// 管理员可查看所有应用的会话，客服只能查看自己应用下的会话
func (sc *SessionController) List(c *gin.Context) {
	// 获取当前用户的认证信息
	userName, role := getAuthUser(c)
	var currentAgent *models.User
	if role == "agent" || role == "admin" {
		u, err := sc.getAgentForAuth(userName)
		if err != nil || u == nil {
			logger.Errorf("session list auth context invalid user=%s role=%s", userName, role)
			response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
			return
		}
		currentAgent = u
	}

	// 解析查询参数
	appIDFilter := strings.TrimSpace(c.Query("app_id"))      // 应用ID筛选
	statusFilter := strings.TrimSpace(c.Query("status"))     // 会话状态筛选
	assignedFilter := strings.TrimSpace(c.Query("assigned")) // 分配状态筛选（mine=我的，unassigned=未分配）
	startTime, _ := strconv.ParseInt(strings.TrimSpace(c.Query("start_time")), 10, 64)
	endTime, _ := strconv.ParseInt(strings.TrimSpace(c.Query("end_time")), 10, 64)

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	noCache := strings.TrimSpace(c.Query("no_cache")) == "1" // 是否跳过缓存

	// 参数边界检查
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	// 尝试从缓存获取
	cacheKey := makeSessionListCacheKey(userName, role, appIDFilter, statusFilter, assignedFilter, startTime, endTime, page, pageSize)
	if !noCache {
		if cached, ok := getSessionListCache(cacheKey); ok {
			logger.Debugf("session list cache hit key=%s", cacheKey)
			response.ResponseSuccess(c, gin.H{
				"data":      cached.Data,
				"total":     cached.Total,
				"page":      cached.Page,
				"page_size": cached.PageSize,
			})
			return
		}
	}

	// 构建数据库查询
	query := store.DB.Model(&models.SessionListIndex{})

	// 解析应用访问权限
	if appIDs, ok := resolveAccessibleAppIDs(role, currentAgent, appIDFilter); ok {
		if len(appIDs) == 0 {
			// 客服没有任何应用权限，返回空列表
			response.ResponseSuccess(c, gin.H{
				"data":      []sessionListItem{},
				"total":     0,
				"page":      page,
				"page_size": pageSize,
			})
			return
		}
		query = query.Where("app_id IN ?", appIDs)
	} else if appIDFilter != "" {
		// 非agent/admin角色指定了app_id筛选
		query = query.Where("app_id = ?", appIDFilter)
	}

	// 状态筛选
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	// agent/admin角色额外筛选
	if role == "agent" || role == "admin" {
		if assignedFilter == "mine" {
			// 只看当前客服负责的会话
			query = query.Where("cur_agent_id = ?", userName)
		}
		if assignedFilter == "unassigned" {
			// 只看未分配的会话
			query = query.Where("(cur_agent_id IS NULL OR cur_agent_id = '')")
		}
	}

	// 时间范围筛选
	if startTime > 0 {
		query = query.Where("last_active_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("last_active_at <= ?", endTime)
	}

	// 查询总记录数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("session list count failed user=%s role=%s err=%v", userName, role, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionListFailed)
		return
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var rows []models.SessionListIndex
	if err := query.Order("last_active_at DESC").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		logger.Errorf("session list query failed user=%s role=%s err=%v", userName, role, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionListFailed)
		return
	}

	// 转换为响应格式
	pageData := make([]sessionListItem, 0, len(rows))
	for _, row := range rows {
		pageData = append(pageData, sessionListItem{
			SID:                row.SID,
			VisitorID:          row.VisitorID,
			AppID:              row.AppID,
			Status:             row.Status,
			CurAgentID:         row.CurAgentID,
			LastClientIP:       row.LastClientIP,
			LastUserAgent:      row.LastUserAgent,
			LastDevice:         row.LastDevice,
			LastGeo:            row.LastGeo,
			RatingScore:        row.RatingScore,
			RatingComment:      row.RatingComment,
			RatedAt:            row.RatedAt,
			LastVisitorMsgTime: row.LastVisitorMsgTime,
			LastAgentReplyTime: row.LastAgentReplyTime,
			LastAgentReadTime:  row.LastAgentReadTime,
			CreatedAt:          row.CreatedAt,
			UnreadCount:        row.UnreadCount,
			LastMessage:        row.LastMessage,
			LastMessageType:    row.LastMessageType,
		})
	}

	// 写入缓存
	if !noCache {
		setSessionListCache(cacheKey, sessionListCacheEntry{
			ExpireAt: time.Now().Add(sessionListCacheTTL).UnixNano(),
			Data:     pageData,
			Total:    int(total),
			Page:     page,
			PageSize: pageSize,
		})
	}

	logger.Infof("session list success user=%s role=%s total=%d page=%d page_size=%d", userName, role, total, page, pageSize)
	response.ResponseSuccess(c, gin.H{
		"data":      pageData,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// resolveAccessibleAppIDs 解析当前用户可访问的应用ID列表
// role: 用户角色
// currentAgent: 当前用户对象
// appIDFilter: 请求中指定的应用ID筛选
// 返回值：(可访问的应用ID列表, 是否应用筛选)
func resolveAccessibleAppIDs(role string, currentAgent *models.User, appIDFilter string) ([]string, bool) {
	filteredAppID := strings.TrimSpace(appIDFilter)
	if role != "agent" && role != "admin" {
		if filteredAppID != "" {
			return []string{filteredAppID}, true
		}
		return nil, false
	}
	if currentAgent == nil {
		return nil, false
	}
	if filteredAppID != "" {
		if isAgentForApp(currentAgent.Apps, filteredAppID) {
			return []string{filteredAppID}, true
		}
		return []string{}, true
	}

	// 解析用户所属应用列表
	appsRaw := strings.TrimSpace(currentAgent.Apps)
	if appsRaw == "" {
		// 用户没有任何应用权限
		return []string{}, true
	}

	// 尝试JSON解析应用列表
	var appIDs []string
	if err := json.Unmarshal([]byte(appsRaw), &appIDs); err == nil {
		normalized := make([]string, 0, len(appIDs))
		hasAll := false
		seen := make(map[string]struct{}, len(appIDs))
		for _, appID := range appIDs {
			value := strings.TrimSpace(appID)
			if value == "" {
				continue
			}
			if strings.EqualFold(value, "all") {
				// "all"表示可访问所有应用
				hasAll = true
				break
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			normalized = append(normalized, value)
		}
		if hasAll {
			return listAllAppIDs(), true
		}
		return normalized, true
	}

	// 尝试字符串匹配（兼容旧格式）
	if strings.Contains(strings.ToLower(appsRaw), "all") {
		return listAllAppIDs(), true
	}

	return nil, false
}

// listAllAppIDs 查询当前所有应用ID
// 用于"all"权限的客服，获取系统中有权限访问的所有应用列表
func listAllAppIDs() []string {
	if store.DB == nil {
		logger.Warnf("list all app ids: db not initialized")
		return nil
	}
	appIDs := make([]string, 0, 64)
	// 从会话索引表获取所有已使用的app_id
	_ = store.DB.Model(&models.SessionListIndex{}).Distinct().Where("app_id <> ''").Pluck("app_id", &appIDs).Error
	if len(appIDs) > 0 {
		return appIDs
	}
	// 从应用表获取所有app_id
	_ = store.DB.Model(&models.App{}).Where("app_id <> ''").Pluck("app_id", &appIDs).Error
	return appIDs
}

// GetMessages 按会话分页查询历史消息
// HTTP GET /api/v1/sessions/messages
// 查询参数：
//   - sid: 会话ID（必填）
//   - limit: 每页消息数量，默认50，最大100
//   - before: 分页游标，填入上一次返回的next_cursor值
//   - snapshot_ts: 快照时间戳，用于分页一致性
func (sc *SessionController) GetMessages(c *gin.Context) {
	// 获取认证信息
	userName, role := getAuthUser(c)
	sid := strings.TrimSpace(c.Query("sid"))
	if sid == "" {
		logger.Errorf("session messages sid missing")
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionIDRequired)
		return
	}

	// 解析分页参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	before := strings.TrimSpace(c.Query("before"))
	snapshotTs, _ := strconv.ParseInt(strings.TrimSpace(c.Query("snapshot_ts")), 10, 64)
	if snapshotTs <= 0 {
		snapshotTs = time.Now().Unix()
	}

	// 获取服务
	ms := service.GetMsgService()
	if ms == nil {
		logger.Errorf("session messages service unavailable sid=%s", sid)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeMessageServiceUnavailable)
		return
	}
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("session messages session service unavailable sid=%s", sid)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	// 获取会话
	session, err := ss.GetSession(sid)
	if err != nil || session == nil {
		logger.Errorf("session messages session not found sid=%s err=%v", sid, err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeSessionNotFound)
		return
	}

	// 权限检查
	if role == "agent" || role == "admin" {
		if !sc.ensureAgentCanAccessSession(userName, session) {
			logger.Errorf("session messages access denied user=%s sid=%s", userName, sid)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
			return
		}
		// 普通客服只能查看自己负责的会话
		if role == "agent" && session.CurAgentID != "" && session.CurAgentID != userName {
			logger.Errorf("session messages owner mismatch user=%s sid=%s owner=%s", userName, sid, session.CurAgentID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionOwnerMismatch)
			return
		}
	}

	// 查询历史消息
	msgs, err := ms.GetMessagesBySessionBeforeSnapshot(sid, before, limit, snapshotTs)
	if err != nil {
		logger.Errorf("session messages query failed sid=%s err=%v", sid, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionMessageQueryFailed)
		return
	}

	// 构建响应
	type msgItem struct {
		Type      string                 `json:"type"`      // 业务消息类型（agent/visitor/system）
		SID       string                 `json:"sid"`       // 会话ID
		Payload   map[string]interface{} `json:"payload"`   // 消息载荷
		Timestamp int64                  `json:"timestamp"` // 消息时间戳
		MsgID     string                 `json:"msg_id"`    // 消息ID
		Read      bool                   `json:"read"`      // 是否已读
	}
	result := make([]msgItem, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		// 构建载荷
		p := map[string]interface{}{
			"content_type": msg.MsgType,
			"content":      msg.Content,
			"timestamp":    msg.Timestamp,
			"msg_id":       msg.MsgID,
		}
		// 解析Meta字段扩展载荷
		if msg.Meta != "" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Meta), &meta); err == nil {
				for k, v := range meta {
					p[k] = v
				}
			}
		}
		// 判断消息是否已读：客服消息以客服已读时间判断，访客消息根据时间判断
		result = append(result, msgItem{
			Type:      deduceMessageBusinessType(msg),
			SID:       sid,
			Payload:   p,
			Timestamp: msg.Timestamp,
			MsgID:     msg.MsgID,
			Read:      deduceMessageBusinessType(msg) != models.WSMessageTypeVisitor || msg.Timestamp <= session.LastAgentReadTime,
		})
	}

	// 计算分页游标
	nextCursor := ""
	if len(msgs) > 0 {
		// 使用当前页最旧消息作为下一页游标，配合 service 层的严格小于过滤可避免游标重复。
		last := msgs[0]
		if len(msgs) > 1 {
			last = msgs[len(msgs)-1]
		}
		nextCursor = strings.TrimSpace(last.MsgID)
	}

	logger.Infof("session messages success sid=%s user=%s msg_count=%d", sid, userName, len(result))
	response.ResponseSuccess(c, gin.H{
		"data":        result,
		"next_cursor": nextCursor,
		"snapshot_ts": snapshotTs,
	})
}

// Accept 接待会话（将未分配会话分配给当前客服）
// HTTP POST /api/v1/sessions/accept
// 请求体：{ "sid": "会话ID" }
func (sc *SessionController) Accept(c *gin.Context) {
	userName, role := getAuthUser(c)

	// 权限检查：只有agent和admin可以接待会话
	if role != "agent" && role != "admin" {
		logger.Errorf("session accept role denied user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAgentRequired)
		return
	}

	// 解析请求参数
	var req struct {
		SID string `json:"sid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("session accept params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionAcceptInvalidParams)
		return
	}

	// 获取会话
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("session accept service unavailable sid=%s", req.SID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	// 执行接待：分配客服给会话
	now := time.Now().Unix()
	session, err := ss.UpdateSession(req.SID, func(s *models.Session) error {
		if !sc.ensureAgentCanAccessSession(userName, s) {
			return errSessionAccessDenied
		}
		if s.Closed {
			return errSessionClosedState
		}
		if s.CurAgentID != "" && s.CurAgentID != userName {
			return errSessionAlreadyTaken
		}
		s.AssignAgent(userName, now)
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errSessionAccessDenied):
			logger.Errorf("session accept access denied user=%s sid=%s", userName, req.SID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		case errors.Is(err, errSessionClosedState):
			logger.Errorf("session accept closed sid=%s", req.SID)
			response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeSessionClosed, "session closed")
		case errors.Is(err, errSessionAlreadyTaken):
			logger.Errorf("session accept already assigned sid=%s user=%s", req.SID, userName)
			response.ResponseErrorWithMsg(c, http.StatusConflict, response.ErrCodeSessionAlreadyAssigned, "already assigned")
		default:
			logger.Errorf("session accept save failed sid=%s user=%s err=%v", req.SID, userName, err)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionAcceptFailed)
		}
		return
	}

	if session == nil {
		logger.Errorf("session accept save failed sid=%s user=%s err=%v", req.SID, userName, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionAcceptFailed)
		return
	}

	// 使缓存失效并记录审计日志
	invalidateSessionListCache()
	RecordAudit(c, "session.accept", "session", req.SID, "success", "")

	logger.Infof("session accepted sid=%s agent=%s", req.SID, userName)
	response.ResponseSuccess(c, gin.H{"session": session})
}

// Transfer 将会话转接给目标客服
// HTTP POST /api/v1/sessions/transfer
// 请求体：{ "sid": "会话ID", "to_agent_name": "目标客服用户名" }
func (sc *SessionController) Transfer(c *gin.Context) {
	userName, role := getAuthUser(c)

	// 权限检查
	if role != "agent" && role != "admin" {
		logger.Errorf("session transfer role denied user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAgentRequired)
		return
	}

	// 解析请求参数
	var req struct {
		SID         string `json:"sid" binding:"required"`
		ToAgentName string `json:"to_agent_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("session transfer params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionTransferInvalidParams)
		return
	}

	// 获取会话
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("session transfer service unavailable user=%s sid=%s", userName, req.SID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}
	session, err := ss.GetSession(req.SID)
	if err != nil || session == nil {
		logger.Errorf("session transfer target not found sid=%s err=%v", req.SID, err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeSessionNotFound)
		return
	}

	// 权限检查：只能转接自己负责的会话
	if !sc.ensureAgentCanAccessSession(userName, session) {
		logger.Errorf("session transfer access denied user=%s sid=%s", userName, req.SID)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		return
	}

	// 业务规则检查：只能转接自己负责的会话
	if session.CurAgentID != userName {
		logger.Errorf("session transfer owner mismatch sid=%s owner=%s user=%s", req.SID, session.CurAgentID, userName)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionOwnerMismatch)
		return
	}

	// 验证目标客服的有效性
	us := service.GetUserService()
	target, err := us.GetUser(req.ToAgentName)
	if err != nil || target == nil || (target.Role != "agent" && target.Role != "admin") || !target.Active {
		logger.Errorf("session transfer target invalid sid=%s target=%s err=%v", req.SID, req.ToAgentName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionTransferTargetInvalid)
		return
	}
	if !isAgentForApp(target.Apps, session.AppID()) {
		logger.Errorf("session transfer target app denied sid=%s target=%s app_id=%s", req.SID, req.ToAgentName, session.AppID())
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionTransferTargetInvalid)
		return
	}

	// 执行转接
	now := time.Now().Unix()
	session, err = ss.UpdateSession(req.SID, func(s *models.Session) error {
		if !sc.ensureAgentCanAccessSession(userName, s) {
			return errSessionAccessDenied
		}
		if s.Closed {
			return errSessionClosedState
		}
		if s.CurAgentID != userName {
			return errSessionOwnerMismatch
		}
		s.AssignAgent(req.ToAgentName, now)
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errSessionAccessDenied):
			logger.Errorf("session transfer access denied user=%s sid=%s", userName, req.SID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		case errors.Is(err, errSessionClosedState):
			logger.Errorf("session transfer rejected closed sid=%s", req.SID)
			response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeSessionClosed, "session closed")
		case errors.Is(err, errSessionOwnerMismatch):
			logger.Errorf("session transfer owner mismatch sid=%s user=%s", req.SID, userName)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionOwnerMismatch)
		default:
			logger.Errorf("session transfer save failed sid=%s from=%s to=%s err=%v", req.SID, userName, req.ToAgentName, err)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionTransferFailed)
		}
		return
	}

	// 使缓存失效
	invalidateSessionListCache()

	// 发送系统消息通知转接
	systemMsg := saveSystemMessageForSession(session, "会话已转接至 "+req.ToAgentName, now)
	_ = PushMessageToVisitor(session.VisitorID(), session.SID, systemMsg)
	PushMessageToAgent(req.ToAgentName, session.SID, systemMsg) // 通知新客服
	PushMessageToAgent(userName, session.SID, systemMsg)        // 通知原客服

	// 记录审计日志
	RecordAudit(c, "session.transfer", "session", req.SID, "success", req.ToAgentName)

	logger.Infof("session transferred sid=%s from=%s to=%s", req.SID, userName, req.ToAgentName)
	response.ResponseSuccess(c, gin.H{"session": session})
}

// Close 结束会话，管理员可强制结束其他客服的会话
// HTTP POST /api/v1/sessions/close
// 请求体：{ "sid": "会话ID" }
func (sc *SessionController) Close(c *gin.Context) {
	userName, role := getAuthUser(c)

	// 权限检查
	if role != "agent" && role != "admin" {
		logger.Errorf("session close role denied user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAgentRequired)
		return
	}

	// 解析请求参数
	var req struct {
		SID string `json:"sid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("session close params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionCloseInvalidParams)
		return
	}

	// 获取会话
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("session close service unavailable user=%s sid=%s", userName, req.SID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	// 业务规则：普通客服只能关闭自己负责的会话，管理员可以关闭任意会话
	isAdmin := role == "admin"

	// 执行关闭
	session, err := ss.UpdateSession(req.SID, func(s *models.Session) error {
		if !sc.ensureAgentCanAccessSession(userName, s) {
			return errSessionAccessDenied
		}
		if s.Closed {
			return errSessionClosedState
		}
		if !isAdmin && s.CurAgentID != userName {
			return errSessionOwnerMismatch
		}
		s.Close()
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errSessionAccessDenied):
			logger.Errorf("session close access denied user=%s sid=%s", userName, req.SID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		case errors.Is(err, errSessionClosedState):
			logger.Errorf("session close rejected closed sid=%s", req.SID)
			response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeSessionClosed, "session closed")
		case errors.Is(err, errSessionOwnerMismatch):
			logger.Errorf("session close owner mismatch sid=%s user=%s", req.SID, userName)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionOwnerMismatch)
		default:
			logger.Errorf("session close save failed sid=%s user=%s err=%v", req.SID, userName, err)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionCloseFailed)
		}
		return
	}

	// 使缓存失效
	invalidateSessionListCache()

	// 构建通知内容
	now := time.Now().Unix()
	content := "会话已结束"
	if isAdmin && session.CurAgentID != "" && session.CurAgentID != userName {
		content = "管理员已强制结束会话"
	}

	// 发送系统消息
	systemMsg := saveSystemMessageForSession(session, content, now)
	_ = PushMessageToVisitor(session.VisitorID(), session.SID, systemMsg)
	PushMessageToAgent(userName, session.SID, systemMsg)

	// 记录审计日志
	RecordAudit(c, "session.close", "session", req.SID, "success", content)

	logger.Infof("session closed sid=%s user=%s is_admin=%t", req.SID, userName, isAdmin)
	response.ResponseSuccess(c, gin.H{"session": session})
}

// MarkRead 标记会话消息已读
// HTTP POST /api/v1/sessions/read
// 请求体：{ "sid": "会话ID" }
func (sc *SessionController) MarkRead(c *gin.Context) {
	userName, role := getAuthUser(c)

	// 权限检查
	if role != "agent" && role != "admin" {
		logger.Errorf("session read role denied user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAgentRequired)
		return
	}

	// 解析请求参数
	var req struct {
		SID string `json:"sid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("session read params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionReadInvalidParams)
		return
	}

	// 获取会话
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("session read service unavailable user=%s sid=%s", userName, req.SID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}

	// 执行已读标记
	now := time.Now().Unix()
	session, err := ss.UpdateSession(req.SID, func(s *models.Session) error {
		if !sc.ensureAgentCanAccessSession(userName, s) {
			return errSessionAccessDenied
		}
		if role == "agent" && s.CurAgentID != userName {
			return errSessionOwnerMismatch
		}
		s.MarkRead(now)
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errSessionAccessDenied):
			logger.Errorf("session read access denied user=%s sid=%s", userName, req.SID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		case errors.Is(err, errSessionOwnerMismatch):
			logger.Errorf("session read owner mismatch sid=%s user=%s", req.SID, userName)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionOwnerMismatch)
		default:
			logger.Errorf("session read save failed sid=%s user=%s err=%v", req.SID, userName, err)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionReadFailed)
		}
		return
	}

	// 使缓存失效
	invalidateSessionListCache()

	logger.Infof("session marked read sid=%s user=%s", req.SID, userName)
	response.ResponseSuccess(c, gin.H{"session": session})
}

// ListAgents 返回可用于当前应用会话的客服列表
// HTTP GET /api/v1/sessions/agents?app_id=xxx
func (sc *SessionController) ListAgents(c *gin.Context) {
	userName, role := getAuthUser(c)
	appID := strings.TrimSpace(c.Query("app_id"))
	if appID == "" {
		logger.Errorf("session agents app_id missing user=%s", userName)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAppConfigInvalidParams)
		return
	}

	// 权限检查：agent/admin均可查看，但agent仅可查看自己可访问应用的客服
	if role != "agent" && role != "admin" {
		logger.Errorf("session agents role denied user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAgentRequired)
		return
	}

	if role == "agent" {
		agent, err := sc.getAgentForAuth(userName)
		if err != nil || agent == nil {
			logger.Errorf("session agents auth context invalid user=%s err=%v", userName, err)
			response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
			return
		}
		if !isAgentForApp(agent.Apps, appID) {
			logger.Errorf("session agents app access denied user=%s app_id=%s", userName, appID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAppAccessDenied)
			return
		}
	}

	us := service.GetUserService()
	if us == nil {
		logger.Errorf("session agents user service unavailable user=%s app_id=%s", userName, appID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	users, err := us.ListUsers("")
	if err != nil {
		logger.Errorf("session agents list users failed user=%s app_id=%s err=%v", userName, appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionAgentsListFailed)
		return
	}

	type agentItem struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Status   int    `json:"status"`
		Active   bool   `json:"active"`
		Avatar   string `json:"avatar"`
	}
	result := make([]agentItem, 0, len(users))
	for _, u := range users {
		// 仅返回启用且在席的agent/admin
		if (u.Role != "agent" && u.Role != "admin") || !u.Active || u.Status != 1 {
			continue
		}
		if !isAgentForApp(u.Apps, appID) {
			continue
		}
		result = append(result, agentItem{
			Username: u.Username,
			Role:     u.Role,
			Status:   u.Status,
			Active:   u.Active,
			Avatar:   u.Avatar,
		})
	}

	logger.Infof("session agents listed user=%s role=%s app_id=%s count=%d", userName, role, appID, len(result))
	response.ResponseSuccess(c, gin.H{"data": result})
}

// MarkFollowUp 标记会话为需跟进
// HTTP POST /api/v1/sessions/follow-up
// 请求体：{ "sid": "会话ID" }
func (sc *SessionController) MarkFollowUp(c *gin.Context) {
	userName, role := getAuthUser(c)

	if role != "agent" && role != "admin" {
		logger.Errorf("session follow-up role denied user=%s role=%s", userName, role)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAgentRequired)
		return
	}

	var req struct {
		SID string `json:"sid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("session follow-up params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionReadInvalidParams)
		return
	}

	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("session follow-up service unavailable user=%s sid=%s", userName, req.SID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}
	session, err := ss.GetSession(req.SID)
	if err != nil || session == nil {
		logger.Errorf("session follow-up target not found sid=%s err=%v", req.SID, err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeSessionNotFound)
		return
	}

	if !sc.ensureAgentCanAccessSession(userName, session) {
		logger.Errorf("session follow-up access denied user=%s sid=%s", userName, req.SID)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		return
	}

	// 普通客服仅可标记自己负责的会话；管理员可标记任何会话
	if role == "agent" && session.CurAgentID != userName {
		logger.Errorf("session follow-up owner mismatch sid=%s owner=%s user=%s", req.SID, session.CurAgentID, userName)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionOwnerMismatch)
		return
	}

	if session.Closed {
		logger.Errorf("session follow-up rejected closed sid=%s", req.SID)
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeSessionClosed, "session closed")
		return
	}

	session, err = ss.UpdateSession(req.SID, func(s *models.Session) error {
		if !sc.ensureAgentCanAccessSession(userName, s) {
			return errSessionAccessDenied
		}
		if role == "agent" && s.CurAgentID != userName {
			return errSessionOwnerMismatch
		}
		if s.Closed {
			return errSessionClosedState
		}
		s.MarkFollowUp()
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errSessionAccessDenied):
			logger.Errorf("session follow-up access denied user=%s sid=%s", userName, req.SID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		case errors.Is(err, errSessionOwnerMismatch):
			logger.Errorf("session follow-up owner mismatch sid=%s user=%s", req.SID, userName)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionOwnerMismatch)
		case errors.Is(err, errSessionClosedState):
			logger.Errorf("session follow-up rejected closed sid=%s", req.SID)
			response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeSessionClosed, "session closed")
		default:
			logger.Errorf("session follow-up save failed sid=%s user=%s err=%v", req.SID, userName, err)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionReadFailed)
		}
		return
	}

	invalidateSessionListCache()
	RecordAudit(c, "session.follow_up", "session", req.SID, "success", "")
	logger.Infof("session follow-up marked sid=%s user=%s", req.SID, userName)
	response.ResponseSuccess(c, gin.H{"session": session})
}

// Rate 访客评价会话
// HTTP POST /api/v1/sessions/rate
// 请求体：{ "sid": "会话ID", "score": 5, "comment": "服务很好" }
func (sc *SessionController) Rate(c *gin.Context) {
	userName, _ := getAuthUser(c)

	// 解析请求参数
	var req struct {
		SID       string `json:"sid" binding:"required"`
		AppID     string `json:"app_id" binding:"required"`
		VisitorID string `json:"visitor_id" binding:"required"`
		Score     int    `json:"score" binding:"required,min=1,max=5"`
		Comment   string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("session rate params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionRateInvalidParams)
		return
	}

	// 获取会话
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("session rate service unavailable sid=%s", req.SID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionServiceUnavailable)
		return
	}
	session, err := ss.GetSession(req.SID)
	if err != nil || session == nil {
		logger.Errorf("session rate target not found sid=%s err=%v", req.SID, err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeSessionNotFound)
		return
	}
	req.AppID = strings.TrimSpace(req.AppID)
	req.VisitorID = strings.TrimSpace(req.VisitorID)
	if session.AppID() != req.AppID || session.VisitorID() != req.VisitorID {
		logger.Errorf("session rate owner mismatch sid=%s app_id=%s visitor_id=%s", req.SID, req.AppID, req.VisitorID)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		return
	}

	app := models.GetApp(req.AppID)
	if app == nil {
		logger.Errorf("session rate app not found sid=%s app_id=%s", req.SID, req.AppID)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeAppNotFound)
		return
	}
	origin := c.GetHeader("Origin")
	referer := c.GetHeader("Referer")
	if !models.IsDomainAllowed(origin, referer, app.AllowDomain) {
		logger.Errorf("session rate origin forbidden sid=%s app_id=%s origin=%s referer=%s", req.SID, req.AppID, origin, referer)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAppAccessDenied)
		return
	}

	// 业务规则检查：只能评价一次
	// 执行评分
	session, err = ss.UpdateSession(req.SID, func(s *models.Session) error {
		if s.AppID() != req.AppID || s.VisitorID() != req.VisitorID {
			return errSessionAccessDenied
		}
		if s.RatingScore > 0 {
			return errSessionAlreadyRated
		}
		if !s.Rate(req.Score, req.Comment, time.Now().Unix()) {
			return errSessionInvalidRating
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errSessionAccessDenied):
			logger.Errorf("session rate owner mismatch sid=%s app_id=%s visitor_id=%s", req.SID, req.AppID, req.VisitorID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSessionAccessDenied)
		case errors.Is(err, errSessionAlreadyRated):
			logger.Errorf("session rate already rated sid=%s", req.SID)
			response.ResponseErrorWithMsg(c, http.StatusConflict, response.ErrCodeSessionRateAlreadyDone, "session already rated")
		case errors.Is(err, errSessionInvalidRating):
			logger.Errorf("session rate score invalid sid=%s score=%d", req.SID, req.Score)
			response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSessionRateInvalidScore)
		default:
			logger.Errorf("session rate save failed sid=%s err=%v", req.SID, err)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionRateFailed)
		}
		return
	}

	// 使缓存失效
	invalidateSessionListCache()

	logger.Infof("session rated sid=%s score=%d comment=%s", req.SID, req.Score, req.Comment)
	response.ResponseSuccess(c, gin.H{"session": session})
}

// deduceMessageBusinessType 根据消息的Meta字段推断消息的业务类型
// 用于区分消息是来自客服、访客还是系统
// msg: 消息对象
// 返回值：消息业务类型（agent/visitor/system）
func deduceMessageBusinessType(msg *models.Message) string {
	if msg == nil {
		return models.WSMessageTypeSystem
	}
	// 系统消息直接返回
	if msg.MsgType == models.WSMessageTypeSystem {
		return models.WSMessageTypeSystem
	}
	// 解析Meta字段获取from属性
	if msg.Meta != "" {
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Meta), &meta); err == nil {
			if from, ok := meta["from"].(string); ok {
				if from == "agent" {
					return models.WSMessageTypeAgent
				}
				if from == "visitor" {
					return models.WSMessageTypeVisitor
				}
			}
		}
	}
	// 默认为访客消息
	return models.WSMessageTypeVisitor
}

// isAgentForApp 检查客服是否有权限访问指定应用
// appsJSON: 客服所属应用的JSON数组字符串
// appID: 要检查的应用ID
// 返回值：是否有权限
func isAgentForApp(appsJSON, appID string) bool {
	if appsJSON == "" {
		return false
	}
	// 尝试JSON解析
	var apps []string
	if err := json.Unmarshal([]byte(appsJSON), &apps); err != nil {
		// 解析失败，尝试字符串匹配
		return strings.Contains(strings.ToLower(appsJSON), "all") ||
			strings.Contains(strings.ToLower(appsJSON), strings.ToLower(appID))
	}
	// 遍历检查是否有"all"权限或匹配的appID
	for _, app := range apps {
		if strings.EqualFold(strings.TrimSpace(app), "all") || strings.EqualFold(strings.TrimSpace(app), appID) {
			return true
		}
	}
	return false
}
