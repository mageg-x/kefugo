package models

import (
	"fmt"
	"strconv"
	"strings"
)

type Message struct {
	MsgID     string `json:"msg_id"`   // m:{visitor_id}:{app_id}:{session_seq}:{msg_seq}
	MsgType   string `json:"msg_type"` // "text", "image", etc.
	Content   string `json:"content,omitempty"`
	Meta      string `json:"meta,omitempty"` // JSON: {content_type,content,url,name,size,duration}
	Timestamp int64  `json:"timestamp"`
}

// ParseMessageID 从 messageID 中解析字段
func ParseMessageID(messageID string) (visitorID, appID string, sessionSeq, msgSeq uint32) {
	parts := strings.Split(messageID, ":")
	if len(parts) != 5 {
		return "", "", 0, 0
	}

	visitorID = parts[1]
	appID = parts[2]

	sessionSeq64, err1 := strconv.ParseUint(parts[3], 10, 32)
	msgSeq64, err2 := strconv.ParseUint(parts[4], 10, 32)
	if err1 != nil || err2 != nil {
		return "", "", 0, 0
	}

	return visitorID, appID, uint32(sessionSeq64), uint32(msgSeq64)
}

// GetMessageID 生成 messageID
func GetMessageID(visitorID, appID string, sessionSeq, msgSeq uint32) string {
	return fmt.Sprintf("m:%s:%s:%010d:%010d", visitorID, appID, sessionSeq, msgSeq)
}
