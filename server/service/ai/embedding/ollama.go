package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kefu-server/models"
)

type ollamaEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type ollamaEmbeddingResponse struct {
	Model     string    `json:"model"`
	Embedding []float32 `json:"embedding"`
}

type OllamaProvider struct {
	baseURL    string
	modelName  string
	dims       int
	timeoutSec int
	client     *http.Client
}

func NewOllamaProvider(cfg *models.APIModelConfig) (*OllamaProvider, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}
	return &OllamaProvider{
		baseURL:    baseURL,
		modelName:  cfg.ModelName,
		dims:       cfg.Dims,
		timeoutSec: timeout,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

func (p *OllamaProvider) Dims() int {
	return p.dims
}

func (p *OllamaProvider) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	results, err := p.GetEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("ollama embedding returned no results")
	}
	return results[0], nil
}

func (p *OllamaProvider) GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vec, err := p.getSingleEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		results = append(results, vec)
	}
	return results, nil
}

func (p *OllamaProvider) getSingleEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := ollamaEmbeddingRequest{
		Model: p.modelName,
		Input: text,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama embedding marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/embed", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama embedding create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama embedding read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embedding api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var ollamaResp struct {
		Model     string      `json:"model"`
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		var legacyResp ollamaEmbeddingResponse
		if err2 := json.Unmarshal(body, &legacyResp); err2 != nil {
			return nil, fmt.Errorf("ollama embedding unmarshal failed: %w", err)
		}
		return legacyResp.Embedding, nil
	}

	if len(ollamaResp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embedding returned empty embeddings")
	}
	return ollamaResp.Embeddings[0], nil
}
