package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"kefu-server/models"
	"kefu-server/store"
)

var (
	markdownHeadingPattern = regexp.MustCompile(`^\s{0,3}#{1,6}\s+\S+`)
	bulletLinePattern      = regexp.MustCompile(`^\s*(?:[-*+]\s+|\d+[.)]\s+)`)
	genericHeadingPattern  = regexp.MustCompile(`^\s*(?:第[一二三四五六七八九十百千0-9]+[章节部分篇条]|[0-9]{1,3}(?:[.．][0-9]{1,3}){0,4}|[一二三四五六七八九十]+[、.．])\s*\S+`)
	shortLabelPattern      = regexp.MustCompile(`^\s*\S.{0,30}[：:]\s*$`)
	multiColumnPattern     = regexp.MustCompile(`\S(?:\s{2,}|\t)\S`)
	extraBlankLinePattern  = regexp.MustCompile(`\n{3,}`)
	multiSpacePattern      = regexp.MustCompile(`\s{2,}`)
)

// RebuildChunksFromRawContent 按文档 raw 内容重新切片入库（用于编辑/重建）。
// 说明：通过 VectorStore 抽象写入向量后端，默认实现为 sqlite 内置存储。
func RebuildChunksFromRawContent(ctx context.Context, vc VectorStore, doc *models.KnowledgeDocument) (int, error) {
	if vc == nil || doc == nil {
		return 0, fmt.Errorf("invalid rebuild input")
	}
	if strings.TrimSpace(doc.RawContent) == "" {
		return 0, nil
	}

	var existed []models.KnowledgeChunk
	if err := store.DB.Where("document_id = ?", doc.ID).Find(&existed).Error; err != nil {
		return 0, fmt.Errorf("load existing chunks failed: %w", err)
	}
	for _, chunk := range existed {
		_ = vc.DeleteVector(ctx, doc.VectorCollection, chunk.VectorID)
	}
	if err := store.DB.Unscoped().Where("document_id = ?", doc.ID).Delete(&models.KnowledgeChunk{}).Error; err != nil {
		return 0, fmt.Errorf("delete old chunks failed: %w", err)
	}

	parts := splitTextForKnowledge(strings.TrimSpace(doc.RawContent), 900, 120)
	now := time.Now()
	created := 0
	for idx, part := range parts {
		vectorID := fmt.Sprintf("kb_%d_doc_%d_chunk_%d_%d", doc.BaseID, doc.ID, idx+1, now.UnixNano())
		meta := map[string]interface{}{
			"app_id":      doc.AppID,
			"base_id":     doc.BaseID,
			"document_id": doc.ID,
			"chunk_seq":   idx + 1,
			"doc_name":    doc.Name,
		}
		if err := vc.InsertText(ctx, doc.VectorCollection, vectorID, part, meta); err != nil {
			return created, err
		}
		row := models.KnowledgeChunk{
			BaseID:       doc.BaseID,
			DocumentID:   doc.ID,
			AppID:        doc.AppID,
			VectorID:     vectorID,
			ChunkSeq:     idx + 1,
			Content:      part,
			ContentChars: len([]rune(part)),
		}
		if err := store.DB.Create(&row).Error; err != nil {
			return created, fmt.Errorf("save chunk row failed: %w", err)
		}
		created++
	}

	doc.ChunkCount = created
	doc.Status = "indexed"
	doc.ErrorMessage = ""
	doc.LastIndexedAt = &now
	if err := store.DB.Save(doc).Error; err != nil {
		return created, fmt.Errorf("update document index result failed: %w", err)
	}
	return created, nil
}

func splitTextForKnowledge(text string, chunkSize, overlap int) []string {
	clean := normalizeKnowledgeText(text)
	if clean == "" {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 900
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 4
	}

	sections := splitKnowledgeSections(clean)
	chunks := make([]string, 0, len(sections))
	for _, section := range sections {
		chunks = append(chunks, splitSectionToChunks(section, chunkSize, overlap)...)
	}
	if len(chunks) == 0 {
		return nil
	}
	return rebalanceKnowledgeChunks(dedupeKnowledgeChunks(chunks), chunkSize)
}

func normalizeKnowledgeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.ReplaceAll(text, "\t", "    ")

	lines := strings.Split(text, "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		normalized = append(normalized, strings.TrimRight(line, " \t"))
	}
	text = strings.Join(normalized, "\n")
	text = extraBlankLinePattern.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func splitKnowledgeSections(text string) []string {
	lines := strings.Split(text, "\n")
	sections := make([]string, 0)
	var current []string

	flush := func() {
		if len(current) == 0 {
			return
		}
		section := normalizeKnowledgeText(strings.Join(current, "\n"))
		if section != "" {
			sections = append(sections, section)
		}
		current = nil
	}

	hasStructuredHeadings := markdownHeadingPattern.MatchString(text) || genericHeadingPattern.MatchString(text)
	if hasStructuredHeadings {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if looksHeadingLine(trimmed) {
				flush()
				current = append(current, trimmed)
				continue
			}
			current = append(current, line)
		}
		flush()
		return sections
	}

	paragraphs := splitParagraphs(lines)
	for _, paragraph := range paragraphs {
		if paragraph != "" {
			sections = append(sections, paragraph)
		}
	}
	return sections
}

func splitParagraphs(lines []string) []string {
	parts := make([]string, 0)
	var paragraph []string
	var listBlock []string
	var tableBlock []string

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		block := collapseParagraphLines(paragraph)
		if block != "" {
			parts = append(parts, block)
		}
		paragraph = nil
	}
	flushList := func() {
		if len(listBlock) == 0 {
			return
		}
		block := normalizeKnowledgeText(strings.Join(listBlock, "\n"))
		if block != "" {
			parts = append(parts, block)
		}
		listBlock = nil
	}
	flushTable := func() {
		if len(tableBlock) == 0 {
			return
		}
		block := normalizeKnowledgeText(strings.Join(tableBlock, "\n"))
		if block != "" {
			parts = append(parts, block)
		}
		tableBlock = nil
	}
	flushAll := func() {
		flushParagraph()
		flushList()
		flushTable()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flushAll()
			continue
		}
		if looksHeadingLine(trimmed) {
			flushAll()
			parts = append(parts, trimmed)
			continue
		}
		if looksTableLine(line) {
			flushParagraph()
			flushList()
			tableBlock = append(tableBlock, normalizeTableLine(line))
			continue
		}
		flushTable()
		if bulletLinePattern.MatchString(trimmed) {
			flushParagraph()
			listBlock = append(listBlock, trimmed)
			continue
		}
		flushList()
		paragraph = append(paragraph, trimmed)
	}
	flushAll()
	return parts
}

func splitSectionToChunks(section string, chunkSize, overlap int) []string {
	section = normalizeKnowledgeText(section)
	if section == "" {
		return nil
	}
	if runeCount(section) <= chunkSize {
		return []string{section}
	}

	units := splitSectionUnits(section)
	if len(units) == 0 {
		return nil
	}
	if len(units) == 1 && runeCount(units[0]) > chunkSize {
		return hardSplitChunk(units[0], chunkSize, overlap)
	}

	result := make([]string, 0)
	current := ""
	for _, unit := range units {
		unit = normalizeKnowledgeText(unit)
		if unit == "" {
			continue
		}
		if runeCount(unit) > chunkSize {
			if strings.TrimSpace(current) != "" {
				result = append(result, strings.TrimSpace(current))
				current = ""
			}
			result = append(result, hardSplitChunk(unit, chunkSize, overlap)...)
			continue
		}
		if current == "" {
			current = unit
			continue
		}
		candidate := current + "\n\n" + unit
		if runeCount(candidate) <= chunkSize {
			current = candidate
			continue
		}
		result = append(result, strings.TrimSpace(current))
		prefix := overlapTail(current, overlap)
		if prefix != "" {
			current = strings.TrimSpace(prefix + "\n" + unit)
		} else {
			current = unit
		}
	}
	if strings.TrimSpace(current) != "" {
		result = append(result, strings.TrimSpace(current))
	}
	return result
}

func splitSectionUnits(section string) []string {
	if looksHeadingLine(firstNonEmptyLine(section)) {
		lines := strings.Split(section, "\n")
		if len(lines) == 1 {
			return []string{section}
		}
		heading := strings.TrimSpace(lines[0])
		body := normalizeKnowledgeText(strings.Join(lines[1:], "\n"))
		if body == "" {
			return []string{heading}
		}
		bodyUnits := splitParagraphs(strings.Split(body, "\n"))
		units := make([]string, 0, len(bodyUnits))
		for i, unit := range bodyUnits {
			if i == 0 {
				units = append(units, heading+"\n"+unit)
			} else {
				units = append(units, unit)
			}
		}
		return units
	}

	paragraphs := splitParagraphs(strings.Split(section, "\n"))
	if len(paragraphs) > 1 {
		return paragraphs
	}
	sentences := splitSentences(section)
	if len(sentences) > 0 {
		return sentences
	}
	return []string{section}
}

