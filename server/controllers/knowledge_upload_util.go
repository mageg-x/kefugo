package controllers

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"kefu-server/config"
)

var knowledgeStripHTMLRegex = regexp.MustCompile(`<[^>]+>`)

type xlsxSharedStrings struct {
	SI []xlsxSharedStringItem `xml:"si"`
}

type xlsxTextRun struct {
	T string `xml:"t"`
}

type xlsxSharedStringItem struct {
	T string `xml:"t"`
	R []xlsxTextRun `xml:"r"`
}

type xlsxWorksheet struct {
	Rows []xlsxRow `xml:"sheetData>row"`
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Type      string `xml:"t,attr"`
	Value     string `xml:"v"`
	InlineStr struct {
		T string `xml:"t"`
		R []xlsxTextRun `xml:"r"`
	} `xml:"is"`
}

func stripKnowledgeHTML(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return strings.TrimSpace(knowledgeStripHTMLRegex.ReplaceAllString(s, " "))
}

func joinXLSXText(t string, runs []xlsxTextRun) string {
	if strings.TrimSpace(t) != "" {
		return t
	}
	if len(runs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, run := range runs {
		b.WriteString(run.T)
	}
	return b.String()
}

func extractXLSXContent(fullPath string) (string, error) {
	reader, err := zip.OpenReader(fullPath)
	if err != nil {
		return "", fmt.Errorf("read xlsx failed")
	}
	defer reader.Close()

	fileMap := make(map[string]*zip.File, len(reader.File))
	sheetNames := make([]string, 0)
	for _, file := range reader.File {
		fileMap[file.Name] = file
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
			sheetNames = append(sheetNames, file.Name)
		}
	}
	sort.Strings(sheetNames)

	sharedStrings := make([]string, 0)
	if file, ok := fileMap["xl/sharedStrings.xml"]; ok {
		rc, openErr := file.Open()
		if openErr != nil {
			return "", fmt.Errorf("open xlsx shared strings failed")
		}
		var shared xlsxSharedStrings
		decodeErr := xml.NewDecoder(rc).Decode(&shared)
		_ = rc.Close()
		if decodeErr != nil {
			return "", fmt.Errorf("parse xlsx shared strings failed")
		}
		for _, item := range shared.SI {
			sharedStrings = append(sharedStrings, joinXLSXText(item.T, item.R))
		}
	}

	parts := make([]string, 0, len(sheetNames))
	for _, sheetName := range sheetNames {
		file := fileMap[sheetName]
		rc, openErr := file.Open()
		if openErr != nil {
			return "", fmt.Errorf("open xlsx worksheet failed")
		}
		var sheet xlsxWorksheet
		decodeErr := xml.NewDecoder(rc).Decode(&sheet)
		_ = rc.Close()
		if decodeErr != nil {
			return "", fmt.Errorf("parse xlsx worksheet failed")
		}

		lines := make([]string, 0, len(sheet.Rows))
		for _, row := range sheet.Rows {
			cells := make([]string, 0, len(row.Cells))
			for _, cell := range row.Cells {
				value := ""
				switch strings.ToLower(strings.TrimSpace(cell.Type)) {
				case "s":
					idx := 0
					_, _ = fmt.Sscanf(strings.TrimSpace(cell.Value), "%d", &idx)
					if idx >= 0 && idx < len(sharedStrings) {
						value = sharedStrings[idx]
					}
				case "inlineStr":
					value = joinXLSXText(cell.InlineStr.T, cell.InlineStr.R)
				default:
					value = cell.Value
				}
				value = strings.TrimSpace(value)
				if value != "" {
					cells = append(cells, value)
				}
			}
			if len(cells) > 0 {
				lines = append(lines, strings.Join(cells, "\t"))
			}
		}
		if len(lines) > 0 {
			parts = append(parts, strings.Join(lines, "\n"))
		}
	}

	content := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if content == "" {
		return "", fmt.Errorf("xlsx content is empty")
	}
	return content, nil
}

func runTextCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s parse failed", filepath.Base(name))
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return "", fmt.Errorf("%s content is empty", filepath.Base(name))
	}
	return text, nil
}

func extractPDFContent(fullPath string) (string, error) {
	return runTextCommand("pdftotext", "-enc", "UTF-8", "-layout", "-nopgbrk", fullPath, "-")
}

