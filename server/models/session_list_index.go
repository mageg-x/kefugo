package models

import "time"

// SessionListIndex 是会话列表查询的 SQLite 索引快照。
// 目标：让会话列表支持高效过滤、排序和分页，避免每次全量扫描 Badger。
type SessionListIndex struct {
	SID                string `gorm:"primaryKey;size:128" json:"sid"`
	VisitorID          string `gorm:"index;size:128" json:"visitor_id"`
	AppID              string `gorm:"size:128;index;index:idx_sli_app_status_active_created,priority:1;index:idx_sli_app_agent_active_created,priority:1" json:"app_id"`
	Status             string `gorm:"size:32;index;index:idx_sli_app_status_active_created,priority:2" json:"status"`
	CurAgentID         string `gorm:"size:64;index;index:idx_sli_agent_active_created,priority:1;index:idx_sli_app_agent_active_created,priority:2" json:"cur_agent_id"`
	LastClientIP       string `gorm:"size:64" json:"last_client_ip"`
	LastUserAgent      string `gorm:"size:512" json:"last_user_agent"`
	LastDevice         string `gorm:"size:64" json:"last_device"`
	LastGeo            string `gorm:"size:64" json:"last_geo"`
	RatingScore        int    `json:"rating_score"`
	RatingComment      string `gorm:"size:255" json:"rating_comment"`
	RatedAt            int64  `gorm:"index" json:"rated_at"`
	LastVisitorMsgTime int64  `gorm:"index" json:"last_visitor_msg_time"`
	LastAgentReplyTime int64  `gorm:"index" json:"last_agent_reply_time"`
	LastAgentReadTime  int64  `json:"last_agent_read_time"`
	CreatedAt          int64  `gorm:"index;index:idx_sli_active_created,priority:2,sort:desc;index:idx_sli_app_status_active_created,priority:4,sort:desc;index:idx_sli_agent_active_created,priority:3,sort:desc;index:idx_sli_app_agent_active_created,priority:4,sort:desc" json:"created_at"`
	LastActiveAt       int64  `gorm:"index;index:idx_sli_active_created,priority:1,sort:desc;index:idx_sli_app_status_active_created,priority:3,sort:desc;index:idx_sli_agent_active_created,priority:2,sort:desc;index:idx_sli_app_agent_active_created,priority:3,sort:desc" json:"last_active_at"`
	UnreadCount        int    `json:"unread_count"`
	LastMessage        string `gorm:"size:255" json:"last_message"`
	LastMessageType    string `gorm:"size:32" json:"last_message_type"`
	UpdatedAt          int64  `gorm:"index" json:"updated_at"`
}

func (SessionListIndex) TableName() string {
	return "session_list_indexes"
}

// BuildSessionListIndex 从会话对象构建列表索引快照。
func BuildSessionListIndex(session *Session) *SessionListIndex {
	if session == nil || session.SID == "" {
		return nil
	}
	visitorID, appID, _ := session.ParseSid()
	if visitorID == "" || appID == "" {
		return nil
	}
	lastActiveAt := session.LastActiveAt
	if lastActiveAt == 0 {
		lastActiveAt = session.LastVisitorMsgTime
	}
	if session.LastAgentReplyTime > lastActiveAt {
		lastActiveAt = session.LastAgentReplyTime
	}
	if lastActiveAt == 0 {
		lastActiveAt = session.CreatedAt
	}
	return &SessionListIndex{
		SID:                session.SID,
		VisitorID:          visitorID,
		AppID:              appID,
		Status:             session.Status(),
		CurAgentID:         session.CurAgentID,
		LastClientIP:       session.LastClientIP,
		LastUserAgent:      session.LastUserAgent,
		LastDevice:         session.LastDevice,
		LastGeo:            session.LastGeo,
		RatingScore:        session.RatingScore,
		RatingComment:      session.RatingComment,
		RatedAt:            session.RatedAt,
		LastVisitorMsgTime: session.LastVisitorMsgTime,
		LastAgentReplyTime: session.LastAgentReplyTime,
		LastAgentReadTime:  session.LastAgentReadTime,
		CreatedAt:          session.CreatedAt,
		LastActiveAt:       lastActiveAt,
		UnreadCount:        session.UnreadCount,
		LastMessage:        session.LastMessage,
		LastMessageType:    session.LastMessageType,
		UpdatedAt:          time.Now().Unix(),
	}
}
