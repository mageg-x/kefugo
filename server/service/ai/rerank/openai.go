package rerank

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

type openAICompatibleRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

type openAICompatibleRerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
		Document       struct {
			Text string `json:"text"`
		} `json:"document"`
	} `json:"results"`
	Model string `json:"model"`
}

type OpenAICompatibleProvider struct {
	apiKey     string
	baseURL    string
	modelName  string
	timeoutSec int
	client     *http.Client
}

func normalizeOpenAICompatibleRerankBaseURL(provider, baseURL string) string {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL != "" {
		return baseURL
	}
	if strings.TrimSpace(provider) == "jina" {
		return "https://api.jina.ai/v1"
	}
	return ""
}

func NewOpenAICompatibleProvider(cfg *models.APIModelConfig) (*OpenAICompatibleProvider, error) {
	baseURL := normalizeOpenAICompatibleRerankBaseURL(cfg.Provider, cfg.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("openai-compatible rerank base_url is required")
	}
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 60
	}
	modelName := strings.TrimSpace(cfg.ModelName)
	if modelName == "" {
		modelName = "local-reranker"
	}
	return &OpenAICompatibleProvider{
		apiKey:     strings.TrimSpace(cfg.APIKey),
		baseURL:    baseURL,
		modelName:  modelName,
		timeoutSec: timeout,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

func (p *OpenAICompatibleProvider) Rerank(query string, candidates []Candidate, topN int) ([]Candidate, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}
	if topN <= 0 || topN > len(candidates) {
		topN = len(candidates)
	}

	docs := make([]string, len(candidates))
	for i, c := range candidates {
		docs[i] = c.Content
	}

	reqBody := openAICompatibleRerankRequest{
		Model:     p.modelName,
		Query:     query,
		Documents: docs,
		TopN:      topN,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai-compatible rerank marshal failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.timeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/rerank", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("openai-compatible rerank create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(p.apiKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai-compatible rerank request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai-compatible rerank read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai-compatible rerank api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var rerankResp openAICompatibleRerankResponse
	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("openai-compatible rerank unmarshal failed: %w", err)
	}

	results := make([]Candidate, 0, len(rerankResp.Results))
	for _, r := range rerankResp.Results {
		if r.Index >= 0 && r.Index < len(candidates) {
			c := candidates[r.Index]
			c.Score = r.RelevanceScore
			results = append(results, c)
		}
	}
	return results, nil
}
