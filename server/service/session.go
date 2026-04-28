package service

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"gorm.io/gorm"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

// 1. 会话存储
// s:{visitor_id}:{app_id}:{session_seq} 为 会话 id
// s:alice:shop123:1234
// │   │     │       │
// │   │     │       └─ session seq
// │   │     └─ app_id
// │   └─ visitor_id

type SessionService struct {
	kv       *badger.DB
	lockPool [256]sync.Mutex
}

const (
	SessionTimeout = 24 * time.Hour // 可配置
)

var (
	instSessionService *SessionService
)

func GetSessionService() *SessionService {
	if instSessionService != nil {
		return instSessionService
	}

	if kv := store.GetStore(); kv == nil { // 单例
		logger.Errorf("session store unavailable")
		return nil
	} else {
		instSessionService = &SessionService{kv: kv}
		return instSessionService
	}
}

func (s *SessionService) lockKey(key string) func() {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	idx := hasher.Sum32() % uint32(len(s.lockPool))
	s.lockPool[idx].Lock()
	return s.lockPool[idx].Unlock
}

// sessionAppIndexKey 返回按 app_id 建立的会话二级索引 key。
func sessionAppIndexKey(appID, sid string) string {
	return fmt.Sprintf("si:app:%s:%s", strings.TrimSpace(appID), strings.TrimSpace(sid))
}

// upsertSessionIndexes 更新会话二级索引，便于按应用快速扫描会话。
func (s *SessionService) upsertSessionIndexes(txn *badger.Txn, session *models.Session) error {
	if session == nil || session.SID == "" {
		return nil
	}
	appID := strings.TrimSpace(session.AppID())
	if appID == "" {
		return nil
	}
	return txn.Set([]byte(sessionAppIndexKey(appID, session.SID)), []byte{1})
}

// GetLatestSession 获取访客最新会话（按 session_seq 最大）
func (s *SessionService) GetLatestSession(visitorID, appID string) (*models.Session, error) {
	prefix := fmt.Sprintf("s:%s:%s:", visitorID, appID)
	var latestSession *models.Session

	err := s.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()

		seekKey := append([]byte(nil), []byte(prefix)...)
		seekKey = append(seekKey, 0xFF)
		for it.Seek(seekKey); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			if !strings.HasPrefix(string(key), prefix) {
				break
			}
			val, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			var sess models.Session
			if err := json.Unmarshal(val, &sess); err != nil {
				continue
			}
			latestSession = &sess
			return nil
		}

		return fmt.Errorf("session not found for visitor=%s app=%s", visitorID, appID)
	})

	if err != nil {
		logger.Errorf("session latest query failed visitor=%s app=%s err=%v", visitorID, appID, err)
		return nil, err
	}
	return latestSession, nil
}

// CreateSession 创建新会话
func (s *SessionService) CreateSession(visitorID, appID string) (*models.Session, error) {
	sessionSeq, err := store.NewSessionSeq() // 需要你实现这个全局序号生成器
	if err != nil {
		logger.Errorf("session seq generate failed visitor=%s app=%s err=%v", visitorID, appID, err)
		return nil, err
	}

	sid := models.GetSessionID(visitorID, appID, sessionSeq)
	now := time.Now().Unix()

	session := &models.Session{
		SID:                sid,
		CreatedAt:          now,
		LastVisitorMsgTime: 0,
		LastAgentReplyTime: 0,
		LastAgentReadTime:  0,
		Closed:             false,
		FollowUp:           false,
	}

	data, _ := json.Marshal(session)

	err = s.kv.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(sid), data); err != nil {
			return err
		}
		return s.upsertSessionIndexes(txn, session)
	})
	if err != nil {
		logger.Errorf("session create save failed sid=%s err=%v", sid, err)
		return nil, err
	}
	s.upsertSessionListIndex(session)

	return session, nil
}

