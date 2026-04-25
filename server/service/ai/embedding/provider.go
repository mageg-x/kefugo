package embedding

import (
	"context"
	"fmt"

	"kefu-server/models"
)

type Provider interface {
	GetEmbedding(ctx context.Context, text string) ([]float32, error)
	GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
	Dims() int
}

func NewProvider(cfg *models.APIModelConfig) (Provider, error) {
	switch cfg.Provider {
	case "ollama":
		return NewOllamaProvider(cfg)
	case "openai", "qwen", "deepseek", "zhipu", "jina":
		return NewOpenAICompatibleProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}
}
