package controllers

import (
	"context"
	"time"

	"kefu-server/models"
	"kefu-server/service"
)

func modelTimeoutByType(modelType string, fallback time.Duration) time.Duration {
	if cfg, err := service.GetEnabledAPIModelConfigByType(modelType); err == nil {
		if cfg.TimeoutSec > 0 {
			timeout := time.Duration(cfg.TimeoutSec) * time.Second
			if timeout > 0 {
				return timeout
			}
		}
	}
	return fallback
}

func modelTimeoutByConfig(cfg models.APIModelConfig, fallback time.Duration) time.Duration {
	if cfg.TimeoutSec > 0 {
		timeout := time.Duration(cfg.TimeoutSec) * time.Second
		if timeout > 0 {
			return timeout
		}
	}
	return fallback
}

func knowledgeIndexTimeout() time.Duration {
	timeout := 10 * time.Minute
	if cfg, err := service.GetEnabledAPIModelConfigByType(string(models.AIModelTypeEmbedding)); err == nil {
		if cfg.TimeoutSec > 0 {
			candidate := time.Duration(cfg.TimeoutSec+30) * time.Second
			if candidate > timeout {
				timeout = candidate
			}
		}
	}
	return timeout
}

func knowledgeIndexContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), knowledgeIndexTimeout())
}

func knowledgeSearchTimeout() time.Duration {
	timeout := modelTimeoutByType(string(models.AIModelTypeEmbedding), 60*time.Second) + 20*time.Second
	if _, err := service.GetEnabledAPIModelConfigByType(string(models.AIModelTypeRerank)); err == nil {
		timeout += modelTimeoutByType(string(models.AIModelTypeRerank), 30*time.Second) + 10*time.Second
	}
	if timeout < 45*time.Second {
		timeout = 45 * time.Second
	}
	if timeout > 10*time.Minute {
		timeout = 10 * time.Minute
	}
	return timeout
}

func knowledgeSearchContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), knowledgeSearchTimeout())
}

func knowledgeQAContext() (context.Context, context.CancelFunc) {
	return knowledgeQAContextWithParent(context.Background())
}

func knowledgeQAContextWithParent(parent context.Context) (context.Context, context.CancelFunc) {
	timeout := knowledgeSearchTimeout() + modelTimeoutByType(string(models.AIModelTypeChat), 120*time.Second) + 20*time.Second
	if timeout < 90*time.Second {
		timeout = 90 * time.Second
	}
	if timeout > 15*time.Minute {
		timeout = 15 * time.Minute
	}
	return context.WithTimeout(parent, timeout)
}

func modelConfigOperationContext(cfg models.APIModelConfig, extra time.Duration, fallback time.Duration) (context.Context, context.CancelFunc) {
	timeout := modelTimeoutByConfig(cfg, fallback) + extra
	if timeout < 30*time.Second {
		timeout = 30 * time.Second
	}
	if timeout > 15*time.Minute {
		timeout = 15 * time.Minute
	}
	return context.WithTimeout(context.Background(), timeout)
}
