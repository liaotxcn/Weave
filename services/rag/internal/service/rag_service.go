package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"weave/services/aichat/pkg"
	"weave/services/rag/internal/cache"
)

// RAGService RAG服务
type RAGService struct {
	documentService *DocumentService
	rragMatcher     *RAGMatcher
	config          *RAGServiceConfig
	logger          *slog.Logger
	redisClient     *cache.RedisCache
	sensitiveFilter *pkg.SensitiveFilter
}

// RAGServiceConfig RAG服务配置
type RAGServiceConfig struct {
	DocumentDir         string  // 文档目录
	MaxChunkSize        int     // 最大块大小
	ChunkOverlap        int     // 块重叠大小
	TopKResults         int     // 返回结果数量
	SimilarityThreshold float64 // 相似度阈值
}

// NewRAGServiceConfig 创建默认配置
func NewRAGServiceConfig() *RAGServiceConfig {
	return &RAGServiceConfig{
		DocumentDir:         ".",
		MaxChunkSize:        8000,
		ChunkOverlap:        200,
		TopKResults:         3,
		SimilarityThreshold: 0.2,
	}
}

// NewRAGService 创建RAG服务
func NewRAGService(
	embedder embedding.Embedder,
	llm model.BaseChatModel,
	config *RAGServiceConfig,
	logger *slog.Logger,
) *RAGService {
	if config == nil {
		config = NewRAGServiceConfig()
	}

	if logger == nil {
		logger = slog.Default().With("component", "rag_service")
	}

	documentService := NewDocumentService(logger)
	ragMatcherConfig := NewRAGMatcherConfig()
	ragMatcher := NewRAGMatcher(embedder, llm, ragMatcherConfig, logger)

	// 初始化Redis缓存
	var redisClient *cache.RedisCache
	var err error
	ctx := context.Background()
	redisClient, err = cache.NewRedisCache(ctx, logger)
	if err != nil {
		logger.Warn("Redis缓存初始化失败，将使用无缓存模式", slog.Any("error", err))
	}

	// 复用 AIChat 安全过滤器
	sensitiveFilter := pkg.NewSensitiveFilter()

	return &RAGService{
		documentService: documentService,
		rragMatcher:     ragMatcher,
		config:          config,
		logger:          logger.With("component", "rag_service"),
		redisClient:     redisClient,
		sensitiveFilter: sensitiveFilter,
	}
}

// Initialize 初始化服务
func (rs *RAGService) Initialize(ctx context.Context) error {
	// 检查文档目录
	if _, err := os.Stat(rs.config.DocumentDir); os.IsNotExist(err) {
		return fmt.Errorf("文档目录不存在: %s", rs.config.DocumentDir)
	}

	// 验证目录权限
	if _, err := os.Open(rs.config.DocumentDir); err != nil {
		return fmt.Errorf("无法访问文档目录: %w", err)
	}

	return nil
}

// Query 执行查询
func (rs *RAGService) Query(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("查询内容为空")
	}

	if rs.sensitiveFilter != nil {
		rs.logger.Debug("执行安全检查", slog.String("query", query))

		if rs.sensitiveFilter.ContainsSensitiveContent(query) ||
			rs.sensitiveFilter.ContainsMaliciousInput(query) ||
			rs.sensitiveFilter.ContainsInjectionPattern(query) {
			rs.logger.Warn("查询包含敏感内容", slog.String("query", query))
			return "", fmt.Errorf("查询包含敏感内容")
		}

		rs.logger.Debug("安全检查通过", slog.String("query", query))
	}

	// 从缓存获取查询结果（安全检查通过后）
	if rs.redisClient != nil {
		if cachedResult, err := rs.redisClient.GetQueryResult(ctx, query); err == nil {
			rs.logger.Debug("从缓存获取查询结果")
			return cachedResult, nil
		}
	}

	// 尝试从缓存获取文档块
	var chunks []*schema.Document
	var err error
	if rs.redisClient != nil {
		chunks, err = rs.redisClient.GetDocumentChunks(ctx, rs.config.DocumentDir)
		if err == nil {
			rs.logger.Debug("从缓存获取文档块")
		}
	}

	// 如果缓存中没有，加载并分割文档
	if chunks == nil || len(chunks) == 0 {
		chunks, err = rs.documentService.LoadAndSplitDocuments(ctx, rs.config.DocumentDir)
		if err != nil {
			rs.logger.Error("加载文档失败", slog.Any("error", err))
			return "", fmt.Errorf("加载文档失败: %w", err)
		}

		if len(chunks) == 0 {
			return "", fmt.Errorf("未加载到任何文档")
		}

		// 将文档块存入缓存
		if rs.redisClient != nil {
			if err := rs.redisClient.SetDocumentChunks(ctx, rs.config.DocumentDir, chunks); err != nil {
				rs.logger.Warn("文档块缓存失败", slog.Any("error", err))
			}
		}
	}

	// 获取相关文档块
	relevantChunks := rs.documentService.GetRelevantChunks(ctx, query, chunks, rs.rragMatcher)

	if len(relevantChunks) == 0 {
		return "", fmt.Errorf("未找到相关文档")
	}

	// 生成最终结果
	result := rs.generateResult(query, relevantChunks)

	// 过滤生成结果中的敏感内容
	if rs.sensitiveFilter != nil {
		if rs.sensitiveFilter.ContainsSensitiveContent(result) {
			return "", fmt.Errorf("生成结果包含敏感内容")
		}
	}

	// 将查询结果存入缓存
	if rs.redisClient != nil {
		if err := rs.redisClient.SetQueryResult(ctx, query, result); err != nil {
			rs.logger.Warn("查询结果缓存失败", slog.Any("error", err))
		}
	}

	return result, nil
}

