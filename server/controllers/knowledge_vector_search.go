package controllers

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils/logger"
)

func searchKnowledgeHitsByApp(ctx context.Context, appID, query string, topK int) ([]service.VectorHit, error) {
	appID = strings.TrimSpace(appID)
	query = strings.TrimSpace(query)
	if appID == "" || query == "" || store.DB == nil {
		return nil, nil
	}
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	var bases []models.KnowledgeBase
	if err := store.DB.Where("app_id = ?", appID).Order("updated_at DESC").Find(&bases).Error; err != nil {
		return nil, err
	}
	if len(bases) == 0 {
		return nil, nil
	}

	vdb := service.GetVectorStore()
	results := make([]service.VectorHit, 0, len(bases)*topK)
	var lastErr error
	successCount := 0

	for _, base := range bases {
		collection := strings.TrimSpace(base.Collection)
		if collection == "" {
			continue
		}
		hits, err := vdb.SearchText(ctx, collection, query, topK)
		if err != nil {
			lastErr = err
			logger.Warnf("knowledge vector search failed app_id=%s base_id=%d collection=%s err=%v", appID, base.ID, collection, err)
			continue
		}
		successCount++
		for _, hit := range hits {
			if hit.Metadata == nil {
				hit.Metadata = make(map[string]interface{})
			}
			hit.Metadata["app_id"] = base.AppID
			hit.Metadata["base_id"] = base.ID
			hit.Metadata["base_name"] = base.Name
			results = append(results, hit)
		}
	}

	if len(results) == 0 {
		if successCount == 0 && lastErr != nil {
			return nil, lastErr
		}
		return nil, nil
	}
	return service.RerankHits(ctx, query, results, topK), nil
}

func vectorHitsToRAGChunks(hits []service.VectorHit) []ragChunk {
	if len(hits) == 0 {
		return nil
	}
	out := make([]ragChunk, 0, len(hits))
	for _, hit := range hits {
		baseName := strings.TrimSpace(metadataString(hit.Metadata, "base_name"))
		docName := strings.TrimSpace(metadataString(hit.Metadata, "doc_name"))
		title := docName
		if title == "" {
			title = baseName
		}
		sourceName := baseName
		if sourceName == "" {
			sourceName = docName
		}
		out = append(out, ragChunk{
			Title:      title,
			Content:    strings.TrimSpace(hit.Content),
			SourceType: "vector",
			SourceName: sourceName,
			Score:      int(math.Round(hit.Score * 1000)),
		})
	}
	return out
}

func metadataString(metadata map[string]interface{}, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		if math.Mod(typed, 1) == 0 {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
