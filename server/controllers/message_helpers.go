package controllers

import (
	"encoding/json"
	"strings"
	"time"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/utils/logger"
)

// saveSystemMessageForSession 统一保存系统消息到消息存储并回写会话摘要字段
// 该方法用于"会话已关闭/已转接/已读"等系统提示类消息
// session: 会话对象
// content: 系统消息内容
// ts: 消息时间戳
// 返回值：构建好的系统消息对象
func saveSystemMessageForSession(session *models.Session, content string, ts int64) *models.Message {
	if session == nil {
		logger.Warnf("save system message skipped: session is nil")
		return nil
	}
	// 如果时间戳无效，使用当前时间
	if ts <= 0 {
		ts = time.Now().Unix()
	}

	// 构建系统消息对象
	msg := &models.Message{
		MsgType:   models.WSMessageTypeSystem,
		Content:   strings.TrimSpace(content),
		Timestamp: ts,
	}

	// 构建消息元数据
	meta := map[string]interface{}{
		"from":         "system",
		"from_name":    "系统",
		"sender_name":  "系统",
		"content_type": models.WSContentTypeText,
		"content":      msg.Content,
		"timestamp":    ts,
	}
	if b, err := json.Marshal(meta); err == nil {
		msg.Meta = string(b)
	}

	// 保存消息到消息服务
	ms := service.GetMsgService()
	ss := service.GetSessionService()
	if ms != nil {
		if msgID, err := ms.SaveMessage(session.VisitorID(), session.AppID(), session.SessionSeq(), msg); err == nil {
			msg.MsgID = msgID
			logger.Debugf("system message saved sid=%s msg_id=%s content=%s", session.SID, msgID, content)
		} else {
			logger.Errorf("system message save failed sid=%s err=%v", session.SID, err)
		}
	}

	// 更新会话的最后消息信息
	if ss != nil {
		if _, err := ss.UpdateSession(session.SID, func(s *models.Session) error {
			s.TouchMessage(models.WSMessageTypeSystem, msg.Content, ts)
			return nil
		}); err != nil {
			logger.Errorf("system message session update failed sid=%s err=%v", session.SID, err)
		}
	}

	// 使会话列表缓存失效
	invalidateSessionListCache()

	return msg
}
