package appboot

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/gin-gonic/gin"
	"kefu-server/config"
	"kefu-server/models"
	"kefu-server/router"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

type Options struct {
	Addr      string
	DataDir   string
	JWTSecret string
	LogLevel  string
}

// InitRuntime 初始化服务运行时依赖并返回路由引擎与配置。
func InitRuntime(opts Options) (*gin.Engine, *config.Config, error) {
	logger.SetLevel(logger.INFO)

	cfg, err := config.BuildFromCLI(opts.Addr, opts.DataDir, opts.JWTSecret, opts.LogLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("build runtime config failed: %w", err)
	}
	config.SetConfig(cfg)

	logger.SetLevel(parseLogLevel(cfg.Log.Level))
	logger.Infof("runtime config: addr=%s data=%s db=%s storage=%s uploads=%s vector=sqlite-vec log=%s",
		cfg.Admin.Address, cfg.Admin.DataDir, cfg.Admin.Database, cfg.Admin.Storage, cfg.Admin.UploadDir, cfg.Log.Level)

	_, err = store.InitStore(cfg.Admin.Storage)
	if err != nil {
		return nil, nil, fmt.Errorf("storage initialization failed: %w", err)
	}
	startBadgerGC()

	db, err := store.InitDB(cfg.Admin.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("database connection failed: %w", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.App{},
		&models.AgentSetting{},
		&models.APIModelConfig{},
		&models.AppAPIKey{},
		&models.KnowledgeBase{},
		&models.KnowledgeDocument{},
		&models.KnowledgeChunk{},
		&models.KnowledgeQAFeedback{},
		&models.KnowledgeVectorCollection{},
		&models.KnowledgeVectorEntry{},
		&models.KnowledgeArticle{},
		&models.FAQItem{},
		&models.QuickReply{},
		&models.AuditLog{},
		&models.SystemSetting{},
		&models.SessionListIndex{},
		&models.VecTable{},
		&models.RebuildTask{},
	); err != nil {
		return nil, nil, fmt.Errorf("database migration failed: %w", err)
	}

	if err := models.CreateDefaultUsers(db); err != nil {
		return nil, nil, fmt.Errorf("create default users failed: %w", err)
	}
	if err := models.CreateDefaultApps(db); err != nil {
		return nil, nil, fmt.Errorf("create default apps failed: %w", err)
	}

	migrateLegacyRoles()
	if ss := service.GetSessionService(); ss != nil {
		if err := ss.RebuildSessionIndexes(); err != nil {
			logger.Warnf("rebuild session indexes failed: %v", err)
		}
		if err := ss.RebuildSessionListIndexes(); err != nil {
			logger.Warnf("rebuild session list indexes failed: %v", err)
		}
	}

	r := router.SetupRouter()
	return r, cfg, nil
}

func BuildOpenURL(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return "http://localhost:5300/"
	}

	host, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		return "http://localhost:5300/"
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%s/", host, port)
}

// migrateLegacyRoles 对历史脏角色做统一归一，避免权限模型长期分叉。
func migrateLegacyRoles() {
	if store.DB == nil {
		return
	}
	if err := store.DB.Model(&models.User{}).
		Where("role NOT IN ?", []string{"admin", "agent"}).
		Update("role", "admin").Error; err != nil {
		logger.Warnf("legacy role migration failed: %v", err)
		return
	}
}

// parseLogLevel 把字符串日志级别映射为内部枚举。
func parseLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return logger.TRACE
	case "debug":
		return logger.DEBUG
	case "warn":
		return logger.WARN
	case "error":
		return logger.ERROR
	case "fatal":
		return logger.FATAL
	case "panic":
		return logger.PANIC
	case "info":
		fallthrough
	default:
		return logger.INFO
	}
}

// startBadgerGC 周期触发 Badger value log GC，回收 TTL 过期数据空间。
func startBadgerGC() {
	kv := store.GetStore()
	if kv == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			for {
				err := kv.RunValueLogGC(0.5)
				if err == nil {
					continue
				}
				if errors.Is(err, badger.ErrNoRewrite) {
					break
				}
				logger.Warnf("badger value log gc failed: %v", err)
				break
			}
		}
	}()
}
