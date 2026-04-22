package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

// 1. 消息存储
// m:{visitor_id}:{app_id}:{session_seq}:{msg_seq}
// m:alice:shop123:1234:23445
// │   │     │       │      └─ msg seq
// │   │     │       └─ session seq
// │   │     └─ app_id
// │   └─ visitor_id

type MessageService struct {
	kv *badger.DB
}

var (
	instMessage *MessageService
)

func GetMsgService() *MessageService {
	if instMessage != nil {
		return instMessage
	}

	if kv := store.GetStore(); kv == nil { // 单例
		return nil
	} else {
		instMessage = &MessageService{kv: kv}
		return instMessage
	}
}

// SaveMessage 保存消息并返回消息 ID。
func (m *MessageService) SaveMessage(visitorID, appID string, sessionSeq uint32, msg *models.Message) (string, error) {
	msgSeq, err := store.NewMsgSeq()
	if err != nil {
		logger.Errorf("message seq generate failed err=%v", err)
		return "", err
	}
	msgID := models.GetMessageID(visitorID, appID, sessionSeq, msgSeq)
	msg.MsgID = msgID
	data, _ := json.Marshal(msg)

	// 设置 TTL：30 天
	ttl := 30 * 24 * time.Hour

	err = m.kv.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(msgID), data).WithTTL(ttl)
		return txn.SetEntry(entry)
	})
	if err != nil {
		logger.Errorf("message save failed msg_id=%s err=%v", msgID, err)
		return "", err
	}
	return msgID, err
}

// GetMessage 根据消息 ID 读取单条消息。
func (m *MessageService) GetMessage(msgID string) (*models.Message, error) {
	var msg models.Message
	err := m.kv.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(msgID))
		if err != nil {
			logger.Errorf("message get failed msg_id=%s err=%v", msgID, err)
			return err
		}
		val, _ := item.ValueCopy(nil)
		return json.Unmarshal(val, &msg)
	})
	if err != nil {
		logger.Errorf("message decode failed msg_id=%s err=%v", msgID, err)
		return nil, err
	}
	return &msg, err
}

// GetMessagesBySession 获取某会话的消息列表（按时间正序）
func (m *MessageService) GetMessagesBySession(sessionID string, limit int) ([]*models.Message, error) {
	if limit <= 0 || limit > 100 { // 防止滥用
		limit = 50
	}

	// 1. 解析 sessionID: s:{visitor}:{app}:{session_seq}
	parts := strings.Split(sessionID, ":")
	if len(parts) != 4 || parts[0] != "s" {
		return nil, fmt.Errorf("invalid sessionID format: %s", sessionID)
	}
	visitorID := parts[1]
	appID := parts[2]
	sessionSeqStr := parts[3]

	// 2. 构造消息 key 前缀: m:{visitor}:{app}:{session_seq}:
	msgPrefix := fmt.Sprintf("m:%s:%s:%s:", visitorID, appID, sessionSeqStr)

	msgs, err := m.collectMessagesByPrefix(msgPrefix, "", limit, 0)

	if err != nil {
		logger.Errorf("messages by session query failed sid=%s err=%v", sessionID, err)
		return nil, err
	}

	return msgs, nil
}

// GetLatestMessageBySession 获取某会话最后一条消息
func (m *MessageService) GetLatestMessageBySession(sessionID string) (*models.Message, error) {
	msgs, err := m.GetMessagesBySession(sessionID, 1)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	return msgs[len(msgs)-1], nil
}

// GetMessagesBySessionBefore 按 msgID 游标向前分页，返回正序（旧->新）
func (m *MessageService) GetMessagesBySessionBefore(sessionID, beforeMsgID string, limit int) ([]*models.Message, error) {
	return m.GetMessagesBySessionBeforeSnapshot(sessionID, beforeMsgID, limit, 0)
}

// GetMessagesBySessionBeforeSnapshot 使用 snapshotTs 快照时间分页，避免并发写入导致翻页抖动。
func (m *MessageService) GetMessagesBySessionBeforeSnapshot(sessionID, beforeMsgID string, limit int, snapshotTs int64) ([]*models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	parts := strings.Split(sessionID, ":")
	if len(parts) != 4 || parts[0] != "s" {
		return nil, fmt.Errorf("invalid sessionID format: %s", sessionID)
	}
	visitorID := parts[1]
	appID := parts[2]
	sessionSeqStr := parts[3]
	msgPrefix := fmt.Sprintf("m:%s:%s:%s:", visitorID, appID, sessionSeqStr)

	msgs, err := m.collectMessagesByPrefix(msgPrefix, beforeMsgID, limit, snapshotTs)
	if err != nil {
		logger.Errorf("messages before query failed sid=%s before=%s err=%v", sessionID, beforeMsgID, err)
		return nil, err
	}
	return msgs, nil
}

func (m *MessageService) collectMessagesByPrefix(prefix string, beforeMsgID string, limit int, snapshotTs int64) ([]*models.Message, error) {
	type kvMessage struct {
		Key string
		Msg *models.Message
	}
	var all []kvMessage

	err := m.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			// 分页游标语义为“严格小于 before”，避免重复返回游标消息导致死循环。
			if beforeMsgID != "" && key >= beforeMsgID {
				continue
			}

			val, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			var msg models.Message
			if err := json.Unmarshal(val, &msg); err != nil {
				continue
			}
			if snapshotTs > 0 && msg.Timestamp > snapshotTs {
				continue
			}
			all = append(all, kvMessage{Key: key, Msg: &msg})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Key < all[j].Key
	})
	if limit > 0 && len(all) > limit {
		all = all[len(all)-limit:]
	}

	result := make([]*models.Message, 0, len(all))
	for _, row := range all {
		result = append(result, row.Msg)
	}
	return result, nil
}

// Close 关闭消息存储句柄。
func (m *MessageService) Close() error {
	m.kv.Close()
	m.kv = nil
	return nil
}
