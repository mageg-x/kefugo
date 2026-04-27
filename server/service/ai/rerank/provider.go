package rerank

import (
	"fmt"

	"kefu-server/models"
)

type Candidate struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type Provider interface {
	Rerank(query string, candidates []Candidate, topN int) ([]Candidate, error)
}

func NewProvider(cfg *models.APIModelConfig) (Provider, error) {
	switch cfg.Provider {
	case "ollama":
		return NewOllamaProvider(cfg)
	case "cohere":
		return NewCohereProvider(cfg)
	case "openai", "qwen", "deepseek", "zhipu", "jina":
		return NewOpenAICompatibleProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported rerank provider: %s", cfg.Provider)
	}
}
