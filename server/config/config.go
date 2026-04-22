package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"kefu-server/utils/logger"
)

type Config struct {
	Admin    AdminConfig    `yaml:"admin"`
	Security SecurityConfig `yaml:"security"`
	Log      LogConfig      `yaml:"log"`
}

type AdminConfig struct {
	Address   string `yaml:"address"`
	DataDir   string `yaml:"data_dir"`
	Database  string `yaml:"database"`
	Storage   string `yaml:"storage"`
	UploadDir string `yaml:"upload_dir"`
}

type SecurityConfig struct {
	JWTSecret string `yaml:"jwt_secret"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

var AppConfig *Config

// SetConfig 设置全局运行时配置。
func SetConfig(cfg *Config) {
	AppConfig = cfg
}

// GetConfig 获取配置
func GetConfig() *Config {
	return AppConfig
}

// DefaultDataDir 返回不同平台下的数据目录默认值。
func DefaultDataDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		base := strings.TrimSpace(os.Getenv("APPDATA"))
		if base == "" {
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "kefu")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "kefu")
	default:
		return filepath.Join(home, ".kefu")
	}
}

// ResolveDataDir 解析数据目录参数（含 ~ 展开），并返回清洗后的路径。
func ResolveDataDir(input string) (string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		value = DefaultDataDir()
	}
	if strings.HasPrefix(value, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Errorf("resolve home dir failed err=%v", err)
			return "", fmt.Errorf("resolve home dir failed: %w", err)
		}
		if value == "~" {
			value = home
		} else {
			value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	return filepath.Clean(value), nil
}

// BuildFromCLI 使用命令行参数构建运行配置，不依赖外部配置文件。
func BuildFromCLI(addr, dataDir, jwtSecret, logLevel string) (*Config, error) {
	resolvedDataDir, err := ResolveDataDir(dataDir)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Admin: AdminConfig{
			Address:   "0.0.0.0:5300",
			DataDir:   resolvedDataDir,
			Database:  filepath.Join(resolvedDataDir, "kefu.db"),
			Storage:   filepath.Join(resolvedDataDir, "message"),
			UploadDir: filepath.Join(resolvedDataDir, "uploads"),
		},
		Security: SecurityConfig{
			JWTSecret: strings.TrimSpace(jwtSecret),
		},
		Log: LogConfig{
			Level: "info",
		},
	}

	if strings.TrimSpace(addr) != "" {
		cfg.Admin.Address = strings.TrimSpace(addr)
	}
	if strings.TrimSpace(logLevel) != "" {
		cfg.Log.Level = strings.TrimSpace(logLevel)
	}

	return cfg, nil
}