// GetOrCreateSession 获取或创建会话
func (s *SessionService) GetOrCreateSession(visitorID, appID string) (*models.Session, error) {
	unlock := s.lockKey("visitor-app:" + strings.TrimSpace(visitorID) + ":" + strings.TrimSpace(appID))
	defer unlock()

	session, err := s.GetLatestSession(visitorID, appID)
	if err == nil {
		// 计算最后活跃时间
		lastActive := session.LastVisitorMsgTime
		if session.LastAgentReplyTime > lastActive {
			lastActive = session.LastAgentReplyTime
		}
		// 新会话尚未收发消息时，使用创建时间兜底，避免被误判为“超时”并重复创建会话。
		if lastActive <= 0 {
			lastActive = session.CreatedAt
		}
		if lastActive <= 0 {
			lastActive = time.Now().Unix()
		}

		// 如果会话未关闭，但已超时 → 自动关闭并新建
		if !session.Closed && time.Since(time.Unix(lastActive, 0)) > resolveSessionTimeout() {
			// 自动关闭旧会话
			session.Close()
			if saveErr := s.saveSessionNoLock(session); saveErr != nil {
				logger.Errorf("session timeout close failed sid=%s err=%v", session.SID, saveErr)
				return nil, saveErr
			}

			// 创建新会话
			return s.CreateSession(visitorID, appID)
		}

		// 未超时且未关闭 → 复用
		if !session.Closed {
			return session, nil
		}
	}

	// 无有效会话 → 创建新会话
	return s.CreateSession(visitorID, appID)
}

// resolveSessionTimeout 从系统设置读取会话超时时间（秒）。
func resolveSessionTimeout() time.Duration {
	if store.DB == nil {
		return SessionTimeout
	}
	type systemSettings struct {
		Timeout int `json:"timeout"`
	}
	var row struct {
		Value string `gorm:"column:value"`
	}
	if err := store.DB.Table("system_settings").Select("value").Where("key = ?", "system_settings").Take(&row).Error; err != nil {
		return SessionTimeout
	}
	var cfg systemSettings
	if err := json.Unmarshal([]byte(row.Value), &cfg); err != nil {
		return SessionTimeout
	}
	if cfg.Timeout <= 0 {
		return SessionTimeout
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	minTimeout := 30 * time.Second
	if timeout < minTimeout {
		return minTimeout
	}
	return timeout
}

// SaveSession 保存会话（用于状态更新）
func (s *SessionService) SaveSession(session *models.Session) error {
	if session == nil {
		logger.Errorf("session save session nil")
		return fmt.Errorf("session is nil")
	}
	unlock := s.lockKey("sid:" + session.SID)
	defer unlock()
	return s.saveSessionNoLock(session)
}

func (s *SessionService) saveSessionNoLock(session *models.Session) error {
	if session.SID == "" {
		logger.Errorf("session save sid empty")
		return fmt.Errorf("session SID is empty")
	}
	data, _ := json.Marshal(session)

	err := s.kv.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte(session.SID), data); err != nil {
			return err
		}
		return s.upsertSessionIndexes(txn, session)
	})
	if err != nil {
		logger.Errorf("session save failed sid=%s err=%v", session.SID, err)
		return err
	}
	s.upsertSessionListIndex(session)
	return err
}

// 获取会话内容
func (s *SessionService) GetSession(sessionID string) (*models.Session, error) {
	return s.getSessionNoLock(sessionID)
}

func (s *SessionService) getSessionNoLock(sessionID string) (*models.Session, error) {
	var session *models.Session
	if sessionID == "" {
		logger.Errorf("session get sid empty")
		return nil, fmt.Errorf("sessionID is empty")
	}

	if s.kv == nil {
		logger.Errorf("session get store unavailable sid=%s", sessionID)
		return nil, fmt.Errorf("kv is not initialized")
	}

	err := s.kv.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(sessionID))
		if err != nil {
			logger.Errorf("session get failed sid=%s err=%v", sessionID, err)
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return json.Unmarshal(val, &session)
	})

	if err != nil {
		logger.Errorf("session get decode failed sid=%s err=%v", sessionID, err)
		return nil, err
	}
	return session, nil
}

func (s *SessionService) UpdateSession(sessionID string, mutate func(*models.Session) error) (*models.Session, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is empty")
	}

	unlock := s.lockKey("sid:" + sessionID)
	defer unlock()

	session, err := s.getSessionNoLock(sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	if mutate != nil {
		if err := mutate(session); err != nil {
			return nil, err
		}
	}
	if err := s.saveSessionNoLock(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *SessionService) IterateSessions(fn func(*models.Session) bool) error {
	if s.kv == nil {
		return fmt.Errorf("kv is not initialized")
	}
	return s.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("s:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			val, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			var session models.Session
			if err := json.Unmarshal(val, &session); err != nil {
				continue
			}
			if !fn(&session) {
				break
			}
		}
		return nil
	})
}