// generateResult 生成最终结果
func (rs *RAGService) generateResult(query string, chunks []*schema.Document) string {
	var builder strings.Builder

	// 添加查询信息
	builder.WriteString(fmt.Sprintf("# 查询: %s\n\n", query))
	builder.WriteString("## 相关文档\n\n")

	// 添加相关文档内容
	for i, chunk := range chunks {
		if i > 0 {
			builder.WriteString("\n---\n\n")
		}

		// 添加文档元信息
		source := "未知"
		if src, ok := chunk.MetaData["source"].(string); ok {
			source = src
		}

		fileType := "未知"
		if ft, ok := chunk.MetaData["file_type"].(string); ok {
			fileType = ft
		}

		builder.WriteString(fmt.Sprintf("### 文档 %d\n", i+1))
		builder.WriteString(fmt.Sprintf("**来源**: %s\n", source))
		builder.WriteString(fmt.Sprintf("**类型**: %s\n", fileType))
		builder.WriteString("\n")

		// 添加文档内容
		builder.WriteString(chunk.Content)
		builder.WriteString("\n")
	}

	return builder.String()
}

// LoadDocuments 加载文档
func (rs *RAGService) LoadDocuments(ctx context.Context, dirPath string) ([]*schema.Document, error) {
	if dirPath == "" {
		dirPath = rs.config.DocumentDir
	}

	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("目录不存在: %s", dirPath)
	}

	// 加载并分割文档
	chunks, err := rs.documentService.LoadAndSplitDocuments(ctx, dirPath)
	if err != nil {
		return nil, fmt.Errorf("加载文档失败: %w", err)
	}

	return chunks, nil
}

// GetDocumentStats 获取文档统计信息
func (rs *RAGService) GetDocumentStats(ctx context.Context, dirPath string) (map[string]any, error) {
	if dirPath == "" {
		dirPath = rs.config.DocumentDir
	}

	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("目录不存在: %s", dirPath)
	}

	// 遍历目录统计文档
	var totalFiles int
	var totalSize int64
	var fileTypes map[string]int = make(map[string]int)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
			totalFiles++
			totalSize += info.Size()
			fileExt := filepath.Ext(path)
			fileTypes[strings.ToLower(fileExt)]++
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}

	stats := map[string]any{
		"directory":    dirPath,
		"totalFiles":   totalFiles,
		"totalSize":    totalSize,
		"fileTypes":    fileTypes,
		"maxChunkSize": rs.config.MaxChunkSize,
		"chunkOverlap": rs.config.ChunkOverlap,
	}

	return stats, nil
}

// HealthCheck 健康检查
func (rs *RAGService) HealthCheck(ctx context.Context) (bool, map[string]any, error) {
	// 检查文档目录
	dirExists := false
	if _, err := os.Stat(rs.config.DocumentDir); err == nil {
		dirExists = true
	}

	// 获取文档统计信息
	stats, err := rs.GetDocumentStats(ctx, rs.config.DocumentDir)
	if err != nil {
		stats = map[string]any{
			"error": err.Error(),
		}
	}

	// 检查服务状态
	status := map[string]any{
		"documentDir": dirExists,
		"stats":       stats,
		"config": map[string]any{
			"maxChunkSize":        rs.config.MaxChunkSize,
			"chunkOverlap":        rs.config.ChunkOverlap,
			"topKResults":         rs.config.TopKResults,
			"similarityThreshold": rs.config.SimilarityThreshold,
		},
	}

	// 服务是否健康
	healthy := dirExists

	return healthy, status, nil
}

