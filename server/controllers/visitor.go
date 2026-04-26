package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/utils"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// WebSocket消息类型常量定义
// 用于区分不同类型的WebSocket消息，便于在消息处理流程中进行路由分发
const (
	MessageTypeReq    = models.WSMessageTypeVisitor // 访客发送的消息类型
	MessageTypeSystem = models.WSMessageTypeSystem  // 系统消息类型（如欢迎语、会话状态通知等）
	MessageTypeTyping = models.WSMessageTypeTyping  // 正在输入状态通知
	MessageTypeClose  = models.WSMessageTypeClose   // 会话关闭消息类型

	// 同一会话新连接建立后，旧连接会被替换关闭。
	// 使用自定义关闭码，前端可识别并停止自动重连，避免连接风暴。
	VisitorConnReplacedCloseCode websocket.StatusCode = 4001
)

// VisitorConn 封装单个访客会话的WebSocket连接状态
// 每个访客连接包含：底层WebSocket连接、会话ID、发送通道（用于异步消息发送）、完成通道（用于连接生命周期管理）
type VisitorConn struct {
	Conn      *websocket.Conn // 底层的WebSocket连接对象
	SessionID string          // 当前连接的会话唯一标识，格式为 s:{visitor_id}:{app_id}:{session_seq}
	SendChan  chan []byte     // 异步消息发送通道，writeLoop从此通道读取消息并发送给访客
	Done      chan struct{}   // 连接生命周期管理通道，连接断开时关闭
}

// 全局访客连接池：session_id => *VisitorConn
// 使用sessionID作为key，可以快速查找某个会话的连接状态
// 注意：这里采用简单map设计，一个sessionID对应一个连接
// 如需支持多端同时在线，可参考agentConns的map[string]map[*AgentConn]struct{}设计
var (
	visitorConns = make(map[string]*VisitorConn) // 会话ID到连接的映射表
	visitorMu    sync.RWMutex                    // 读写锁，保护visitorConns的并发访问

	aiBotReplyInflight   = make(map[string]struct{})
	aiBotReplyInflightMu sync.Mutex
)

// PushTypingToVisitor 向访客推送"客服正在输入"事件
// 当客服开始输入消息时，会调用此函数通知访客端显示输入提示
// visitorID: 访客唯一标识（当前未使用，保留参数）
// sessionID: 会话唯一标识，用于定位访客连接
func PushTypingToVisitor(visitorID, sessionID string) {
	visitorMu.RLock()
	conn, ok := visitorConns[sessionID]
	visitorMu.RUnlock()
	if !ok || conn == nil {
		logger.Debugf("visitor typing skip: connection not found sid=%s", sessionID)
		return
	}

	// 构建typing通知的消息体
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"from":      "agent",           // 标识消息来源为客服
		"timestamp": time.Now().Unix(), // 当前时间戳
	})

	// 构建WebSocket数据包
	packet := models.WSPacket{
		Type:      models.WSMessageTypeTyping, // 消息类型为typing
		SID:       utils.Base58Encode([]byte(sessionID)),
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}
	body, _ := json.Marshal(packet)

	// 尝试非阻塞发送消息到SendChan
	// 如果发送缓冲区满（channel容量为128），说明访客端读取速度跟不上，跳过本次推送
	select {
	case conn.SendChan <- body:
		logger.Debugf("visitor typing sent sid=%s", sessionID)
	default:
		logger.Warnf("visitor typing send buffer full sid=%s", sessionID)
	}
}

// registerVisitorConn 注册访客WebSocket连接到连接池
// sessionID: 会话唯一标识
// conn: 访客连接对象
// 此函数在访客成功建立WebSocket连接后调用，将连接加入全局连接池以便后续消息推送
func registerVisitorConn(sessionID string, conn *VisitorConn) {
	visitorMu.Lock()
	oldConn := visitorConns[sessionID]
	visitorConns[sessionID] = conn
	visitorMu.Unlock()

	// 同一会话快速重连时，用新连接替换旧连接，并主动关闭旧连接。
	// 旧连接的defer注销会通过实例比对避免误删新连接。
	if oldConn != nil && oldConn != conn {
		_ = oldConn.Conn.Close(VisitorConnReplacedCloseCode, "replaced_by_newer_connection")
	}
	logger.Infof("visitor conn registered sid=%s", sessionID)
}

// unregisterVisitorConnInstance 按连接实例注销访客WebSocket连接
// sessionID: 会话唯一标识
// conn: 具体连接实例
// 仅当当前map中的连接与传入实例一致时才删除，避免旧连接误删新连接
func unregisterVisitorConnInstance(sessionID string, conn *VisitorConn) {
	visitorMu.Lock()
	cur := visitorConns[sessionID]
	if cur == conn {
		delete(visitorConns, sessionID)
	}
	visitorMu.Unlock()
	logger.Infof("visitor conn unregistered sid=%s matched=%t", sessionID, cur == conn)
}

// PushMessageToVisitor 向访客推送消息（由客服侧或系统侧调用）
// 这是消息推送的核心函数，客服发送消息或系统发送通知时都调用此函数
// visitorID: 访客唯一标识（当前未使用，通过sessionID定位）
// sessionID: 会话唯一标识，用于定位访客连接
// msg: 要推送的消息对象
// 返回值：如果成功发送到缓冲区返回nil，访客离线返回nil（触发离线通知），其他错误返回error
func PushMessageToVisitor(visitorID, sessionID string, msg *models.Message) error {
	visitorMu.RLock()
	conn, ok := visitorConns[sessionID]
	visitorMu.RUnlock()

	// 连接不存在，说明访客已离线
	if !ok || conn == nil {
		logger.Warnf("visitor offline skip push sid=%s msg_id=%s", sessionID, msg.MsgID)
		// 访客离线时触发外部通知（如有配置webhook）
		if msg != nil {
			notifyOfflineDevice(sessionID, msg)
		}
		return nil
	}

	// 对sessionID进行Base58编码用于WebSocket传输
	sidEncoded := utils.Base58Encode([]byte(sessionID))

	// 根据消息类型确定packetType
	// 客服消息使用agent类型，系统消息使用system类型
	packetType := models.WSMessageTypeAgent
	if msg.MsgType == models.WSMessageTypeSystem {
		packetType = models.WSMessageTypeSystem
	}

	// 构建WebSocket数据包
	packet := models.BuildOutgoingWSPacket(packetType, msg, sidEncoded)
	payload, _ := json.Marshal(packet)

	// 尝试非阻塞发送消息到SendChan
	select {
	case conn.SendChan <- payload:
		logger.Debugf("visitor msg sent sid=%s msg_id=%s type=%s", sessionID, msg.MsgID, msg.MsgType)
	default:
		// 发送缓冲区满，说明访客端网络可能拥塞或读取速度慢
		logger.Warnf("visitor send buffer full sid=%s msg_id=%s", sessionID, msg.MsgID)
	}
	return nil
}