func splitSentences(text string) []string {
	text = normalizeKnowledgeText(text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	parts := make([]string, 0)
	start := 0
	for i, r := range runes {
		if strings.ContainsRune("。！？!?；;.\n", r) {
			part := strings.TrimSpace(string(runes[start : i+1]))
			if part != "" {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	if start < len(runes) {
		part := strings.TrimSpace(string(runes[start:]))
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func hardSplitChunk(text string, chunkSize, overlap int) []string {
	text = normalizeKnowledgeText(text)
	if text == "" {
		return nil
	}
	runes := []rune(text)
	if len(runes) <= chunkSize {
		return []string{text}
	}
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	result := make([]string, 0, len(runes)/step+1)
	for i := 0; i < len(runes); i += step {
		end := findChunkBoundary(runes, i, chunkSize)
		part := strings.TrimSpace(string(runes[i:end]))
		if part != "" {
			result = append(result, part)
		}
		if end >= len(runes) {
			break
		}
	}
	return result
}

func overlapTail(text string, overlap int) string {
	if overlap <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= overlap {
		return string(runes)
	}
	start := len(runes) - overlap
	if start < 0 {
		start = 0
	}
	for start < len(runes)-1 && !looksNaturalBoundary(runes[start]) {
		start++
	}
	return strings.TrimSpace(string(runes[start:]))
}

func runeCount(text string) int {
	return len([]rune(strings.TrimSpace(text)))
}

func dedupeKnowledgeChunks(chunks []string) []string {
	seen := make(map[string]struct{}, len(chunks))
	result := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = normalizeKnowledgeText(chunk)
		if chunk == "" {
			continue
		}
		if _, ok := seen[chunk]; ok {
			continue
		}
		seen[chunk] = struct{}{}
		result = append(result, chunk)
	}
	return result
}

func looksHeadingLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	return markdownHeadingPattern.MatchString(line) || genericHeadingPattern.MatchString(line) || shortLabelPattern.MatchString(line)
}

func looksTableLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.Count(trimmed, "\t") >= 1 {
		return true
	}
	if strings.Count(trimmed, "|") >= 2 {
		return true
	}
	return multiColumnPattern.MatchString(trimmed)
}

func normalizeTableLine(line string) string {
	line = strings.ReplaceAll(line, "\t", " | ")
	line = multiSpacePattern.ReplaceAllString(strings.TrimSpace(line), " | ")
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	return "| " + line + " |"
}

func collapseParagraphLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	merged := strings.TrimSpace(lines[0])
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		merged += joinParagraphLine(merged, line)
	}
	return normalizeKnowledgeText(merged)
}

func joinParagraphLine(prev, next string) string {
	if prev == "" {
		return next
	}
	if next == "" {
		return ""
	}
	prevRune := lastRune(prev)
	nextRune := firstRune(next)
	if isASCIIWordRune(prevRune) && isASCIIWordRune(nextRune) {
		return " " + next
	}
	if looksNaturalBoundary(prevRune) {
		return "\n" + next
	}
	return next
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func lastRune(text string) rune {
	rs := []rune(strings.TrimSpace(text))
	if len(rs) == 0 {
		return 0
	}
	return rs[len(rs)-1]
}

func firstRune(text string) rune {
	rs := []rune(strings.TrimSpace(text))
	if len(rs) == 0 {
		return 0
	}
	return rs[0]
}

func isASCIIWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func looksNaturalBoundary(r rune) bool {
	return strings.ContainsRune("。！？!?；;：:，,、.)]）】}>", r)
}

func findChunkBoundary(runes []rune, start, chunkSize int) int {
	end := start + chunkSize
	if end >= len(runes) {
		return len(runes)
	}
	searchLeft := end - minInt(80, chunkSize/3)
	if searchLeft < start {
		searchLeft = start
	}
	for i := end; i >= searchLeft; i-- {
		if i > start && looksNaturalBoundary(runes[i-1]) {
			return i
		}
		if i > start && runes[i-1] == '\n' {
			return i
		}
	}
	return end
}

func rebalanceKnowledgeChunks(chunks []string, chunkSize int) []string {
	if len(chunks) == 0 {
		return nil
	}
	result := make([]string, 0, len(chunks))
	for i := 0; i < len(chunks); i++ {
		current := normalizeKnowledgeText(chunks[i])
		if current == "" {
			continue
		}
		if i+1 < len(chunks) && looksHeadingLine(current) {
			next := normalizeKnowledgeText(chunks[i+1])
			if next != "" && runeCount(current)+2+runeCount(next) <= chunkSize {
				result = append(result, current+"\n\n"+next)
				i++
				continue
			}
		}
		if len(result) > 0 && runeCount(current) <= maxInt(120, chunkSize/5) {
			prev := result[len(result)-1]
			if runeCount(prev)+2+runeCount(current) <= chunkSize {
				result[len(result)-1] = prev + "\n\n" + current
				continue
			}
		}
		result = append(result, current)
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
