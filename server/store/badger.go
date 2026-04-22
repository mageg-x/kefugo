package store

import (
	"fmt"
	"kefu-server/utils/logger"
	"strconv"

	"github.com/dgraph-io/badger/v4"
)

var (
	KV *badger.DB
)

// GetStore 返回全局 Badger 句柄。
func GetStore() *badger.DB { return KV }

// InitStore 初始化 Badger 存储（单例）。
func InitStore(dbPath string) (*badger.DB, error) {
	if KV != nil { // 单例
		return KV, nil
	}

	opts := badger.DefaultOptions(dbPath).
		WithLogger(nil).                // 禁用 badger 自带日志（用你的 logger）
		WithValueLogFileSize(128 << 20) // 128MB

	db, err := badger.Open(opts)
	if err != nil {
		logger.Errorf("badger open failed: %v", err)
		return nil, fmt.Errorf("failed to open badger: %w", err)
	}
	KV = db
	return KV, nil
}

// GetNextID 原子获取下一个自增 ID（线程安全）。
func GetNextID(counterKey string) (uint32, error) {
	if KV == nil {
		return 0, fmt.Errorf("store not initialized")
	}

	var id uint32
	err := KV.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(counterKey))
		if err == badger.ErrKeyNotFound {
			id = 1
		} else if err != nil {
			logger.Errorf("badger counter get failed key=%s err=%v", counterKey, err)
			return err
		} else {
			val, _ := item.ValueCopy(nil)
			current, _ := strconv.ParseUint(string(val), 10, 64)
			id = uint32(current + 1)
		}
		return txn.Set([]byte(counterKey), []byte(strconv.FormatUint(uint64(id), 10)))
	})
	return id, err
}

// NewSessionSeq 获取全局会话序号。
func NewSessionSeq() (uint32, error) {
	seq, err := GetNextID("counter:session")
	if err != nil {
		logger.Errorf("session seq generate failed err=%v", err)
		return 0, err
	}
	return seq, nil
}

// NewMsgSeq 获取全局消息序号。
func NewMsgSeq() (uint32, error) {
	seq, err := GetNextID("counter:message")
	if err != nil {
		logger.Errorf("message seq generate failed err=%v", err)
		return 0, err
	}
	return seq, nil
}