// IsVisitorSessionOnline 判断访客会话是否在线
// sessionID: 会话唯一标识
// 返回值：在线返回true，离线返回false
// 判断依据：连接池中存在且Done通道未关闭
func IsVisitorSessionOnline(sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		logger.Debugf("visitor online check: empty session id")
		return false
	}
	visitorMu.RLock()
	conn, ok := visitorConns[sessionID]
	visitorMu.RUnlock()
	if !ok || conn == nil {
		logger.Debugf("visitor offline: connection not found sid=%s", sessionID)
		return false
	}
	// 检查Done通道是否已关闭（连接是否断开）
	select {
	case <-conn.Done:
		logger.Debugf("visitor offline: connection closed sid=%s", sessionID)
		return false
	default:
	}
	return true
}

// notifyOfflineDevice 在访客离线时触发外部通知webhook
// 当消息无法推送给访客（因为连接已断开）时，调用此函数向配置的webhook地址发送通知
// sessionID: 会话唯一标识
// msg: 未能送达的消息对象
// webhook通知内容包含：事件类型、会话ID、消息ID、消息类型、内容、时间戳
func notifyOfflineDevice(sessionID string, msg *models.Message) {
	cfg := getSystemSettingsCached()
	webhook := strings.TrimSpace(cfg.OfflineNotifyURL)
	// 没有配置webhook或消息为空，直接返回
	if webhook == "" || msg == nil {
		logger.Debugf("offline notify skipped: webhook=%s msg_nil=%t", webhook == "", msg == nil)
		return
	}

	// 构建webhook请求体
	body := map[string]interface{}{
		"event":      "visitor.offline.message", // 事件类型标识
		"session_id": sessionID,
		"msg_id":     msg.MsgID,
		"msg_type":   msg.MsgType,
		"content":    msg.Content,
		"timestamp":  msg.Timestamp,
	}
	payload, _ := json.Marshal(body)

	// 异步发送webhook请求，使用goroutine避免阻塞主流程
	go func() {
		client := &http.Client{Timeout: 3 * time.Second} // 3秒超时
		maxAttempts := 3                                 // 最多重试3次

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(payload))
			if err != nil {
				// webhook URL格式错误，无法构建请求
				logger.Errorf("offline notify request build failed sid=%s err=%v", sessionID, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err == nil {
				// 成功发送，读取并关闭响应体
				_, _ = io.Copy(io.Discard, resp.Body)
				statusCode := resp.StatusCode
				_ = resp.Body.Close()
				if statusCode >= 200 && statusCode < 300 {
					// 2xx状态码表示成功
					logger.Infof("offline notify success sid=%s attempt=%d status=%d", sessionID, attempt, statusCode)
					return
				}
				// 非2xx状态码，记录错误并重试
				logger.Errorf("offline notify request status invalid sid=%s attempt=%d status=%d", sessionID, attempt, statusCode)
			} else {
				// 网络错误，记录错误并重试
				logger.Errorf("offline notify request failed sid=%s attempt=%d err=%v", sessionID, attempt, err)
			}

			// 指数退避重试：1次失败等待200ms，2次失败等待400ms
			if attempt < maxAttempts {
				backoff := time.Duration(1<<(attempt-1)) * 200 * time.Millisecond
				logger.Debugf("offline notify retry after %v sid=%s", backoff, sessionID)
				time.Sleep(backoff)
			}
		}
	}()
}

// parseNotifyEmails 解析邮件通知接收人列表
// 支持多种分隔符：逗号、分号、换行符、中文逗号、中文分号
// raw: 原始邮件地址字符串，可能包含多种分隔符
// 返回值：去重后的邮件地址列表
func parseNotifyEmails(raw string) []string {
	// 使用FieldsFunc按多种分隔符分割字符串
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '，' || r == '；'
	})

	// 去重处理
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		email := strings.TrimSpace(part)
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			// 重复邮箱，跳过
			continue
		}
		seen[email] = struct{}{}
		result = append(result, email)
	}
	return result
}

// notifyNewSessionByEmail 当有新会话创建时，发送邮件通知给管理员
// session: 新会话对象
// msg: 会话的第一条消息对象
// 邮件内容包含：会话ID、访客ID、应用ID、消息类型、消息内容、时间
// 此函数异步执行，不阻塞主流程
func notifyNewSessionByEmail(session *models.Session, msg *models.Message) {
	// 参数校验
	if session == nil || msg == nil {
		logger.Warnf("new session email notify skipped: session or msg is nil")
		return
	}

	// 获取系统配置，检查是否启用邮件通知
	cfg := getSystemSettingsCached()
	if !cfg.EmailNotify {
		logger.Debugf("new session email notify skipped: email notify disabled")
		return
	}

	// 解析邮件接收人列表
	recipients := parseNotifyEmails(cfg.NotifyEmail)
	if len(recipients) == 0 {
		logger.Warnf("new session email notify skipped sid=%s reason=empty_recipients", session.SID)
		return
	}

	// 解析会话信息用于邮件正文
	visitorID, appID, _ := session.ParseSid()
	subject := fmt.Sprintf("[kefu] new session %s", session.SID)

	// 构建邮件正文（纯文本格式）
	body := strings.Join([]string{
		"new visitor session incoming",
		"sid: " + session.SID,
		"app_id: " + appID,
		"visitor_id: " + visitorID,
		"content_type: " + strings.TrimSpace(msg.MsgType),
		"content: " + strings.TrimSpace(msg.Content),
		"time: " + time.Unix(msg.Timestamp, 0).Format(time.RFC3339),
	}, "\r\n")

	// 设置发件人地址
	from := cfg.SMTPFrom
	if from == "" {
		from = "kefu@localhost"
	}

	// 构建原始邮件内容（MIME格式）
	raw := "To: " + strings.Join(recipients, ",") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body

	// 异步发送邮件，不阻塞主流程
	go func() {
		smtpAddr := cfg.SMTPHost
		smtpPort := cfg.SMTPPort

		// SMTP地址为空时使用默认值
		if smtpAddr == "" {
			smtpAddr = "127.0.0.1"
		}
		// SMTP端口为0或负数时使用默认值25
		if smtpPort <= 0 {
			smtpPort = 25
		}

		addr := fmt.Sprintf("%s:%d", smtpAddr, smtpPort)

		// 配置SMTP认证（如果提供了用户名）
		var auth smtp.Auth
		if cfg.SMTPUser != "" {
			auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, smtpAddr)
		}

		// 发送邮件
		if err := smtp.SendMail(addr, auth, from, recipients, []byte(raw)); err != nil {
			logger.Errorf("new session email notify failed sid=%s addr=%s err=%v", session.SID, addr, err)
			return
		}
		logger.Infof("new session email notify sent sid=%s recipients=%d addr=%s", session.SID, len(recipients), addr)
	}()
}

