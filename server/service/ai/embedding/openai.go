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

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type OpenAICompatibleProvider struct {
	baseURL    string
	apiKey     string
	modelName  string
	dims       int
	timeoutSec int
	client     *http.Client
}

func NewOpenAICompatibleProvider(cfg *models.APIModelConfig) (*OpenAICompatibleProvider, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("openai-compatible embedding base_url is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}
	return &OpenAICompatibleProvider{
		baseURL:    baseURL,
		apiKey:     strings.TrimSpace(cfg.APIKey),
		modelName:  cfg.ModelName,
		dims:       cfg.Dims,
		timeoutSec: timeout,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

func (p *OpenAICompatibleProvider) Dims() int {
	return p.dims
}

func (p *OpenAICompatibleProvider) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	results, err := p.GetEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("openai embedding returned no results")
	}
	return results[0], nil
}

func (p *OpenAICompatibleProvider) GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := openAIEmbeddingRequest{
		Model: p.modelName,
		Input: texts,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai embedding marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/embeddings", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("openai embedding create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(p.apiKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai embedding read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embedding api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var embResp openAIEmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("openai embedding unmarshal failed: %w", err)
	}

	results := make([][]float32, len(embResp.Data))
	for i, d := range embResp.Data {
		results[i] = d.Embedding
	}
	return results, nil
}
