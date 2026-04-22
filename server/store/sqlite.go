package store

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"

	"kefu-server/utils/logger"
)

type logWriter struct{}

var (
	DB *gorm.DB
)

func (w logWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	// If there are multiple lines, remove the first line
	if idx := strings.Index(msg, "\n"); idx != -1 {
		msg = msg[idx+1:]
	}
	// Remove all newlines
	msg = strings.ReplaceAll(msg, "\n", " ")
	// Log as Info level
	logger.Infof("[SQL] %s", msg)
	return len(p), nil
}

// InitDB 初始化 SQLite 数据库连接，并在启动时确保目录存在。
func InitDB(dbPath string) (*gorm.DB, error) {
	// Ensure data directory exists
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		err := os.MkdirAll(dbDir, 0755)
		if err != nil {
			logger.Errorf("sqlite data directory create failed: %v", err)
			return nil, err
		}
	}
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		logger.Errorf("sqlite open failed: %v", err)
		return nil, err
	}

	DB, err = gorm.Open(sqlite.New(sqlite.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: glogger.New(log.New(logWriter{}, "", 0), glogger.Config{LogLevel: glogger.Info}),
	})
	if err != nil {
		logger.Errorf("sqlite open failed: %v", err)
		return nil, err
	}

	return DB, nil
}