// notifyAgentUnavailableChannels 在“客服不在线或不在席”时，触发统一通知渠道：
// 1) 邮件通知（全局收件人）
// 2) 企微通知（仅发送给当前会话负责客服，且该客服已绑定企微）
func notifyAgentUnavailableChannels(session *models.Session, msg *models.Message) {
	if session == nil || msg == nil {
		return
	}
	agentName := strings.TrimSpace(session.CurAgentID)
	if agentName == "" {
		return
	}

	userService := service.GetUserService()
	if userService == nil {
		return
	}
	agent, err := userService.GetUser(agentName)
	if err != nil || agent == nil {
		logger.Errorf("notify unavailable channels agent not found sid=%s agent=%s err=%v", session.SID, agentName, err)
		return
	}

	online := len(getAgentConnSnapshot(agentName)) > 0
	onSeat := agent.Status == 1
	if online && onSeat {
		return
	}

	// 邮件通知：全局收件人
	notifyNewSessionByEmail(session, msg)

	// 企微通知：仅当前客服（已绑定时才会实际发送）
	reason := "离线"
	if online && !onSeat {
		reason = "离席"
	} else if !online && onSeat {
		reason = "未登录"
	} else if !online && !onSeat {
		reason = "未登录且离席"
	}
	title := "访客有新消息"
	content := strings.Join([]string{
		"会话ID: " + session.SID,
		"客服: " + agentName,
		"状态: " + reason,
		"消息类型: " + strings.TrimSpace(msg.MsgType),
		"消息内容: " + strings.TrimSpace(msg.Content),
	}, "\n")
	if err := SendNotificationToAgent(agent.ID, title, content); err != nil {
		logger.Errorf("notify unavailable channels send wecom failed sid=%s agent=%s err=%v", session.SID, agentName, err)
	}
}

// VisitorController 访客端控制器
// 负责处理访客相关的HTTP请求和WebSocket连接
type VisitorController struct{}

type aiBotTestRequest struct {
	AppID string `json:"app_id"`
	Query string `json:"query"`
}

// AIBotTest 测试 SDK 自动问答机器人输出。
// HTTP POST /api/v1/ai/bot-test
func (vc *VisitorController) AIBotTest(c *gin.Context) {
	var req aiBotTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}
	appID := strings.TrimSpace(req.AppID)
	query := strings.TrimSpace(req.Query)
	if appID == "" || query == "" {
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeInvalidParams, "app_id/query required")
		return
	}

	cfg := getSystemSettingsCached()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	answer := strings.TrimSpace(vc.generateAIBotAnswer(ctx, &models.Session{SID: models.GetSessionID("ai_test", appID, 0)}, query, cfg, "", nil))
	if answer == "" {
		answer = "未生成有效答案，请检查知识库或模型配置。"
	}

	response.ResponseSuccess(c, gin.H{
		"suggestion": answer,
	})
}

// History 返回访客可见的当前会话历史消息
// 用于SDK刷新后恢复会话，或访客端拉取历史消息
// HTTP GET /api/v1/visitor/history
// 查询参数：
//   - visitor_id: 访客唯一标识（必填）
//   - app_id: 应用唯一标识（必填）
//   - limit: 每页消息数量，默认50，最大100（可选）
//   - before: 分页游标，填入上一次返回的next_cursor值（可选）
func (vc *VisitorController) History(c *gin.Context) {
	// 解析查询参数
	visitorID := strings.TrimSpace(c.Query("visitor_id"))
	appID := strings.TrimSpace(c.Query("app_id"))
	limit, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("limit", "50")))
	before := strings.TrimSpace(c.Query("before"))

	// 校验limit范围，防止滥用
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// 必填参数校验
	if visitorID == "" || appID == "" {
		logger.Errorf("visitor history params missing visitor_id=%s app_id=%s", visitorID, appID)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeInvalidParams)
		return
	}

	// 校验请求来源域名是否在应用白名单内
	origin := c.GetHeader("Origin")
	referer := c.GetHeader("Referer")
	if !vc.isValidOrigin(appID, origin, referer) {
		logger.Errorf("visitor history origin forbidden app_id=%s origin=%s referer=%s", appID, origin, referer)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeSecurityDomainNotAllowed)
		return
	}

	// 获取会话服务和消息服务
	ss := service.GetSessionService()
	ms := service.GetMsgService()
	if ss == nil || ms == nil {
		logger.Errorf("visitor history service unavailable visitor_id=%s app_id=%s", visitorID, appID)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeServiceUnavailable)
		return
	}

	// 获取或创建会话
	session, err := ss.GetOrCreateSession(visitorID, appID)
	if err != nil || session == nil {
		logger.Errorf("visitor history get session failed visitor_id=%s app_id=%s err=%v", visitorID, appID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeSessionNotFound)
		return
	}

	// 获取历史消息（使用快照时间避免并发写入导致翻页抖动）
	msgs, err := ms.GetMessagesBySessionBeforeSnapshot(session.SID, before, limit, time.Now().Unix())
	if err != nil {
		logger.Errorf("visitor history get messages failed sid=%s err=%v", session.SID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeMessageFetchFailed)
		return
	}

	// 转换消息格式，构建API响应
	type item struct {
		Type      string                 `json:"type"`      // 业务消息类型（agent/visitor/system）
		SID       string                 `json:"sid"`       // 会话ID
		Payload   map[string]interface{} `json:"payload"`   // 消息载荷
		Timestamp int64                  `json:"timestamp"` // 消息时间戳
		MsgID     string                 `json:"msg_id"`    // 消息唯一ID
	}
	rows := make([]item, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		// 根据Meta字段中的from属性判断消息的业务类型
		bizType := deduceMessageBusinessType(msg)

		// 构建消息载荷
		payload := map[string]interface{}{
			"content_type": msg.MsgType,
			"content":      msg.Content,
			"timestamp":    msg.Timestamp,
			"msg_id":       msg.MsgID,
		}

		// 解析Meta字段，扩展payload内容
		if strings.TrimSpace(msg.Meta) != "" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Meta), &meta); err == nil {
				for k, v := range meta {
					payload[k] = v
				}
			}
		}
		rows = append(rows, item{
			Type:      bizType,
			SID:       session.SID,
			Payload:   payload,
			Timestamp: msg.Timestamp,
			MsgID:     msg.MsgID,
		})
	}

	// 计算分页游标：如果有消息，第一条消息的msgID作为下次查询的游标
	nextCursor := ""
	if len(msgs) > 0 {
		nextCursor = msgs[0].MsgID
	}

	logger.Infof("visitor history success sid=%s visitor_id=%s app_id=%s msg_count=%d", session.SID, visitorID, appID, len(rows))
	response.ResponseSuccess(c, gin.H{
		"sid":         session.SID,
		"messages":    rows,
		"next_cursor": nextCursor,
	})
}

