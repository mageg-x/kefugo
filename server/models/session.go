package models

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	SessionStatusUnAssigned = "unassigned" // 未分配客服
	SessionStatusUnRead     = "unread"     // 未读新消息
	SessionStatusUnReply    = "unreply"    // 有未回复
	SessionStatusAssigned   = "assigned"   // 已分配客服
	SessionStatusFollowUP   = "follow"     // 待跟进
	SessionStatusClosed     = "closed"     // 会话结束
)

type Session struct {
	SID                 string `json:"sid"`                    // s:{visitor_id}:{app_id}:{session_seq}
	CurAgentID          string `json:"cur_agent_id,omitempty"` // 当前负责客服
	CreatedAt           int64  `json:"created_at"`             // 创建时间
	LastClientIP        string `json:"last_client_ip,omitempty"`
	LastUserAgent       string `json:"last_user_agent,omitempty"`
	LastDevice          string `json:"last_device,omitempty"`
	LastGeo             string `json:"last_geo,omitempty"`
	LastVisitorMsgTime  int64  `json:"last_visitor_msg_time"`             // 最后访客消息时间
	LastVisitorMsgID    string `json:"last_visitor_msg_id,omitempty"`     // 最后访客消息ID
	LastAgentReplyTime  int64  `json:"last_agent_reply_time"`             // 最后客服回复消息时间
	LastAgentReplyMsgID string `json:"last_agent_reply_msg_id,omitempty"` // 最后客服/机器人回复消息ID
	LastAgentReadTime   int64  `json:"last_agent_read_time"`              // 最后客服已读消息时间
	LastMessage         string `json:"last_message,omitempty"`            // 最近一条消息摘要
	LastMessageType     string `json:"last_message_type,omitempty"`       // 最近一条消息类型
	LastVisitorAckMsgID string `json:"last_visitor_ack_msg_id,omitempty"` // 最后成功推送给访客的消息ID
	UnreadCount         int    `json:"unread_count"`                      // 未读消息数
	LastActiveAt        int64  `json:"last_active_at"`                    // 最近活跃时间（访客或客服）
	Closed              bool   `json:"closed"`                            // 会话是否关闭
	FollowUp            bool   `json:"need_follow_up"`                    // 会话是否需要跟进
	RatingScore         int    `json:"rating_score"`                      // 1~5，0表示未评分
	RatingComment       string `json:"rating_comment,omitempty"`
	RatedAt             int64  `json:"rated_at"`
}

// ParseSessionID 从 session_id 解析 visitor_id、app_id 与 session_seq。
func ParseSessionID(sessionID string) (visitorID, appID string, sessionSeq uint32) {
	s := Session{SID: sessionID}
	return s.ParseSid()
}

// GetSessionID 按统一规则构建会话 ID：s:{visitor}:{app}:{seq}。
func GetSessionID(visitorID, appID string, sessionSeq uint32) string {
	return fmt.Sprintf("s:%s:%s:%010d", visitorID, appID, sessionSeq)
}

// 1. 访客发消息
func (s *Session) OnVisitorMessage(ts int64, msgID string) {
	if s.Closed {
		return
	}
	s.LastVisitorMsgTime = ts
	s.MarkVisitorMessage(msgID)
	s.LastActiveAt = ts
	s.UnreadCount++
}

// 2. 客服发送普通回复
func (s *Session) OnAgentReply(ts int64, msgID string) {
	if s.Closed {
		return
	}
	s.LastAgentReplyTime = ts
	s.MarkAgentReplyMessage(msgID)
	s.LastAgentReadTime = ts // 回复即视为已读
	s.LastActiveAt = ts
	s.UnreadCount = 0
}

// 3. 客服接单
func (s *Session) AssignAgent(agentID string, ts int64) {
	if s.Closed {
		return
	}
	s.CurAgentID = agentID
	// 注意：不改 Read/Reply 时间！
}

// 4. 客服点开会话（触发 badge 清除）
func (s *Session) MarkRead(ts int64) {
	if s.Closed {
		return
	}
	if s.CurAgentID != "" {
		s.LastAgentReadTime = ts
		s.UnreadCount = 0
	}
}

// 5. 客服标记“需后续跟进”
func (s *Session) MarkFollowUp() {
	if s.Closed {
		return
	}
	s.FollowUp = true
	// 关键：清空已读时间，确保 badge 出现
	s.LastAgentReadTime = 0
}

// 6. 关闭会话
func (s *Session) Close() {
	s.Closed = true
	s.FollowUp = false
}

