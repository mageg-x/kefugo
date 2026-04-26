package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/utils"
	"kefu-server/utils/logger"
)

// 客服WebSocket消息类型常量定义
// 用于区分不同类型的WebSocket消息，便于在消息处理流程中进行路由分发
const (
	MessageTypeRsp         = models.WSMessageTypeAgent  // 客服发送的回复消息类型
	MessageTypeAgentTyping = models.WSMessageTypeTyping // 客服正在输入状态通知
)

// AgentConn 表示单个客服的WebSocket连接
// 每个客服连接包含：底层WebSocket连接、客服ID、发送通道（用于异步消息发送）、完成通道（用于连接生命周期管理）
// 注意：一个客服可以有多个连接（如多设备登录），所以使用map来管理
type AgentConn struct {
	Conn     *websocket.Conn // 底层的WebSocket连接对象
	AgentID  string          // 客服唯一标识（用户名）
	SendChan chan []byte     // 异步消息发送通道，writeLoop从此通道读取消息并发送给客服
	Done     chan struct{}   // 连接生命周期管理通道，连接断开时关闭
}

// 全局客服连接池：agent_id => 所有连接集合
// 使用agent_id作为key，可以快速查找某个客服的所有连接
// 一个客服可能有多个设备同时在线，所以用map[*AgentConn]struct{}表示一组连接
var (
	agentConns = struct {
		mu    sync.RWMutex
		conns map[string]map[*AgentConn]struct{}
	}{
		conns: make(map[string]map[*AgentConn]struct{}),
	}
)

// registerAgentConn 注册客服WebSocket连接到连接池
// agentID: 客服唯一标识
// conn: 客服连接对象
// 此函数在客服成功建立WebSocket连接后调用，将连接加入全局连接池以便后续消息推送
// 支持同一客服多设备同时在线
func registerAgentConn(agentID string, conn *AgentConn) {
	agentConns.mu.Lock()
	defer agentConns.mu.Unlock()
	if _, ok := agentConns.conns[agentID]; !ok {
		agentConns.conns[agentID] = make(map[*AgentConn]struct{})
	}
	agentConns.conns[agentID][conn] = struct{}{}
	logger.Infof("agent conn registered agent_id=%s conn_ptr=%p", agentID, conn)
}

// unregisterAgentConn 注销指定客服的所有WebSocket连接
// 通常在客服登出或被删除时调用，清除该客服的所有连接
func unregisterAgentConn(agentID string) {
	agentConns.mu.Lock()
	defer agentConns.mu.Unlock()
	connSet, ok := agentConns.conns[agentID]
	if !ok {
		logger.Debugf("agent conn unregister skipped: not found agent_id=%s", agentID)
		return
	}
	// 关闭所有连接并从映射表中删除
	for conn := range connSet {
		delete(connSet, conn)
	}
	delete(agentConns.conns, agentID)
	logger.Infof("agent conn unregistered all agent_id=%s", agentID)
}

// unregisterAgentConnInstance 注销客服的单个WebSocket连接实例
// 当客服在某个设备断开连接时调用，只删除该设备对应的连接，保留其他设备连接
// 这是支持多设备同时在线的关键实现
func unregisterAgentConnInstance(agentID string, target *AgentConn) {
	agentConns.mu.Lock()
	defer agentConns.mu.Unlock()
	connSet, ok := agentConns.conns[agentID]
	if !ok {
		logger.Debugf("agent conn instance unregister skipped: agent not found agent_id=%s", agentID)
		return
	}
	delete(connSet, target)
	// 如果该客服没有其他连接了，删除整个key
	if len(connSet) == 0 {
		delete(agentConns.conns, agentID)
		logger.Infof("agent conn instance removed, last connection agent_id=%s", agentID)
	}
}

// getAgentConnSnapshot 获取指定客服所有连接的快照
// 返回连接指针列表，用于向客服推送消息
// 返回值是一个切片，调用方可以安全遍历发送消息
func getAgentConnSnapshot(agentID string) []*AgentConn {
	agentConns.mu.RLock()
	defer agentConns.mu.RUnlock()
	connSet, ok := agentConns.conns[agentID]
	if !ok || len(connSet) == 0 {
		logger.Debugf("agent conn snapshot empty agent_id=%s", agentID)
		return nil
	}
	result := make([]*AgentConn, 0, len(connSet))
	for conn := range connSet {
		if conn != nil {
			result = append(result, conn)
		}
	}
	return result
}