// WSHandler 处理访客WebSocket连接与初始化流程
// HTTP GET /ws/chat?visitor_id=xxx&app_id=xxx
// 这是访客端建立WebSocket长连接的入口函数
// 流程：参数校验 -> 域名校验 -> 获取/创建会话 -> 建立WebSocket连接 -> 注册连接 -> 发送欢迎语 -> 推送离线消息 -> 启动读写循环
func (vc *VisitorController) WSHandler(c *gin.Context) {
	logger.Infof("visitor websocket connect attempt remote=%s", c.Request.RemoteAddr)

	// 解析查询参数
	visitorID := strings.TrimSpace(c.Query("visitor_id"))
	appID := strings.TrimSpace(c.Query("app_id"))

	logger.Infof("visitor websocket params visitor_id=%s app_id=%s", visitorID, appID)

	// 必填参数校验
	if visitorID == "" || appID == "" {
		logger.Errorf("visitor websocket params missing visitor_id=%s app_id=%s", visitorID, appID)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 获取HTTP头信息用于域名校验
	origin := c.GetHeader("Origin")
	referer := c.GetHeader("Referer")
	logger.Infof("visitor websocket headers origin=%s referer=%s", origin, referer)

	// 校验请求来源域名是否在应用白名单内
	if !vc.isValidOrigin(appID, origin, referer) {
		logger.Errorf("visitor websocket origin forbidden app_id=%s origin=%s referer=%s", appID, origin, referer)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// 获取会话服务
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("visitor session service unavailable")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 获取或创建会话
	// 如果存在未超时的未关闭会话则复用，否则创建新会话
	session, err := ss.GetOrCreateSession(visitorID, appID)
	if err != nil {
		logger.Errorf("visitor get or create session failed visitor_id=%s app_id=%s err=%v", visitorID, appID, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	logger.Infof("visitor websocket session ready sid=%s", session.SID)

	// 构建WebSocket Accept选项
	acceptOptions := &websocket.AcceptOptions{}
	originPatterns := buildVisitorOriginPatterns(origin, referer)
	if len(originPatterns) > 0 {
		acceptOptions.OriginPatterns = originPatterns
		logger.Infof("visitor websocket accept origin patterns sid=%s patterns=%v", session.SID, originPatterns)
	}

	// 接受WebSocket连接
	conn, err := websocket.Accept(c.Writer, c.Request, acceptOptions)
	if err != nil {
		logger.Errorf("visitor websocket accept failed sid=%s err=%v", session.SID, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer conn.CloseNow()

	logger.Infof("visitor websocket accepted sid=%s", session.SID)

	// 创建访客连接对象
	visitorConn := &VisitorConn{
		Conn:      conn,
		SessionID: session.SID,
		SendChan:  make(chan []byte, 128), // 128容量缓冲
		Done:      make(chan struct{}),
	}

	// 提取访客设备信息
	clientIP := strings.TrimSpace(c.ClientIP())
	userAgent := strings.TrimSpace(c.GetHeader("User-Agent"))
	device := detectDevice(userAgent)
	// 优先从X-Forwarded-Country头获取国家信息，其次从CF-IPCountry（Cloudflare）获取
	geo := strings.TrimSpace(c.GetHeader("X-Forwarded-Country"))
	if geo == "" {
		geo = strings.TrimSpace(c.GetHeader("CF-IPCountry"))
	}

	// 更新会话的访客设备信息
	session.TouchVisitorProfile(clientIP, userAgent, device, geo)
	_ = ss.SaveSession(session)
	invalidateSessionListCache()

	// 将连接注册到全局连接池
	registerVisitorConn(session.SID, visitorConn)
	defer unregisterVisitorConnInstance(session.SID, visitorConn)

	// 如果会话没有消息记录（首次访问），发送欢迎语。
	// 欢迎语只作为当前连接的临时提示，不写入历史消息，避免固定 visitor
	// 在复用会话时反复回放很早之前的欢迎语。
	if session.LastMessage == "" {
		app := models.GetApp(session.AppID())
		if app != nil && strings.TrimSpace(app.WelcomeMsg) != "" {
			now := time.Now().Unix()
			welPayload := map[string]interface{}{
				"from":         "system",
				"from_name":    "系统",
				"sender_name":  "系统",
				"content_type": models.WSContentTypeText,
				"content":      app.WelcomeMsg,
				"timestamp":    now,
			}
			wb, _ := json.Marshal(welPayload)
			welMsg := models.Message{
				MsgType:   models.WSMessageTypeSystem,
				Content:   app.WelcomeMsg,
				Meta:      string(wb),
				Timestamp: now,
			}
			_ = PushMessageToVisitor(session.VisitorID(), session.SID, &welMsg)
		}
	}

	// 推送访客离线期间的消息（如果有的话）
	vc.pushOfflineMessages(session)

	// 创建上下文用于管理读写协程的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动读写协程
	// readLoop: 读取访客发送的消息
	// writeLoop: 向访客发送消息（从SendChan读取）并发送ping保活
	go vc.readLoop(ctx, visitorConn)
	go vc.writeLoop(ctx, visitorConn)

	// 等待连接断开信号
	<-visitorConn.Done
	logger.Infof("visitor websocket disconnected sid=%s", session.SID)
}

// readLoop 负责读取访客发来的WebSocket消息
// 这是一个阻塞函数，会持续从WebSocket连接读取消息直到连接断开或发生错误
// ctx: 上下文对象，用于接收取消信号
// vconn: 访客连接对象
func (vc *VisitorController) readLoop(ctx context.Context, vconn *VisitorConn) {
	defer close(vconn.Done)

	for {
		// 设置120秒读超时，防止长时间无数据导致的连接僵死
		readCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		_, data, err := vconn.Conn.Read(readCtx)
		cancel()

		if err != nil {
			// 检查是否为正常断开
			if websocket.CloseStatus(err) == -1 {
				// -1表示非正常关闭（如网络中断、进程崩溃等）
				logger.Errorf("visitor websocket read failed sid=%s err=%v", vconn.SessionID, err)
			} else {
				// 正常关闭，连接断开是预期行为
				logger.Infof("visitor websocket read done sid=%s status=%v", vconn.SessionID, websocket.CloseStatus(err))
			}
			return
		}

		// 解析WebSocket消息包
		var req models.WSPacket
		if err := json.Unmarshal(data, &req); err != nil {
			// JSON解析失败，忽略这条消息
			logger.Debugf("visitor websocket json parse failed sid=%s err=%v", vconn.SessionID, err)
			continue
		}

		// 空消息类型，忽略
		if req.Type == "" {
			logger.Debugf("visitor websocket empty message type sid=%s", vconn.SessionID)
			continue
		}

		// 收到关闭消息，正常关闭连接
		if req.Type == MessageTypeClose {
			logger.Infof("visitor websocket close received sid=%s", vconn.SessionID)
			return
		}

		// 空载荷，忽略
		if len(req.Payload) == 0 {
			logger.Debugf("visitor websocket empty payload sid=%s type=%s", vconn.SessionID, req.Type)
			continue
		}

		// 处理消息确认（访客收到消息后发送的ACK）
		if req.Type == models.WSMessageTypeAck {
			vc.handleAck(vconn.SessionID, req.Payload)
			continue
		}

		// 处理普通消息（文本、图片、语音、文件等）
		vc.handleMessage(vconn.SessionID, req.Type, req.Payload)
	}
}

// writeLoop 负责把待发送消息写回访客WebSocket连接，并定时发送ping保活
// 这个函数从SendChan通道读取消息并发送给访客，同时每25秒发送一次ping探测连接状态
// ctx: 上下文对象，用于接收取消信号
// vconn: 访客连接对象
func (vc *VisitorController) writeLoop(ctx context.Context, vconn *VisitorConn) {
	// 每25秒发送一次ping保活
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文被取消，退出循环
			logger.Debugf("visitor write loop context cancelled sid=%s", vconn.SessionID)
			return
		case <-vconn.Done:
			// 连接已断开，退出循环
			logger.Debugf("visitor write loop connection done sid=%s", vconn.SessionID)
			return
		case msg := <-vconn.SendChan:
			// 有消息需要发送
			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := vconn.Conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				// 写入失败，连接可能已断开
				logger.Errorf("visitor websocket write failed sid=%s err=%v", vconn.SessionID, err)
				return
			}
		case <-ticker.C:
			// 定时ping探测
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := vconn.Conn.Ping(pingCtx); err != nil {
				cancel()
				// ping失败，连接可能已断开
				logger.Errorf("visitor websocket ping failed sid=%s err=%v", vconn.SessionID, err)
				return
			}
			cancel()
		}
	}
}

// handleAck 处理访客的消息确认（ACK）
// 当消息成功推送给访客后，访客会发送ACK确认，这里更新会话的最后确认消息ID
// sessionID: 会话唯一标识
// payloadBytes: ACK消息的载荷数据
func (vc *VisitorController) handleAck(sessionID string, payloadBytes []byte) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || len(payloadBytes) == 0 {
		logger.Debugf("visitor ack skipped: empty session_id or payload")
		return
	}

	// 解析ACK载荷
	payload, err := models.ParseWSAckPayload(payloadBytes)
	if err != nil {
		logger.Errorf("visitor ack parse failed sid=%s err=%v", sessionID, err)
		return
	}

	msgID := strings.TrimSpace(payload.MsgID)
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("visitor ack session service unavailable sid=%s msg_id=%s", sessionID, msgID)
		return
	}

	// 获取会话对象
	session, err := ss.GetSession(sessionID)
	if err != nil || session == nil {
		logger.Errorf("visitor ack session not found sid=%s msg_id=%s err=%v", sessionID, msgID, err)
		return
	}

	// 更新会话的最后确认消息ID
	session.MarkVisitorDelivered(msgID)
	if err := ss.SaveSession(session); err != nil {
		logger.Errorf("visitor ack save session failed sid=%s msg_id=%s err=%v", sessionID, msgID, err)
	}
}

// handleMessage 处理访客侧消息并转发给已分配客服
// 这是消息处理的核心函数，访客发送的所有类型消息都通过此函数处理
// sessionID: 会话唯一标识
// msgType: 消息类型
// payloadBytes: 消息载荷数据
func (vc *VisitorController) handleMessage(sessionID, msgType string, payloadBytes []byte) {
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("visitor session service unavailable sid=%s", sessionID)
		return
	}

	// 获取会话对象
	session, err := ss.GetSession(sessionID)
	if err != nil || session == nil {
		logger.Errorf("visitor session not found sid=%s err=%v", sessionID, err)
		return
	}

	now := time.Now().Unix()

	// 商业化行为：访客在已关闭会话中继续发送消息时，自动重开当前会话并进入待接待队列
	// 这样设计有两个好处：
	// 1. 避免用户消息被拒绝，提供更好的体验
	// 2. 避免刷新页面时不断创建新会话（因为使用相同的sessionID）
	if session.Closed {
		logger.Infof("visitor message reopen closed session sid=%s", sessionID)
		session.ReopenByVisitor(now)
		if err := ss.SaveSession(session); err != nil {
			logger.Errorf("visitor reopen closed session save failed sid=%s err=%v", sessionID, err)
		}
		invalidateSessionListCache()
	}

	// 处理typing状态消息
	if msgType == MessageTypeTyping {
		// 只有已分配客服且会话未关闭时才推送typing状态给客服
		if session.CurAgentID != "" && !session.Closed {
			PushTypingToAgent(session.CurAgentID, session.SID)
		}
		// typing只表示输入态，不应计入未读消息数，只更新最后活跃时间
		session.LastActiveAt = now
		_ = ss.SaveSession(session)
		return
	}

	// 只有真实消息才计入访客未读与会话状态推进
	session.OnVisitorMessage(now, "")

	ms := service.GetMsgService()
	if ms == nil {
		logger.Errorf("visitor message service unavailable sid=%s", sessionID)
		return
	}

	// 解析消息载荷
	payload, err := models.ParseWSMessagePayload(payloadBytes)
	if err != nil {
		logger.Errorf("visitor payload parse failed sid=%s err=%v", sessionID, err)
		return
	}

	// 重置消息ID和时间戳，由服务端重新生成
	payload.MsgID = ""
	payload.Timestamp = now

	// 内容安全过滤
	payload.Content = filterSensitiveText(payload.Content)

	// 构建消息元数据
	payloadMap := map[string]interface{}{
		"from":         "visitor",
		"from_name":    session.VisitorID(),
		"sender_name":  session.VisitorID(),
		"content_type": payload.ContentType,
		"content":      payload.Content,
		"url":          payload.URL,
		"name":         payload.Name,
		"size":         payload.Size,
		"duration":     payload.Duration,
		"client_id":    payload.ClientID,
		"timestamp":    payload.Timestamp,
	}

	// 如果有引用回复，添加到元数据
	if payload.ReplyTo != nil {
		payloadMap["reply_to"] = payload.ReplyTo
	}
	metaBytes, _ := json.Marshal(payloadMap)

	// 构建消息对象
	msg := models.Message{
		Content:   payload.Content,
		MsgType:   payload.ContentType,
		Meta:      string(metaBytes),
		Timestamp: now,
	}

	// 保存消息到存储
	msgID, err := ms.SaveMessage(session.VisitorID(), session.AppID(), session.SessionSeq(), &msg)
	if err != nil {
		logger.Errorf("visitor message save failed sid=%s err=%v", sessionID, err)
		return
	}
	msg.MsgID = msgID
	session.MarkVisitorMessage(msg.MsgID)

	// 更新会话的最后消息信息
	session.TouchMessage(payload.ContentType, payload.Content, now)

	_ = ss.SaveSession(session)
	invalidateSessionListCache()

	// 获取系统配置检查自动分配策略
	cfg := getSystemSettingsCached()

	// 自动分配客服：如果会话还没有分配客服且开启了自动分配
	if session.CurAgentID == "" {
		if cfg.AutoAssign {
			us := service.GetUserService()
			if agent, _ := us.FindAgent(session.AppID()); agent != nil {
				// 分配客服
				session.AssignAgent(agent.Username, now)
				_ = ss.SaveSession(session)
				invalidateSessionListCache()

				// 通知访客已被分配
				PushMessageToVisitor(session.VisitorID(), session.SID, &models.Message{
					MsgType:   MessageTypeSystem,
					Content:   fmt.Sprintf("%s 为您服务", agent.Username),
					Timestamp: now,
				})
			}
		}
	}

	shouldTriggerAIBot := vc.shouldTriggerAIBotReply(session, payload, cfg)

	// 推送消息给客服或广播新会话通知
	if session.CurAgentID != "" {
		// 已有分配客服，直接推送消息给客服
		PushMessageToAgent(session.AgentID(), session.SID, &msg)
		// 统一通知触发条件：客服不在线或不在席时，触发邮件+企微通知
		notifyAgentUnavailableChannels(session, &msg)
	} else {
		// 未分配会话固定广播给所有在线客服（不再受“新会话通知”开关影响）
		PushMessageToAllOnlineAgents(session.SID, session.AppID(), &msg)
		// 仅在未启用 AI 机器人自动应答时，提示客服繁忙。
		if !shouldTriggerAIBot {
			PushMessageToVisitor(session.VisitorID(), session.SID, &models.Message{
				MsgType:   MessageTypeSystem,
				Content:   "当前客服繁忙，请稍等",
				Timestamp: now,
			})
		}
	}

	// SDK 访客自动问答：访客发送文本后自动触发机器人回复。
	if shouldTriggerAIBot {
		go vc.dispatchAIBotReply(session.SID, msg.MsgID, payload.Content, now, cfg)
	}
}

