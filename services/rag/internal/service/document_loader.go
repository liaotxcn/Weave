package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
)

// DocumentLoader 文档加载器接口
type DocumentLoader interface {
	LoadDocumentFromPath(ctx context.Context, filePath string) ([]*schema.Document, error)
	LoadDocumentsFromDir(ctx context.Context, dirPath string) ([]*schema.Document, error)
}

// documentLoader 文档加载器实现
type documentLoader struct {
	logger *slog.Logger
}

// NewDocumentLoader 创建新的文档加载器
func NewDocumentLoader(logger *slog.Logger) DocumentLoader {
	return &documentLoader{
		logger: logger.With("component", "document_loader"),
	}
}

// LoadDocumentFromPath 从指定路径加载文档
func (d *documentLoader) LoadDocumentFromPath(ctx context.Context, filePath string) ([]*schema.Document, error) {

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		d.logger.Error("文件不存在", "filePath", filePath)
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		d.logger.Error("读取文件失败", "filePath", filePath, "error", err)
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 使用文本解析器解析内容
	textParser := parser.TextParser{}
	docs, err := textParser.Parse(ctx, strings.NewReader(string(content)),
		parser.WithURI(filePath),
		parser.WithExtraMeta(map[string]any{
			"source":    "local",
			"file_type": strings.ToLower(filepath.Ext(filePath)),
			"file_size": len(content),
		}),
	)
	if err != nil {
		d.logger.Error("解析文档失败", "filePath", filePath, "error", err)
		return nil, fmt.Errorf("解析文档失败: %w", err)
	}

	return docs, nil
}

// LoadDocumentsFromDir 从指定目录加载所有文档
func (d *documentLoader) LoadDocumentsFromDir(ctx context.Context, dirPath string) ([]*schema.Document, error) {

	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		d.logger.Error("目录不存在", "dirPath", dirPath)
		return nil, fmt.Errorf("目录不存在: %s", dirPath)
	}

	var allDocs []*schema.Document

	// 遍历目录
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 只处理 Markdown 文件
		if strings.HasSuffix(strings.ToLower(path), ".md") {
			docs, err := d.LoadDocumentFromPath(ctx, path)
			if err != nil {
				d.logger.Warn("加载文档失败", "path", path, "error", err)
				return nil // 继续处理其他文件
			}
			allDocs = append(allDocs, docs...)
		}

		return nil
	})

	if err != nil {
		d.logger.Error("遍历目录失败", "dirPath", dirPath, "error", err)
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}

	return allDocs, nil
}

// DocumentSplitter 文档分割器接口
type DocumentSplitter interface {
	SplitDocuments(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error)
}

// MarkdownSplitter Markdown文档分割器
type MarkdownSplitter struct {
	MaxChunkSize int // 最大块大小（字符数）
	ChunkOverlap int // 块重叠大小
	logger       *slog.Logger
}

// NewMarkdownSplitter 创建Markdown分割器
func NewMarkdownSplitter(maxChunkSize, chunkOverlap int, logger *slog.Logger) DocumentSplitter {
	if logger == nil {
		logger = slog.Default().With("component", "markdown_splitter")
	}
	return &MarkdownSplitter{
		MaxChunkSize: maxChunkSize,
		ChunkOverlap: chunkOverlap,
		logger:       logger.With("component", "markdown_splitter"),
	}
}

// SplitDocuments 分割文档
func (ms *MarkdownSplitter) SplitDocuments(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
	var result []*schema.Document

	for _, doc := range docs {
		chunks, err := ms.splitMarkdownContent(doc.Content)
		if err != nil {
			ms.logger.Error("分割文档失败", "docID", doc.ID, "error", err)
			return nil, fmt.Errorf("分割文档失败: %w", err)
		}

		for i, chunk := range chunks {
			// 创建新的文档块
			chunkDoc := &schema.Document{
				ID:      fmt.Sprintf("%s_chunk_%d", doc.ID, i),
				Content: chunk,
				MetaData: map[string]any{
					"source":       doc.MetaData["source"],
					"file_type":    doc.MetaData["file_type"],
					"chunk_index":  i,
					"total_chunks": len(chunks),
					"parent_id":    doc.ID,
				},
			}
			result = append(result, chunkDoc)
		}
	}

	return result, nil
}