func getAgentConnSnapshotByApp(appID string) []*AgentConn {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil
	}

	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("agent conn snapshot by app unavailable app_id=%s", appID)
		return nil
	}

	agentConns.mu.RLock()
	agentIDs := make([]string, 0, len(agentConns.conns))
	for agentID := range agentConns.conns {
		agentIDs = append(agentIDs, agentID)
	}
	agentConns.mu.RUnlock()

	allowed := make(map[string]struct{}, len(agentIDs))
	for _, agentID := range agentIDs {
		user, err := userService.GetUser(agentID)
		if err != nil || user == nil {
			continue
		}
		if isAgentForApp(user.Apps, appID) {
			allowed[agentID] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil
	}

	agentConns.mu.RLock()
	defer agentConns.mu.RUnlock()

	result := make([]*AgentConn, 0, len(agentConns.conns))
	for agentID, connSet := range agentConns.conns {
		if _, ok := allowed[agentID]; !ok {
			continue
		}
		for conn := range connSet {
			if conn != nil {
				result = append(result, conn)
			}
		}
	}
	return result
}

// PushMessageToAllOnlineAgents 向所有在线客服广播消息
// 用于未分配会话的新会话实时通知，让所有客服都能看到新会话提示
// 这是实现"有新会话了"功能的关键函数
func PushMessageToAllOnlineAgents(sessionID, appID string, msg *models.Message) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(appID) == "" || msg == nil {
		logger.Debugf("agent broadcast skip: empty session_id or msg")
		return
	}

	conns := getAgentConnSnapshotByApp(appID)
	if len(conns) == 0 {
		logger.Debugf("agent broadcast skip: no eligible online connection app_id=%s sid=%s", appID, sessionID)
		return
	}

	// 对sessionID进行Base58编码用于WebSocket传输
	sidEncoded := utils.Base58Encode([]byte(sessionID))
	// 构建WebSocket数据包，消息类型为visitor表示这是访客发来的消息
	packet := models.BuildOutgoingWSPacket(models.WSMessageTypeVisitor, msg, sidEncoded)
	payload, _ := json.Marshal(packet)

	// 遍历符合应用权限的在线客服连接
	for _, conn := range conns {
		if conn == nil {
			continue
		}
		select {
		case conn.SendChan <- payload:
			logger.Debugf("agent broadcast sent agent_id=%s app_id=%s sid=%s", conn.AgentID, appID, sessionID)
		default:
			logger.Warnf("agent broadcast send buffer full agent_id=%s app_id=%s sid=%s", conn.AgentID, appID, sessionID)
		}
	}
}

// PushMessageToAgent 向指定客服推送消息
// 客服发送消息给访客时调用此函数
// 如果客服多设备在线，消息会发送到所有设备
func PushMessageToAgent(agentID, sessionID string, msg *models.Message) {
	conns := getAgentConnSnapshot(agentID)
	if len(conns) == 0 {
		logger.Debugf("agent push skip: no online connection agent_id=%s sid=%s", agentID, sessionID)
		return
	}

	// 对sessionID进行Base58编码用于WebSocket传输
	sidEncoded := utils.Base58Encode([]byte(sessionID))
	// 构建WebSocket数据包，消息类型为visitor表示这是访客的消息
	packet := models.BuildOutgoingWSPacket(models.WSMessageTypeVisitor, msg, sidEncoded)
	payload, _ := json.Marshal(packet)

	// 向该客服的所有连接发送消息
	for _, conn := range conns {
		select {
		case conn.SendChan <- payload:
			logger.Debugf("agent msg sent agent_id=%s sid=%s msg_id=%s", agentID, sessionID, msg.MsgID)
		default:
			logger.Warnf("agent msg send buffer full agent_id=%s sid=%s", agentID, sessionID)
		}
	}
}

// PushTypingToAgent 向客服推送"访客正在输入"事件
// 当访客开始输入消息时，会调用此函数通知客服端显示输入提示
func PushTypingToAgent(agentID, sessionID string) {
	conns := getAgentConnSnapshot(agentID)
	if len(conns) == 0 {
		logger.Debugf("agent typing skip: no online connection agent_id=%s sid=%s", agentID, sessionID)
		return
	}

	// 构建typing通知的消息体
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"from":      "visitor",         // 标识消息来源为访客
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

	// 向该客服的所有连接发送typing通知
	for _, conn := range conns {
		select {
		case conn.SendChan <- body:
			logger.Debugf("agent typing sent agent_id=%s sid=%s", agentID, sessionID)
		default:
			logger.Warnf("agent typing send buffer full agent_id=%s sid=%s", agentID, sessionID)
		}
	}
}

// AgentController 客服端控制器
// 负责处理客服相关的HTTP请求和WebSocket连接
type AgentController struct{}