func (vc *VisitorController) shouldTriggerAIBotReply(session *models.Session, payload *models.WSMessagePayload, cfg systemSettings) bool {
	if session == nil || payload == nil {
		return false
	}
	if !cfg.AIBotEnabled {
		return false
	}
	if session.Closed {
		return false
	}
	if !cfg.AIBotWhenAssigned && strings.TrimSpace(session.CurAgentID) != "" {
		return false
	}
	if strings.TrimSpace(payload.ContentType) != models.WSContentTypeText {
		return false
	}
	return strings.TrimSpace(payload.Content) != ""
}

func beginAIBotReplyInflight(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	aiBotReplyInflightMu.Lock()
	defer aiBotReplyInflightMu.Unlock()
	if _, exists := aiBotReplyInflight[key]; exists {
		return false
	}
	aiBotReplyInflight[key] = struct{}{}
	return true
}

func endAIBotReplyInflight(key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	aiBotReplyInflightMu.Lock()
	delete(aiBotReplyInflight, key)
	aiBotReplyInflightMu.Unlock()
}

func buildAIBotReplyInflightKey(sessionID, triggerMsgID, question string, triggerTs int64) string {
	sessionID = strings.TrimSpace(sessionID)
	triggerMsgID = strings.TrimSpace(triggerMsgID)
	question = strings.TrimSpace(question)
	if triggerMsgID != "" {
		return sessionID + "|" + triggerMsgID
	}
	return sessionID + "|" + strconv.FormatInt(triggerTs, 10) + "|" + question
}