func (s *SessionService) IterateSessionsByAppIDs(appIDs []string, fn func(*models.Session) bool) error {
	if s.kv == nil {
		return fmt.Errorf("kv is not initialized")
	}
	if len(appIDs) == 0 {
		return nil
	}

	uniqueAppIDs := make([]string, 0, len(appIDs))
	seenApp := make(map[string]struct{}, len(appIDs))
	for _, appID := range appIDs {
		normalized := strings.TrimSpace(appID)
		if normalized == "" {
			continue
		}
		if _, ok := seenApp[normalized]; ok {
			continue
		}
		seenApp[normalized] = struct{}{}
		uniqueAppIDs = append(uniqueAppIDs, normalized)
	}
	if len(uniqueAppIDs) == 0 {
		return nil
	}

	return s.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		seenSID := make(map[string]struct{})
		stop := false
		for _, appID := range uniqueAppIDs {
			if stop {
				break
			}
			prefix := []byte(fmt.Sprintf("si:app:%s:", appID))
			it.Seek(prefix)
			for ; it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				key := string(item.Key())
				sid := strings.TrimPrefix(key, fmt.Sprintf("si:app:%s:", appID))
				if sid == "" {
					continue
				}
				if _, ok := seenSID[sid]; ok {
					continue
				}
				seenSID[sid] = struct{}{}

				sessionItem, err := txn.Get([]byte(sid))
				if err != nil {
					continue
				}
				val, err := sessionItem.ValueCopy(nil)
				if err != nil {
					continue
				}
				var session models.Session
				if err := json.Unmarshal(val, &session); err != nil {
					continue
				}
				if !fn(&session) {
					stop = true
					break
				}
			}
		}
		return nil
	})
}

func (s *SessionService) RebuildSessionIndexes() error {
	if s.kv == nil {
		return fmt.Errorf("kv is not initialized")
	}

	type indexPair struct {
		key string
	}
	indexes := make([]indexPair, 0, 1024)

	err := s.kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("s:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			val, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			var session models.Session
			if err := json.Unmarshal(val, &session); err != nil {
				continue
			}
			appID := strings.TrimSpace(session.AppID())
			if appID == "" || session.SID == "" {
				continue
			}
			indexes = append(indexes, indexPair{key: sessionAppIndexKey(appID, session.SID)})
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		return nil
	}

	for start := 0; start < len(indexes); start += 500 {
		end := start + 500
		if end > len(indexes) {
			end = len(indexes)
		}
		if err := s.kv.Update(func(txn *badger.Txn) error {
			for _, pair := range indexes[start:end] {
				if err := txn.Set([]byte(pair.key), []byte{1}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *SessionService) upsertSessionListIndex(session *models.Session) {
	if store.DB == nil || session == nil || session.SID == "" {
		return
	}
	idx := models.BuildSessionListIndex(session)
	if idx == nil {
		return
	}
	if err := store.DB.Save(idx).Error; err != nil {
		logger.Errorf("session list index upsert failed sid=%s err=%v", session.SID, err)
	}
}

// RebuildSessionListIndexes 全量重建 SQLite 会话列表索引。
func (s *SessionService) RebuildSessionListIndexes() error {
	if store.DB == nil {
		return nil
	}
	if err := store.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.SessionListIndex{}).Error; err != nil {
		return err
	}
	batch := make([]*models.SessionListIndex, 0, 1000)
	var iterateErr error
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := store.DB.CreateInBatches(batch, 200).Error; err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	if err := s.IterateSessions(func(session *models.Session) bool {
		idx := models.BuildSessionListIndex(session)
		if idx == nil {
			return true
		}
		batch = append(batch, idx)
		if len(batch) >= 1000 {
			if err := flush(); err != nil {
				logger.Errorf("session list index batch flush failed err=%v", err)
				iterateErr = err
				return false
			}
		}
		return true
	}); err != nil {
		return err
	}
	if iterateErr != nil {
		return iterateErr
	}
	return flush()
}

func (s *SessionService) CountActiveSessionsByAgent(agentUserName string) (int64, error) {
	var count int64
	err := s.IterateSessions(func(session *models.Session) bool {
		if session.CurAgentID == agentUserName && !session.Closed {
			count++
		}
		return true
	})
	return count, err
}
