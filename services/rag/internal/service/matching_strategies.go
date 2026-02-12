package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// MatchResult 匹配结果
type MatchResult struct {
	Document *schema.Document
	Score    float64
	Reason   string
	Strategy string
}

// RAGMatcherConfig RAG匹配器配置
type RAGMatcherConfig struct {
	VectorSimilarityThreshold float64 // 向量相似度阈值
	VectorSearchTopK          int     // 向量搜索返回数量
	LLMMatchingEnabled        bool    // 是否启用LLM语义匹配
	LLMMatchingThreshold      float64 // LLM匹配阈值
	KeywordMatchingEnabled    bool    // 是否启用关键词匹配
}

// NewRAGMatcherConfig 创建默认配置
func NewRAGMatcherConfig() *RAGMatcherConfig {
	return &RAGMatcherConfig{
		VectorSimilarityThreshold: 0.7,
		VectorSearchTopK:          3,
		LLMMatchingEnabled:        true,
		LLMMatchingThreshold:      0.7,
		KeywordMatchingEnabled:    true,
	}
}

// MatchingStrategy 匹配策略接口
type MatchingStrategy interface {
	Name() string
	Match(ctx context.Context, query string, documents []*schema.Document) ([]*MatchResult, error)
}

// VectorSimilarityStrategy 向量相似度匹配策略
type VectorSimilarityStrategy struct {
	embedder embedding.Embedder
	config   *RAGMatcherConfig
	logger   *slog.Logger
}

// NewVectorSimilarityStrategy 创建向量相似度策略
func NewVectorSimilarityStrategy(embedder embedding.Embedder, config *RAGMatcherConfig, logger *slog.Logger) *VectorSimilarityStrategy {
	return &VectorSimilarityStrategy{
		embedder: embedder,
		config:   config,
		logger:   logger,
	}
}

func (v *VectorSimilarityStrategy) Name() string {
	return "vector"
}

func (v *VectorSimilarityStrategy) Match(ctx context.Context, query string, documents []*schema.Document) ([]*MatchResult, error) {
	v.logger.Info("开始向量相似度匹配",
		slog.String("query", query),
		slog.Int("documentCount", len(documents)),
	)

	// 获取查询向量
	v.logger.Info("开始查询向量化")
	queryEmbedding, err := v.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		v.logger.Error("查询向量化失败", slog.Any("error", err))
		return nil, fmt.Errorf("查询向量化失败: %w", err)
	}

	if len(queryEmbedding) == 0 {
		v.logger.Error("查询向量为空")
		return nil, fmt.Errorf("查询向量为空")
	}

	queryVector := queryEmbedding[0]
	v.logger.Info("查询向量化完成", slog.Int("vectorDimension", len(queryVector)))

	var results []*MatchResult
	documentProcessed := 0
	documentWithVector := 0

	// 为每个文档计算相似度
	for _, doc := range documents {
		documentProcessed++
		// 如果文档已有向量，直接使用
		var docVector []float64
		existingVector := doc.DenseVector()
		if len(existingVector) > 0 {
			docVector = existingVector
			documentWithVector++
		} else {
			// 否则生成文档向量
			v.logger.Debug("生成文档向量", slog.String("documentID", doc.ID))
			docEmbedding, err := v.embedder.EmbedStrings(ctx, []string{doc.Content})
			if err != nil {
				v.logger.Warn("文档向量化失败", slog.String("documentID", doc.ID), slog.Any("error", err))
				continue
			}
			if len(docEmbedding) > 0 {
				docVector = docEmbedding[0]
				// 缓存向量到文档中
				doc = doc.WithDenseVector(docVector)
				documentWithVector++
			}
		}

		if len(docVector) == 0 {
			continue
		}

		// 计算余弦相似度
		similarity := v.cosineSimilarity(queryVector, docVector)

		// 详细记录每个文档的相似度
		v.logger.Info("文档相似度计算",
			slog.String("documentID", doc.ID),
			slog.String("query", query),
			slog.Float64("similarity", similarity),
			slog.Bool("aboveThreshold", similarity >= v.config.VectorSimilarityThreshold),
		)

		if similarity >= v.config.VectorSimilarityThreshold {
			results = append(results, &MatchResult{
				Document: doc,
				Score:    similarity,
				Reason:   fmt.Sprintf("向量相似度: %.3f", similarity),
				Strategy: v.Name(),
			})
			v.logger.Debug("文档匹配成功",
				slog.String("documentID", doc.ID),
				slog.Float64("similarity", similarity),
			)
		} else {
			v.logger.Debug("文档未达到阈值",
				slog.String("documentID", doc.ID),
				slog.Float64("similarity", similarity),
				slog.Float64("threshold", v.config.VectorSimilarityThreshold),
			)
		}
	}

	v.logger.Info("文档处理完成",
		slog.Int("totalProcessed", documentProcessed),
		slog.Int("withVector", documentWithVector),
		slog.Int("matched", len(results)),
	)

	// 按相似度排序并限制数量
	if len(results) > v.config.VectorSearchTopK {
		v.logger.Info("排序匹配结果",
			slog.Int("beforeSort", len(results)),
			slog.Int("topK", v.config.VectorSearchTopK),
		)
		// 简单排序
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[i].Score < results[j].Score {
					results[i], results[j] = results[j], results[i]
				}
			}
		}
		results = results[:v.config.VectorSearchTopK]
		v.logger.Info("排序完成", slog.Int("afterSort", len(results)))
	}

	// 输出最终匹配结果
	for i, result := range results {
		v.logger.Info("向量匹配结果",
			slog.Int("rank", i+1),
			slog.String("documentID", result.Document.ID),
			slog.Float64("score", result.Score),
			slog.String("reason", result.Reason),
		)
	}

	return results, nil
}