// splitMarkdownContent 按章节分割Markdown内容
func (ms *MarkdownSplitter) splitMarkdownContent(content string) ([]string, error) {
	lines := strings.Split(content, "\n")
	var chunks []string
	var currentChunk strings.Builder
	var currentSize int

	for _, line := range lines {
		lineSize := len(line) + 1 // +1 for newline

		// 如果是新的章节标题（# 或 ##），并且当前块不为空，则开始新块
		if (strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ")) && currentSize > 0 {
			// 如果当前块超过最大大小，保存它
			if currentSize > ms.MaxChunkSize {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
				currentSize = 0
			}
		}

		// 如果添加这一行会超过最大大小，先保存当前块
		if currentSize+lineSize > ms.MaxChunkSize && currentSize > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
			currentSize = 0
		}

		// 添加当前行
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
		currentSize += lineSize
	}

	// 添加最后一个块
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks, nil
}

// DocumentService 文档服务
type DocumentService struct {
	loader        DocumentLoader
	splitter      DocumentSplitter
	logger        *slog.Logger
	contentFilter *ContentFilter
}

// NewDocumentService 创建文档服务
func NewDocumentService(logger *slog.Logger) *DocumentService {
	loader := NewDocumentLoader(logger)
	// 设置合理的块大小，考虑到模型token限制
	splitter := NewMarkdownSplitter(8000, 200, logger)

	return &DocumentService{
		loader:        loader,
		splitter:      splitter,
		logger:        logger.With("component", "document_service"),
		contentFilter: NewContentFilter(logger),
	}
}

// LoadAndSplitDocuments 加载并分割文档
func (ds *DocumentService) LoadAndSplitDocuments(ctx context.Context, dirPath string) ([]*schema.Document, error) {
	ds.logger.Info("开始加载文档", slog.String("directory", dirPath))

	// 加载文档
	docs, err := ds.loader.LoadDocumentsFromDir(ctx, dirPath)
	if err != nil {
		ds.logger.Error("加载文档失败", slog.Any("error", err))
		return nil, fmt.Errorf("加载文档失败: %w", err)
	}

	ds.logger.Info("文档加载完成", slog.Int("originalDocuments", len(docs)))

	// 分割文档
	chunks, err := ds.splitter.SplitDocuments(ctx, docs)
	if err != nil {
		ds.logger.Error("分割文档失败", slog.Any("error", err))
		return nil, fmt.Errorf("分割文档失败: %w", err)
	}

	ds.logger.Info("文档分割完成", slog.Int("chunks", len(chunks)))

	return chunks, nil
}

// GetRelevantChunks 获取相关的文档块
func (ds *DocumentService) GetRelevantChunks(ctx context.Context, query string, chunks []*schema.Document, ragMatcher *RAGMatcher) []*schema.Document {
	// 使用RAG匹配器获取相关文档
	results, err := ragMatcher.Match(ctx, query, chunks)
	if err != nil {
		ds.logger.Error("RAG匹配失败", slog.Any("error", err))
		// 降级到简单匹配
		return ds.fallbackMatching(query, chunks)
	}

	if len(results) == 0 {

		return ds.fallbackMatching(query, chunks)
	}

	// 提取文档
	var relevantChunks []*schema.Document
	for _, result := range results {
		relevantChunks = append(relevantChunks, result.Document)
	}

	return relevantChunks
}

// fallbackMatching 简单匹配作为后备
func (ds *DocumentService) fallbackMatching(query string, chunks []*schema.Document) []*schema.Document {
	var relevant []*schema.Document
	queryLower := strings.ToLower(query)

	for _, chunk := range chunks {
		contentLower := strings.ToLower(chunk.Content)
		if strings.Contains(contentLower, queryLower) {
			relevant = append(relevant, chunk)
			if len(relevant) >= 3 { // 限制返回数量
				break
			}
		}
	}

	if len(relevant) == 0 {

		// 返回前3个块作为通用内容
		limit := 3
		if len(chunks) < limit {
			limit = len(chunks)
		}
		return chunks[:limit]
	}

	return relevant
}
