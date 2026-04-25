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

type ollamaRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type ollamaRerankResponse struct {
	Results []struct {
		Index    int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

type OllamaProvider struct {
	baseURL    string
	modelName  string
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
		timeoutSec: timeout,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

func (p *OllamaProvider) Rerank(query string, candidates []Candidate, topN int) ([]Candidate, error) {
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

	reqBody := ollamaRerankRequest{
		Model:     p.modelName,
		Query:     query,
		Documents: docs,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama rerank marshal failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.timeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/rerank", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama rerank create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama rerank request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama rerank read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama rerank api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var rerankResp ollamaRerankResponse
	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("ollama rerank unmarshal failed: %w", err)
	}

	scored := make([]Candidate, len(candidates))
	copy(scored, candidates)
	for _, r := range rerankResp.Results {
		if r.Index >= 0 && r.Index < len(scored) {
			scored[r.Index].Score = r.RelevanceScore
		}
	}

	sorted := make([]Candidate, 0, topN)
	for i := 0; i < len(rerankResp.Results) && i < topN; i++ {
		idx := rerankResp.Results[i].Index
		if idx >= 0 && idx < len(scored) {
			sorted = append(sorted, scored[idx])
		}
	}
	return sorted, nil
}