// ReopenByVisitor 访客在“已结束会话”中再次发送消息时，按商业客服常见行为重开会话。
// 设计说明：
// 1) 不生成新 SID，避免刷新页面/重连时产生大量空会话；
// 2) 清空原负责客服，让会话重新进入可接待流程；
// 3) 重置已读/回复时间，确保客服侧能看到未读提醒与待处理状态。
func (s *Session) ReopenByVisitor(ts int64) {
	s.Closed = false
	s.FollowUp = false
	s.CurAgentID = ""
	s.LastAgentReplyTime = 0
	s.LastAgentReplyMsgID = ""
	s.LastAgentReadTime = 0
	if ts > 0 {
		s.LastActiveAt = ts
	}
}

func (s *Session) TouchMessage(msgType, content string, ts int64) {
	s.LastMessageType = strings.TrimSpace(msgType)
	s.LastMessage = strings.TrimSpace(content)
	if ts > 0 {
		s.LastActiveAt = ts
	}
}

func (s *Session) TouchVisitorProfile(ip, userAgent, device, geo string) {
	if strings.TrimSpace(ip) != "" {
		s.LastClientIP = strings.TrimSpace(ip)
	}
	if strings.TrimSpace(userAgent) != "" {
		s.LastUserAgent = strings.TrimSpace(userAgent)
	}
	if strings.TrimSpace(device) != "" {
		s.LastDevice = strings.TrimSpace(device)
	}
	if strings.TrimSpace(geo) != "" {
		s.LastGeo = strings.TrimSpace(geo)
	}
}

func (s *Session) MarkVisitorDelivered(msgID string) {
	msgID = strings.TrimSpace(msgID)
	if msgID == "" {
		return
	}
	if s.LastVisitorAckMsgID == "" || msgID > s.LastVisitorAckMsgID {
		s.LastVisitorAckMsgID = msgID
	}
}

func (s *Session) MarkVisitorMessage(msgID string) {
	msgID = strings.TrimSpace(msgID)
	if msgID == "" {
		return
	}
	if s.LastVisitorMsgID == "" || msgID > s.LastVisitorMsgID {
		s.LastVisitorMsgID = msgID
	}
}

func (s *Session) MarkAgentReplyMessage(msgID string) {
	msgID = strings.TrimSpace(msgID)
	if msgID == "" {
		return
	}
	if s.LastAgentReplyMsgID == "" || msgID > s.LastAgentReplyMsgID {
		s.LastAgentReplyMsgID = msgID
	}
}

func (s *Session) Rate(score int, comment string, ts int64) bool {
	if score < 1 || score > 5 {
		return false
	}
	s.RatingScore = score
	s.RatingComment = strings.TrimSpace(comment)
	s.RatedAt = ts
	return true
}

// 7. 获取会话状态
func (s *Session) Status() string {
	if s.Closed {
		return SessionStatusClosed
	}
	if s.CurAgentID == "" {
		return SessionStatusUnAssigned
	}
	if s.LastVisitorMsgTime > s.LastAgentReadTime {
		return SessionStatusUnRead
	}
	if s.LastVisitorMsgID != "" && s.LastAgentReplyMsgID != "" {
		if s.LastVisitorMsgID > s.LastAgentReplyMsgID {
			return SessionStatusUnReply
		}
	}
	if s.LastVisitorMsgTime > s.LastAgentReplyTime {
		return SessionStatusUnReply
	}
	if s.FollowUp {
		return SessionStatusFollowUP
	}
	return SessionStatusAssigned
}

// ParseSid 解析当前会话对象中的 SID 字段。
func (s *Session) ParseSid() (visitorID, appID string, sessionSeq uint32) {
	parts := strings.Split(s.SID, ":")
	if len(parts) != 4 {
		return "", "", 0
	}

	visitorID = parts[1]
	appID = parts[2]
	sessionSeq64, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return "", "", 0
	}
	sessionSeq = uint32(sessionSeq64)
	return
}

// 9. 获取 visitor_id
func (s *Session) VisitorID() string {
	visitor, _, _ := s.ParseSid()
	return visitor
}

// 10. 获取 app_id
func (s *Session) AppID() string {
	_, appID, _ := s.ParseSid()
	return appID
}

// 11. 获取 agent_id
func (s *Session) AgentID() string {
	return s.CurAgentID
}

// 12. 获取 session_seq
func (s *Session) SessionSeq() uint32 {
	_, _, sessionSeq := s.ParseSid()
	return sessionSeq
}