func (vc *VisitorController) canContinueAIBotReply(session *models.Session, triggerMsgID string, triggerTs int64, cfg systemSettings) bool {
	if session == nil || session.Closed {
		return false
	}
	triggerMsgID = strings.TrimSpace(triggerMsgID)
	if !cfg.AIBotWhenAssigned && strings.TrimSpace(session.CurAgentID) != "" {
		return false
	}
	// 会话里已经出现更新的访客消息，当前这次生成已过时，避免旧答案覆盖新问题。
	if triggerMsgID != "" && session.LastVisitorMsgID != "" && session.LastVisitorMsgID > triggerMsgID {
		return false
	}
	if triggerMsgID == "" && triggerTs > 0 && session.LastVisitorMsgTime > triggerTs {
		return false
	}
	// 若客服或机器人已经对这条消息之后做过回复，也不再发送陈旧答案。
	if triggerMsgID != "" && session.LastAgentReplyMsgID != "" && session.LastAgentReplyMsgID > triggerMsgID {
		return false
	}
	if triggerMsgID == "" && triggerTs > 0 && session.LastAgentReplyTime > triggerTs {
		return false
	}
	return true
}

func (vc *VisitorController) dispatchAIBotReply(sessionID, triggerMsgID, question string, triggerTs int64, cfg systemSettings) {
	sessionID = strings.TrimSpace(sessionID)
	triggerMsgID = strings.TrimSpace(triggerMsgID)
	question = strings.TrimSpace(question)
	if sessionID == "" || question == "" {
		return
	}
	inflightKey := buildAIBotReplyInflightKey(sessionID, triggerMsgID, question, triggerTs)
	if !beginAIBotReplyInflight(inflightKey) {
		logger.Warnf("ai bot dispatch skipped duplicated inflight sid=%s trigger_msg_id=%s", sessionID, triggerMsgID)
		return
	}
	defer endAIBotReplyInflight(inflightKey)

	ss := service.GetSessionService()
	ms := service.GetMsgService()
	if ss == nil || ms == nil {
		logger.Errorf("ai bot dispatch service unavailable sid=%s", sessionID)
		return
	}

	session, err := ss.GetSession(sessionID)
	if err != nil || session == nil {
		logger.Errorf("ai bot dispatch session not found sid=%s err=%v", sessionID, err)
		return
	}
	if !vc.canContinueAIBotReply(session, triggerMsgID, triggerTs, cfg) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// 先推送 typing，提升交互反馈。
	PushTypingToVisitor(session.VisitorID(), session.SID)
	streamKey := fmt.Sprintf("ai:%s:%d", session.SID, time.Now().UnixNano())
	answer := vc.generateAIBotAnswer(ctx, session, question, cfg, streamKey, func() bool {
		latest, latestErr := ss.GetSession(sessionID)
		if latestErr != nil || latest == nil {
			cancel()
			return false
		}
		if !vc.canContinueAIBotReply(latest, triggerMsgID, triggerTs, cfg) {
			cancel()
			return false
		}
		session = latest
		return true
	})
	if ctx.Err() != nil && strings.TrimSpace(answer) == "" {
		return
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		answer = "已收到您的问题，我先帮您记录并转交人工客服继续跟进。"
	}

	latest, err := ss.GetSession(sessionID)
	if err != nil || latest == nil {
		logger.Errorf("ai bot reload session failed sid=%s err=%v", sessionID, err)
		return
	}
	if !vc.canContinueAIBotReply(latest, triggerMsgID, triggerTs, cfg) {
		return
	}
	session = latest

	now := time.Now().Unix()
	botName := strings.TrimSpace(cfg.AIBotName)
	if botName == "" {
		botName = "AI机器人"
	}
	payloadMap := map[string]interface{}{
		"from":         "agent",
		"agent_name":   botName,
		"from_name":    botName,
		"sender_name":  botName,
		"content_type": models.WSContentTypeText,
		"content":      answer,
		"stream":       true,
		"stream_final": true,
		"stream_key":   streamKey,
		"timestamp":    now,
	}
	metaBytes, _ := json.Marshal(payloadMap)
	msg := models.Message{
		Content:   answer,
		MsgType:   models.WSContentTypeText,
		Meta:      string(metaBytes),
		Timestamp: now,
	}
	msgID, err := ms.SaveMessage(session.VisitorID(), session.AppID(), session.SessionSeq(), &msg)
	if err != nil {
		logger.Errorf("ai bot save message failed sid=%s err=%v", sessionID, err)
		return
	}
	msg.MsgID = msgID

	// 机器人已完成回复，清理未读并更新会话摘要。
	session.OnAgentReply(now, msg.MsgID)
	session.TouchMessage(models.WSContentTypeText, answer, now)
	if err := ss.SaveSession(session); err != nil {
		logger.Errorf("ai bot save session failed sid=%s err=%v", sessionID, err)
	}
	invalidateSessionListCache()

	_ = PushMessageToVisitor(session.VisitorID(), session.SID, &msg)
	if strings.TrimSpace(session.CurAgentID) != "" {
		PushMessageToAgent(session.CurAgentID, session.SID, &msg)
	}
}