// WSHandler 处理客服WebSocket连接
// HTTP GET /ws/agent
// 这是客服端建立WebSocket长连接的入口函数
// 需要通过AuthMiddleware进行身份认证
// 流程：身份校验 -> 建立WebSocket连接 -> 注册连接 -> 启动读写循环
func (ac *AgentController) WSHandler(c *gin.Context) {
	// 从AuthMiddleware设置的上下文获取用户名
	userName, exists := c.Get("userName")
	if !exists || userName == "" {
		logger.Errorf("agent auth context missing")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	agentID := userName.(string)

	// 获取用户服务验证客服身份
	us := service.GetUserService()
	if us == nil {
		logger.Errorf("agent user service unavailable")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 获取用户信息验证身份
	agent, err := us.GetUser(agentID)
	if agent == nil || err != nil {
		logger.Errorf("agent user not found id=%s err=%v", agentID, err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 验证用户角色和状态：必须是agent或admin，且状态为激活
	if (agent.Role != "agent" && agent.Role != "admin") || !agent.Active {
		logger.Errorf("agent role or status invalid id=%s role=%s active=%t", agentID, agent.Role, agent.Active)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 接受WebSocket连接
	conn, err := websocket.Accept(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("agent websocket accept failed id=%s err=%v", agentID, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer conn.CloseNow()

	logger.Infof("agent websocket connected agent_id=%s role=%s", agentID, agent.Role)

	// 创建客服连接对象
	agentConn := &AgentConn{
		Conn:     conn,
		AgentID:  agentID,
		SendChan: make(chan []byte, 256), // 256容量缓冲，比访客端大因为客服可能收到更多消息
		Done:     make(chan struct{}),
	}

	// 将连接注册到全局连接池
	registerAgentConn(agentID, agentConn)
	// 断开时只注销这个连接实例，保留其他设备连接
	defer unregisterAgentConnInstance(agentID, agentConn)

	// 创建上下文用于管理读写协程的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动读写协程
	// readLoop: 读取客服发送的消息
	// writeLoop: 向客服发送消息（从SendChan读取）并发送ping保活
	go ac.readLoop(ctx, agentConn)
	go ac.writeLoop(ctx, agentConn)

	// 等待连接断开信号
	<-agentConn.Done
	logger.Infof("agent websocket disconnected agent_id=%s", agentID)
}

// readLoop 负责读取客服发来的WebSocket消息
// 这是一个阻塞函数，会持续从WebSocket连接读取消息直到连接断开或发生错误
func (ac *AgentController) readLoop(ctx context.Context, conn *AgentConn) {
	defer close(conn.Done)

	for {
		// 设置120秒读超时，防止长时间无数据导致的连接僵死
		readCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		_, data, err := conn.Conn.Read(readCtx)
		cancel()

		if err != nil {
			// 检查是否为正常断开
			if websocket.CloseStatus(err) == -1 {
				// -1表示非正常关闭（如网络中断、进程崩溃等）
				logger.Errorf("agent websocket read failed agent_id=%s err=%v", conn.AgentID, err)
			} else {
				// 正常关闭，连接断开是预期行为
				logger.Infof("agent websocket read done agent_id=%s status=%v", conn.AgentID, websocket.CloseStatus(err))
			}
			return
		}

		// 解析WebSocket消息包
		var req models.WSPacket
		if err := json.Unmarshal(data, &req); err != nil {
			logger.Debugf("agent websocket json parse failed agent_id=%s err=%v", conn.AgentID, err)
			continue
		}

		// 解码sessionID（WebSocket传输时使用Base58编码）
		sessionIDBytes := utils.Base58Decode(req.SID)
		if len(sessionIDBytes) == 0 {
			logger.Debugf("agent websocket sid decode failed agent_id=%s sid=%s", conn.AgentID, req.SID)
			continue
		}
		sessionID := string(sessionIDBytes)

		// 处理消息
		ac.handleMessage(conn.AgentID, sessionID, req.Type, req.Payload)
	}
}

// writeLoop 负责把待发送消息写回客服WebSocket连接，并定时发送ping保活
// 这个函数从SendChan通道读取消息并发送给客服，同时每25秒发送一次ping探测连接状态
func (ac *AgentController) writeLoop(ctx context.Context, conn *AgentConn) {
	// 每25秒发送一次ping保活
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文被取消，退出循环
			logger.Debugf("agent write loop context cancelled agent_id=%s", conn.AgentID)
			return
		case <-conn.Done:
			// 连接已断开，退出循环
			logger.Debugf("agent write loop connection done agent_id=%s", conn.AgentID)
			return
		case msg := <-conn.SendChan:
			// 有消息需要发送
			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				logger.Errorf("agent websocket write failed agent_id=%s err=%v", conn.AgentID, err)
				return
			}
		case <-ticker.C:
			// 定时ping探测
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := conn.Conn.Ping(pingCtx); err != nil {
				cancel()
				logger.Errorf("agent websocket ping failed agent_id=%s err=%v", conn.AgentID, err)
				return
			}
			cancel()
		}
	}
}

// handleMessage 处理客服侧WebSocket消息
// 这是客服消息处理的核心函数，客服发送的所有类型消息都通过此函数处理
// agentID: 客服唯一标识
// sessionID: 会话唯一标识
// actionType: 消息类型（回复/typing/关闭）
// payloadBytes: 消息载荷数据
func (ac *AgentController) handleMessage(agentID, sessionID, actionType string, payloadBytes []byte) {
	ss := service.GetSessionService()
	if ss == nil {
		logger.Errorf("agent session service unavailable agent_id=%s sid=%s", agentID, sessionID)
		return
	}

	// 获取会话对象
	session, err := ss.GetSession(sessionID)
	if err != nil || session == nil {
		logger.Errorf("agent session not found agent_id=%s sid=%s err=%v", agentID, sessionID, err)
		return
	}

	// 安全校验：确保只有当前负责这个会话的客服才能操作
	if session.CurAgentID != agentID {
		logger.Errorf("agent session owner mismatch agent_id=%s sid=%s owner=%s", agentID, sessionID, session.CurAgentID)
		return
	}

	now := time.Now().Unix()

	// 根据消息类型进行不同处理
	switch actionType {
	case MessageTypeRsp:
		// 客服发送回复消息
		// 首先检查会话是否已关闭，已关闭则拒绝发送
		if session.Closed || session.Status() == models.SessionStatusClosed {
			logger.Warnf("agent send blocked: session closed agent_id=%s sid=%s", agentID, sessionID)
			// 向客服发送系统消息告知发送失败
			blockPayload := map[string]interface{}{
				"from":         "system",
				"content_type": models.WSContentTypeText,
				"content":      "会话已结束，消息未发送",
				"code":         "session_closed_blocked",
				"timestamp":    now,
			}
			blockBytes, _ := json.Marshal(blockPayload)
			blockMsg := models.Message{
				MsgType:   models.WSMessageTypeSystem,
				Content:   "会话已结束，消息未发送",
				Meta:      string(blockBytes),
				Timestamp: now,
			}
			PushMessageToAgent(agentID, sessionID, &blockMsg)
			return
		}

		// 获取消息服务
		ms := service.GetMsgService()
		if ms == nil {
			logger.Errorf("agent message service unavailable agent_id=%s sid=%s", agentID, sessionID)
			return
		}

		// 解析消息载荷
		payload, err := models.ParseWSMessagePayload(payloadBytes)
		if err != nil {
			logger.Errorf("agent payload parse failed agent_id=%s sid=%s err=%v", agentID, sessionID, err)
			return
		}

		// 重置消息ID和时间戳，由服务端重新生成
		payload.MsgID = ""
		payload.Timestamp = now

		// 内容安全过滤
		payload.Content = filterSensitiveText(payload.Content)

		// 构建消息元数据
		payloadMap := map[string]interface{}{
			"from":         "agent",
			"agent_name":   agentID,
			"from_name":    agentID,
			"sender_name":  agentID,
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
			logger.Errorf("agent message save failed agent_id=%s sid=%s err=%v", agentID, sessionID, err)
			return
		}
		msg.MsgID = msgID

		// 更新会话状态：回复消息视为已读，未读计数清零
		session.OnAgentReply(now, msg.MsgID)
		session.TouchMessage(payload.ContentType, payload.Content, now)
		_ = ss.SaveSession(session)
		invalidateSessionListCache()

		// 推送消息给访客
		PushMessageToVisitor(session.VisitorID(), sessionID, &msg)

	case MessageTypeAgentTyping:
		// 客服正在输入，推送给访客
		PushTypingToVisitor(session.VisitorID(), sessionID)

	case MessageTypeClose:
		// 客服关闭会话
		session.Close()
		_ = ss.SaveSession(session)
		invalidateSessionListCache()

		// 保存系统消息通知会话结束
		systemMsg := saveSystemMessageForSession(session, "会话已结束", now)

		// 推送会话结束消息给访客和客服
		PushMessageToVisitor(session.VisitorID(), sessionID, systemMsg)
		PushMessageToAgent(agentID, sessionID, systemMsg)

	default:
		logger.Debugf("agent action unhandled type=%s agent_id=%s sid=%s", actionType, agentID, sessionID)
	}
}