// cosineSimilarity 计算余弦相似度
func (v *VectorSimilarityStrategy) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// LLMSemanticStrategy LLM语义匹配策略
type LLMSemanticStrategy struct {
	llm    model.BaseChatModel
	config *RAGMatcherConfig
	logger *slog.Logger
}

// NewLLMSemanticStrategy 创建LLM语义策略
func NewLLMSemanticStrategy(llm model.BaseChatModel, config *RAGMatcherConfig, logger *slog.Logger) *LLMSemanticStrategy {
	if logger == nil {
		logger = slog.Default().With("component", "llm_semantic_strategy")
	}
	return &LLMSemanticStrategy{
		llm:    llm,
		config: config,
		logger: logger,
	}
}

func (l *LLMSemanticStrategy) Name() string {
	return "llm"
}

func (l *LLMSemanticStrategy) Match(ctx context.Context, query string, documents []*schema.Document) ([]*MatchResult, error) {
	if !l.config.LLMMatchingEnabled {
		return nil, nil
	}

	var results []*MatchResult

	// 批量处理文档以提高效率
	batchSize := 5
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		batchResults, err := l.matchBatch(ctx, query, batch)
		if err != nil {
			continue
		}

		results = append(results, batchResults...)
	}

	return results, nil
}

// matchBatch 批量匹配文档
func (l *LLMSemanticStrategy) matchBatch(ctx context.Context, query string, documents []*schema.Document) ([]*MatchResult, error) {
	// 构建提示词
	prompt := l.buildMatchingPrompt(query, documents)

	// 调用LLM
	messages := []*schema.Message{
		schema.SystemMessage("你是一个专业的文档匹配助手，需要评估查询与文档的语义相关性。"),
		schema.UserMessage(prompt),
	}

	response, err := l.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析LLM响应
	return l.parseMatchingResponse(response.Content, documents)
}

// buildMatchingPrompt 构建匹配提示词
func (l *LLMSemanticStrategy) buildMatchingPrompt(query string, documents []*schema.Document) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("查询: %s\n\n", query))
	builder.WriteString("请评估以下文档与查询的语义相关性，为每个文档打分(0-1):\n\n")

	for i, doc := range documents {
		builder.WriteString(fmt.Sprintf("文档%d (ID: %s):\n%s\n\n", i+1, doc.ID, doc.Content))
	}

	builder.WriteString("请按以下格式返回结果:\n")
	builder.WriteString("文档ID|分数|理由\n")
	builder.WriteString("例如: doc1|0.85|查询与文档主题高度相关\n")

	return builder.String()
}

// parseMatchingResponse 解析LLM匹配响应
func (l *LLMSemanticStrategy) parseMatchingResponse(response string, documents []*schema.Document) ([]*MatchResult, error) {
	var results []*MatchResult
	docMap := make(map[string]*schema.Document)

	// 创建文档ID映射
	for _, doc := range documents {
		docMap[doc.ID] = doc
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "|") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		docID := strings.TrimSpace(parts[0])
		scoreStr := strings.TrimSpace(parts[1])
		reason := strings.TrimSpace(parts[2])

		// 解析分数
		var score float64
		if _, err := fmt.Sscanf(scoreStr, "%f", &score); err != nil {
			continue
		}

		// 检查分数阈值
		if score < l.config.LLMMatchingThreshold {
			continue
		}

		// 查找对应文档
		if doc, exists := docMap[docID]; exists {
			results = append(results, &MatchResult{
				Document: doc,
				Score:    score,
				Reason:   fmt.Sprintf("LLM语义匹配: %s", reason),
				Strategy: l.Name(),
			})
		}
	}

	return results, nil
}

// KeywordStrategy 关键词匹配策略（后备策略）
type KeywordStrategy struct {
	logger *slog.Logger
}

// NewKeywordStrategy 创建关键词策略
func NewKeywordStrategy(logger *slog.Logger) *KeywordStrategy {
	if logger == nil {
		logger = slog.Default().With("component", "keyword_strategy")
	}
	return &KeywordStrategy{
		logger: logger,
	}
}

func (k *KeywordStrategy) Name() string {
	return "keyword"
}

func (k *KeywordStrategy) Match(ctx context.Context, query string, documents []*schema.Document) ([]*MatchResult, error) {
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	var results []*MatchResult

	for _, doc := range documents {
		contentLower := strings.ToLower(doc.Content)

		// 计算关键词匹配分数
		matchCount := 0
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				matchCount++
			}
		}

		if matchCount > 0 {
			score := float64(matchCount) / float64(len(queryWords))
			results = append(results, &MatchResult{
				Document: doc,
				Score:    score,
				Reason:   fmt.Sprintf("关键词匹配: %d/%d", matchCount, len(queryWords)),
				Strategy: k.Name(),
			})
		}
	}

	return results, nil
}