func (vc *VisitorController) pushAIBotStreamDelta(session *models.Session, botName string, streamKey string, delta string, finished bool) {
	if session == nil {
		return
	}
	delta = strings.TrimSpace(delta)
	if delta == "" {
		return
	}
	now := time.Now().Unix()
	payloadMap := map[string]interface{}{
		"from":         "agent",
		"agent_name":   botName,
		"from_name":    botName,
		"sender_name":  botName,
		"content_type": models.WSContentTypeText,
		"content":      delta,
		"stream":       true,
		"stream_delta": true,
		"stream_final": finished,
		"stream_key":   streamKey,
		"timestamp":    now,
	}
	metaBytes, _ := json.Marshal(payloadMap)
	msg := models.Message{
		Content:   delta,
		MsgType:   models.WSContentTypeText,
		Meta:      string(metaBytes),
		Timestamp: now,
	}
	_ = PushMessageToVisitor(session.VisitorID(), session.SID, &msg)
	if strings.TrimSpace(session.CurAgentID) != "" {
		PushMessageToAgent(session.CurAgentID, session.SID, &msg)
	}
}

func (vc *VisitorController) generateAIBotAnswer(ctx context.Context, session *models.Session, question string, cfg systemSettings, streamKey string, canStream func() bool) string {
	if session == nil {
		return ""
	}
	appID := strings.TrimSpace(session.AppID())
	question = strings.TrimSpace(question)
	if appID == "" || question == "" {
		return ""
	}

	topK := cfg.AIBotTopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	systemPrompt := strings.TrimSpace(cfg.AIBotPrompt)
	if systemPrompt == "" {
		systemPrompt = "你是一名客服机器人，回答要准确、简洁、礼貌。证据不足时明确说明并建议转人工。"
	}

	botName := strings.TrimSpace(cfg.AIBotName)
	if botName == "" {
		botName = "AI机器人"
	}
	lastTypingPush := time.Now()

	hits, err := searchKnowledgeHitsByApp(ctx, appID, question, topK)
	if err == nil && len(hits) > 0 {
		modelOverride := strings.TrimSpace(cfg.AIBotModel)
		if modelOverride != "" {
			if modelCfg, cfgErr := service.GetEnabledAPIModelConfig(); cfgErr == nil {
				modelCfg.ModelName = modelOverride
				if out, inferErr := service.AnswerWithAPIModelStreamWithSystemPrompt(ctx, modelCfg, question, hits, systemPrompt, func(current string) {
					if strings.TrimSpace(streamKey) == "" {
						return
					}
					if canStream != nil && !canStream() {
						return
					}
					vc.pushAIBotStreamDelta(session, botName, streamKey, current, false)
					if time.Since(lastTypingPush) > 1500*time.Millisecond {
						PushTypingToVisitor(session.VisitorID(), session.SID)
						lastTypingPush = time.Now()
					}
				}); inferErr == nil && out != nil && strings.TrimSpace(out.Answer) != "" {
					return strings.TrimSpace(out.Answer)
				}
			}
		}
		if out, _, inferErr := service.AnswerWithEnabledAPIModelStreamWithSystemPrompt(ctx, question, hits, systemPrompt, func(current string) {
			if strings.TrimSpace(streamKey) == "" {
				return
			}
			if canStream != nil && !canStream() {
				return
			}
			vc.pushAIBotStreamDelta(session, botName, streamKey, current, false)
			if time.Since(lastTypingPush) > 1500*time.Millisecond {
				PushTypingToVisitor(session.VisitorID(), session.SID)
				lastTypingPush = time.Now()
			}
		}); inferErr == nil && out != nil && strings.TrimSpace(out.Answer) != "" {
			return strings.TrimSpace(out.Answer)
		}
		return composeAISuggestion(cfg.AIBotStyle, "", question, vectorHitsToRAGChunks(hits))
	}
	return composeAISuggestion(cfg.AIBotStyle, "", question, nil)
}