func extractDOCXContentWithPandoc(fullPath string) (string, error) {
	return runTextCommand("pandoc", "-f", "docx", "-t", "gfm", "--wrap=none", fullPath)
}

func extractDOCXContentFallback(fullPath string) (string, error) {
	reader, err := zip.OpenReader(fullPath)
	if err != nil {
		return "", fmt.Errorf("read docx failed")
	}
	defer reader.Close()

	var documentFile *zip.File
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			documentFile = file
			break
		}
	}
	if documentFile == nil {
		return "", fmt.Errorf("docx document.xml not found")
	}
	rc, err := documentFile.Open()
	if err != nil {
		return "", fmt.Errorf("open docx document failed")
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var b strings.Builder
	lastWasBreak := true
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("parse docx document failed")
		}
		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "tab":
				b.WriteString("\t")
				lastWasBreak = false
			case "br", "cr":
				b.WriteString("\n")
				lastWasBreak = true
			case "p":
				if !lastWasBreak && b.Len() > 0 {
					b.WriteString("\n")
				}
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "p":
				if !lastWasBreak {
					b.WriteString("\n\n")
					lastWasBreak = true
				}
			case "tr":
				if !lastWasBreak {
					b.WriteString("\n")
					lastWasBreak = true
				}
			case "tc":
				if !lastWasBreak {
					b.WriteString("\t")
					lastWasBreak = false
				}
			}
		case xml.CharData:
			text := string(tok)
			if strings.TrimSpace(text) == "" {
				continue
			}
			b.WriteString(text)
			lastWasBreak = false
		}
	}
	content := strings.TrimSpace(b.String())
	if content == "" {
		return "", fmt.Errorf("docx content is empty")
	}
	return content, nil
}

func extractDOCXContent(fullPath string) (string, error) {
	content, err := extractDOCXContentWithPandoc(fullPath)
	if err == nil && strings.TrimSpace(content) != "" {
		return content, nil
	}
	return extractDOCXContentFallback(fullPath)
}

func extractKnowledgeContentFromURL(fileURL string, name string) (content string, sourceType string, sourceName string, err error) {
	cfg := config.GetConfig()
	uploadDir := ""
	if cfg != nil {
		uploadDir = strings.TrimSpace(cfg.Admin.UploadDir)
	}
	if uploadDir == "" {
		uploadDir = "data/uploads"
	}
	if !strings.HasPrefix(fileURL, "/uploads/") {
		return "", "", "", fmt.Errorf("unsupported file url")
	}
	fileName := strings.TrimPrefix(fileURL, "/uploads/")
	fileName = filepath.Base(fileName)
	if strings.TrimSpace(fileName) == "" {
		return "", "", "", fmt.Errorf("invalid file url")
	}
	fullPath := filepath.Join(uploadDir, fileName)
	body, readErr := os.ReadFile(fullPath)
	if readErr != nil {
		return "", "", "", fmt.Errorf("read upload file failed")
	}

	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	sourceType = "upload"
	sourceName = strings.TrimSpace(name)
	if sourceName == "" {
		sourceName = fileName
	}

	switch ext {
	case ".txt", ".md", ".csv", ".tsv", ".json", ".log", ".html", ".htm":
		t := string(body)
		if !utf8.ValidString(t) {
			return "", "", "", fmt.Errorf("file encoding is not utf-8")
		}
		if ext == ".html" || ext == ".htm" {
			t = stripKnowledgeHTML(t)
		}
		content = strings.TrimSpace(t)
	case ".xlsx":
		content, err = extractXLSXContent(fullPath)
		if err != nil {
			return "", "", "", err
		}
	case ".pdf":
		content, err = extractPDFContent(fullPath)
		if err != nil {
			return "", "", "", err
		}
	case ".docx":
		content, err = extractDOCXContent(fullPath)
		if err != nil {
			return "", "", "", err
		}
	case ".doc":
		return "", "", "", fmt.Errorf("legacy .doc is not supported, please convert to .docx")
	default:
		t := string(body)
		if !utf8.ValidString(t) {
			return "", "", "", fmt.Errorf("file type not supported for text parsing")
		}
		content = strings.TrimSpace(t)
	}
	if len([]rune(content)) > 30000 {
		content = string([]rune(content)[:30000])
	}
	return content, sourceType, sourceName, nil
}