// Close 关闭服务
func (rs *RAGService) Close(ctx context.Context) error {
	// 关闭Redis缓存连接
	if rs.redisClient != nil {
		if err := rs.redisClient.Close(ctx); err != nil {
			rs.logger.Warn("Redis缓存关闭失败", slog.Any("error", err))
		}
	}
	return nil
}

// ModelScopeEmbedder 使用 ModelScope API 生成嵌入客户端
type ModelScopeEmbedder struct {
	apiKey           string
	embedModel       string
	baseURL          string
	client           *http.Client
	logger           *slog.Logger
	queryInstruction string
}

// DefaultQueryInstruction 默认查询指令
const DefaultQueryInstruction = "为这个句子生成表示以用于检索相关文章："

// NewModelScopeEmbedder 创建新的 ModelScope 嵌入客户端
func NewModelScopeEmbedder(apiKey, embedModel, baseURL string) embedding.Embedder {
	logger := slog.Default().With("component", "modelscope_embedder")
	return &ModelScopeEmbedder{
		apiKey:           apiKey,
		embedModel:       embedModel,
		baseURL:          baseURL,
		client:           &http.Client{},
		logger:           logger,
		queryInstruction: DefaultQueryInstruction,
	}
}

// EmbedStrings 生成文本嵌入（实现 embedding.Embedder 接口）
func (me *ModelScopeEmbedder) EmbedStrings(ctx context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
	var allEmbeddings [][]float64
	for _, text := range texts {
		embedding, err := me.embedSingle(ctx, text)
		if err != nil {
			me.logger.Error("文本向量化失败", slog.String("text", text), slog.Any("error", err))
			return nil, fmt.Errorf("文本向量化失败: %w", err)
		}
		allEmbeddings = append(allEmbeddings, embedding)
	}
	return allEmbeddings, nil
}

// embedSingle 生成单个文本的嵌入
func (me *ModelScopeEmbedder) embedSingle(ctx context.Context, text string) ([]float64, error) {
	// 添加查询指令以提高检索效果
	enhancedText := me.queryInstruction + text

	// ModelScope 嵌入 API 请求（使用 input 参数，符合嵌入API标准格式）
	reqBody, err := json.Marshal(map[string]interface{}{
		"model": me.embedModel,
		"input": enhancedText, // 使用增强后的文本（包含查询指令）
	})
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", me.baseURL+"/embeddings", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+me.apiKey)

	resp, err := me.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 请求失败: %s, 状态码: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("嵌入结果为空")
	}

	return response.Data[0].Embedding, nil
}

// OllamaEmbedder 使用 Ollama API 生成嵌入客户端
type OllamaEmbedder struct {
	baseURL    string
	embedModel string
	client     *http.Client
	logger     *slog.Logger
}

// NewOllamaEmbedder 创建新的 Ollama 嵌入客户端
func NewOllamaEmbedder(baseURL, embedModel string) embedding.Embedder {
	logger := slog.Default().With("component", "ollama_embedder")
	return &OllamaEmbedder{
		baseURL:    baseURL,
		embedModel: embedModel,
		client:     &http.Client{},
		logger:     logger,
	}
}

// EmbedStrings 生成文本嵌入
func (oe *OllamaEmbedder) EmbedStrings(ctx context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
	var allEmbeddings [][]float64
	for _, text := range texts {
		embedding, err := oe.embedSingle(ctx, text)
		if err != nil {
			oe.logger.Error("文本向量化失败", slog.String("text", text), slog.Any("error", err))
			return nil, fmt.Errorf("文本向量化失败: %w", err)
		}
		allEmbeddings = append(allEmbeddings, embedding)
	}
	return allEmbeddings, nil
}

// embedSingle 生成单个文本的嵌入
func (oe *OllamaEmbedder) embedSingle(ctx context.Context, text string) ([]float64, error) {
	// Ollama 嵌入 API 请求
	reqBody, err := json.Marshal(map[string]string{
		"model":  oe.embedModel,
		"prompt": text,
	})
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", oe.baseURL+"/api/embeddings", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := oe.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 请求失败: %s, 状态码: %d", string(body), resp.StatusCode)
	}

	var response struct {
		Embedding []float64 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return response.Embedding, nil
}
