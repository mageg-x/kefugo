package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	WSMessageTypeVisitor = "message.visitor"
	WSMessageTypeAgent   = "message.agent"
	WSMessageTypeSystem  = "message.system"
	WSMessageTypeTyping  = "message.typing"
	WSMessageTypeAck     = "message.ack"
	WSMessageTypeClose   = "message.close"
)

const (
	WSContentTypeText  = "text"
	WSContentTypeImage = "image"
	WSContentTypeAudio = "audio"
	WSContentTypeFile  = "file"
)

// WSPacket 是统一 WebSocket 消息信封。
type WSPacket struct {
	Type      string          `json:"type"`
	SID       string          `json:"sid,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
	MsgID     string          `json:"msg_id,omitempty"`
}

type WSReplyTo struct {
	MsgID       string `json:"msg_id,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Preview     string `json:"preview,omitempty"`
	Sender      string `json:"sender,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
}

type WSMessagePayload struct {
	MsgID       string     `json:"msg_id,omitempty"`
	ClientID    string     `json:"client_id,omitempty"`
	Code        string     `json:"code,omitempty"`
	From        string     `json:"from,omitempty"`
	AgentName   string     `json:"agent_name,omitempty"`
	FromName    string     `json:"from_name,omitempty"`
	SenderName  string     `json:"sender_name,omitempty"`
	Stream      bool       `json:"stream,omitempty"`
	StreamDelta bool       `json:"stream_delta,omitempty"`
	StreamFinal bool       `json:"stream_final,omitempty"`
	StreamKey   string     `json:"stream_key,omitempty"`
	ContentType string     `json:"content_type,omitempty"`
	Content     string     `json:"content,omitempty"`
	URL         string     `json:"url,omitempty"`
	Name        string     `json:"name,omitempty"`
	ReplyTo     *WSReplyTo `json:"reply_to,omitempty"`
	Duration    int64      `json:"duration,omitempty"`
	Size        int64      `json:"size,omitempty"`
	Timestamp   int64      `json:"timestamp,omitempty"`
}

// WSAckPayload 表示客户端送达确认包。
type WSAckPayload struct {
	MsgID     string `json:"msg_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

const (
	MaxMessageTextBytes  = 16 * 1024
	MaxReplyPreviewBytes = 512
	MaxMediaURLBytes     = 8 * 1024 * 1024
	MaxFileNameBytes     = 255
	MaxFileSizeBytes     = 8 * 1024 * 1024
)

// Normalize 对消息 payload 做兼容归一（默认文本、content/url 互补）。
func (p *WSMessagePayload) Normalize() {
	if p.ContentType == "" {
		p.ContentType = WSContentTypeText
	}
	if p.ContentType != WSContentTypeText &&
		p.ContentType != WSContentTypeImage &&
		p.ContentType != WSContentTypeAudio &&
		p.ContentType != WSContentTypeFile {
		p.ContentType = WSContentTypeText
	}
	if p.Content == "" {
		p.Content = p.URL
	}
}

// ParseWSMessagePayload 解析并校验 WS payload。
func ParseWSMessagePayload(raw json.RawMessage) (*WSMessagePayload, error) {
	var payload WSMessagePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	payload.Normalize()
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ParseWSAckPayload 解析并校验 ack payload。
func ParseWSAckPayload(raw json.RawMessage) (*WSAckPayload, error) {
	var payload WSAckPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	payload.MsgID = strings.TrimSpace(payload.MsgID)
	if payload.MsgID == "" {
		return nil, fmt.Errorf("msg_id required")
	}
	return &payload, nil
}

// Validate 按 content_type 校验 payload 字段合法性。
func (p *WSMessagePayload) Validate() error {
	if p.ReplyTo != nil && len(strings.TrimSpace(p.ReplyTo.Preview)) > MaxReplyPreviewBytes {
		return fmt.Errorf("reply preview too long")
	}

	switch p.ContentType {
	case WSContentTypeText:
		if len(p.Content) > MaxMessageTextBytes {
			return fmt.Errorf("text too long")
		}
	case WSContentTypeImage, WSContentTypeAudio, WSContentTypeFile:
		urlText := strings.TrimSpace(p.URL)
		if urlText == "" {
			urlText = strings.TrimSpace(p.Content)
		}
		if urlText == "" {
			return fmt.Errorf("url required")
		}
		if len(urlText) > MaxMediaURLBytes {
			return fmt.Errorf("url/content too large")
		}
		if !(strings.HasPrefix(urlText, "data:") ||
			strings.HasPrefix(urlText, "http://") ||
			strings.HasPrefix(urlText, "https://") ||
			strings.HasPrefix(urlText, "/")) {
			return fmt.Errorf("invalid media url")
		}
		if p.Size < 0 || p.Size > MaxFileSizeBytes {
			return fmt.Errorf("file too large")
		}
		if len(strings.TrimSpace(p.Name)) > MaxFileNameBytes {
			return fmt.Errorf("file name too long")
		}
		if err := validateMediaDataURL(urlText, p.ContentType); err != nil {
			return err
		}
		if p.ContentType == WSContentTypeAudio && p.Duration < 0 {
			return fmt.Errorf("invalid audio duration")
		}
	default:
		return fmt.Errorf("invalid content_type")
	}
	return nil
}

// validateMediaDataURL 校验 data url 的媒体类型是否符合 content_type。
func validateMediaDataURL(urlText, contentType string) error {
	if !strings.HasPrefix(urlText, "data:") {
		return nil
	}
	commaPos := strings.Index(urlText, ",")
	if commaPos <= 5 {
		return fmt.Errorf("invalid data url")
	}
	header := strings.ToLower(strings.TrimSpace(urlText[:commaPos]))
	switch contentType {
	case WSContentTypeImage:
		if !strings.HasPrefix(header, "data:image/") {
			return fmt.Errorf("invalid image media type")
		}
	case WSContentTypeAudio:
		if !strings.HasPrefix(header, "data:audio/") {
			return fmt.Errorf("invalid audio media type")
		}
	case WSContentTypeFile:
		if strings.HasPrefix(header, "data:image/") || strings.HasPrefix(header, "data:audio/") {
			return fmt.Errorf("invalid file media type")
		}
	}
	return nil
}

// BuildOutgoingWSPacket 构建对外发送的统一 WS 消息信封。
func BuildOutgoingWSPacket(packetType string, msg *Message, sidEncoded string) WSPacket {
	payload := WSMessagePayload{
		MsgID:       msg.MsgID,
		ContentType: msg.MsgType,
		Timestamp:   msg.Timestamp,
	}

	if len(msg.Meta) > 0 {
		var fromMeta WSMessagePayload
		if err := json.Unmarshal([]byte(msg.Meta), &fromMeta); err == nil {
			payload.From = fromMeta.From
			payload.AgentName = fromMeta.AgentName
			payload.FromName = fromMeta.FromName
			payload.SenderName = fromMeta.SenderName
			payload.Stream = fromMeta.Stream
			payload.StreamDelta = fromMeta.StreamDelta
			payload.StreamFinal = fromMeta.StreamFinal
			payload.StreamKey = fromMeta.StreamKey
			payload.ReplyTo = fromMeta.ReplyTo
			payload.Content = fromMeta.Content
			payload.URL = fromMeta.URL
			payload.Name = fromMeta.Name
			payload.Size = fromMeta.Size
			payload.Duration = fromMeta.Duration
			payload.ClientID = fromMeta.ClientID
			payload.Code = fromMeta.Code
			if fromMeta.ContentType != "" {
				payload.ContentType = fromMeta.ContentType
			}
		}
	}

	if payload.Content == "" {
		payload.Content = msg.Content
	}
	if payload.URL == "" && (payload.ContentType == WSContentTypeImage || payload.ContentType == WSContentTypeAudio || payload.ContentType == WSContentTypeFile) {
		payload.URL = msg.Content
	}
	payload.Normalize()
	payloadBytes, _ := json.Marshal(payload)

	pkt := WSPacket{
		Type:      packetType,
		Payload:   payloadBytes,
		Timestamp: msg.Timestamp,
		MsgID:     msg.MsgID,
	}

	if sidEncoded != "" {
		pkt.SID = sidEncoded
	}

	return pkt
}