// detectDevice 基于User-Agent进行简易设备类型识别
// userAgent: HTTP请求头中的User-Agent字符串
// 返回值：设备类型枚举值（mobile/tablet/desktop/unknown）
func detectDevice(userAgent string) string {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return "unknown"
	}
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	if strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") {
		return "tablet"
	}
	return "desktop"
}

// pushOfflineMessages 在访客重连后补推离线期间未送达的客服/系统消息
// 当访客重新建立WebSocket连接时，调用此函数将离线期间积累的消息推送给访客
// 判断依据：消息ID大于LastVisitorAckMsgID的消息表示未确认送达
// session: 会话对象
func (vc *VisitorController) pushOfflineMessages(session *models.Session) {
	if session == nil || session.SID == "" {
		logger.Debugf("push offline messages skipped: session is nil or empty sid")
		return
	}
	ms := service.GetMsgService()
	if ms == nil {
		logger.Errorf("push offline messages failed: message service unavailable sid=%s", session.SID)
		return
	}

	// 获取会话的所有消息（最多100条）
	msgs, err := ms.GetMessagesBySession(session.SID, 100)
	if err != nil || len(msgs) == 0 {
		logger.Debugf("push offline messages skipped: no messages sid=%s err=%v", session.SID, err)
		return
	}

	// 获取访客最后确认的消息ID
	ack := strings.TrimSpace(session.LastVisitorAckMsgID)
	for _, msg := range msgs {
		if msg == nil || msg.MsgID == "" {
			continue
		}
		// 跳过已确认送达的消息
		if ack != "" && msg.MsgID <= ack {
			continue
		}
		// 跳过typing状态消息
		if msg.MsgType == models.WSMessageTypeTyping {
			continue
		}
		// 跳过访客自己发送的消息（不需要补推自己发的消息）
		bizType := deduceMessageBusinessType(msg)
		if bizType == models.WSMessageTypeVisitor {
			continue
		}
		// 推送消息给访客
		_ = PushMessageToVisitor(session.VisitorID(), session.SID, msg)
	}
}

// isValidOrigin 校验访客WebSocket请求来源域名是否在应用白名单内
// 这是安全校验的重要环节，防止恶意站点伪造客服系统请求
// appID: 应用唯一标识
// origin: Origin请求头值
// referer: Referer请求头值
// 返回值：域名在白名单内返回true，否则返回false
func (vc *VisitorController) isValidOrigin(appID, origin, referer string) bool {
	logger.Infof("visitor websocket origin check app_id=%s origin=%s referer=%s", appID, origin, referer)

	// 获取应用配置
	app := models.GetApp(appID)
	if app == nil {
		logger.Errorf("visitor websocket app not found app_id=%s", appID)
		return false
	}

	// 调用模型层进行域名匹配校验
	result := models.IsDomainAllowed(origin, referer, app.AllowDomain)
	logger.Infof("visitor websocket domain check result=%t allow_domain=%s", result, app.AllowDomain)

	return result
}

// buildVisitorOriginPatterns 生成websocket.Accept需要的OriginPatterns
// websocket库在握手阶段需要验证Origin头，这里生成匹配模式列表
// 说明：
// 1. 先由业务白名单完成域名授权（isValidOrigin）
// 2. 这里仅用于websocket库握手阶段匹配Origin（包含端口场景）
// origin: Origin请求头值
// referer: Referer请求头值
// 返回值：Origin匹配模式列表
func buildVisitorOriginPatterns(origin, referer string) []string {
	candidates := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)

	// 添加候选值到列表（去重）
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		candidates = append(candidates, v)
	}

	// 解析URL并生成多种匹配模式
	parse := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		u, err := url.Parse(raw)
		if err != nil || strings.TrimSpace(u.Host) == "" {
			logger.Debugf("build origin patterns parse failed: raw=%s err=%v", raw, err)
			return
		}
		host := strings.TrimSpace(u.Host)           // 含端口，例如 127.0.0.1:5500
		hostname := strings.TrimSpace(u.Hostname()) // 不含端口，例如 127.0.0.1
		scheme := strings.TrimSpace(u.Scheme)

		// 添加各种匹配模式
		add(host)
		add(hostname)
		if hostname != "" {
			add(hostname + ":*") // 支持任意端口
		}
		if scheme != "" {
			add(scheme + "://" + host)
			if hostname != "" {
				add(scheme + "://" + hostname + ":*")
			}
		}
	}

	// 优先解析origin
	parse(origin)

	// 如果origin解析失败或为空，尝试解析referer
	if len(candidates) == 0 {
		parse(referer)
	}

	return candidates
}
