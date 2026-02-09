package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// RAGMatcher RAG匹配器
type RAGMatcher struct {
	vectorStrategy  *VectorSimilarityStrategy
	llmStrategy     *LLMSemanticStrategy
	keywordStrategy *KeywordStrategy
	config          *RAGMatcherConfig
	logger          *slog.Logger
}

// NewRAGMatcher 创建RAG匹配器
func NewRAGMatcher(
	embedder embedding.Embedder,
	llm model.BaseChatModel,
	config *RAGMatcherConfig,
	logger *slog.Logger,
) *RAGMatcher {
	if config == nil {
		config = NewRAGMatcherConfig()
	}

	if logger == nil {
		logger = slog.Default().With("component", "rag_matcher")
	}

	return &RAGMatcher{
		vectorStrategy:  NewVectorSimilarityStrategy(embedder, config, logger),
		llmStrategy:     NewLLMSemanticStrategy(llm, config, logger),
		keywordStrategy: NewKeywordStrategy(logger),
		config:          config,
		logger:          logger.With("component", "rag_matcher"),
	}
}

// Match 执行匹配
func (rm *RAGMatcher) Match(ctx context.Context, query string, documents []*schema.Document) ([]*MatchResult, error) {
	if len(documents) == 0 {
		return nil, fmt.Errorf("文档集合为空")
	}

	// 执行向量相似度匹配
	vectorResults, _ := rm.vectorStrategy.Match(ctx, query, documents)

	// 执行LLM语义匹配
	llmResults, _ := rm.llmStrategy.Match(ctx, query, documents)

	// 执行关键词匹配
	keywordResults, _ := rm.keywordStrategy.Match(ctx, query, documents)

	// 融合结果
	fusedResults := rm.fuseResults(vectorResults, llmResults, keywordResults)

	// 排序并限制结果数量
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	// 限制返回数量
	maxResults := 3
	if len(fusedResults) > maxResults {
		fusedResults = fusedResults[:maxResults]
	}

	return fusedResults, nil
}

// fuseResults 融合不同策略的匹配结果
func (rm *RAGMatcher) fuseResults(vectorResults, llmResults, keywordResults []*MatchResult) []*MatchResult {
	// 使用map去重，键为文档ID
	resultMap := make(map[string]*MatchResult)

	// 添加向量匹配结果（权重最高）
	for _, result := range vectorResults {
		// 严格过滤：只有相似度高于阈值的结果才考虑
		if result.Score >= rm.config.VectorSimilarityThreshold {
			result.Score *= 0.6 // 向量匹配权重
			resultMap[result.Document.ID] = result
		}
	}

	// 添加LLM语义匹配结果
	for _, result := range llmResults {
		// 严格过滤：只有相似度高于阈值的结果才考虑
		if result.Score >= rm.config.LLMMatchingThreshold {
			if existing, ok := resultMap[result.Document.ID]; ok {
				// 如果文档已存在，融合分数
				existing.Score += result.Score * 0.3 // LLM匹配权重
				existing.Reason += " | " + result.Reason
			} else {
				result.Score *= 0.3
				resultMap[result.Document.ID] = result
			}
		}
	}

	// 添加关键词匹配结果（权重最低）
	// 只有当没有其他匹配结果时才使用关键词匹配
	if len(resultMap) == 0 {
		for _, result := range keywordResults {
			if existing, ok := resultMap[result.Document.ID]; ok {
				// 如果文档已存在，融合分数
				existing.Score += result.Score * 0.1 // 关键词匹配权重
				existing.Reason += " | " + result.Reason
			} else {
				result.Score *= 0.1
				resultMap[result.Document.ID] = result
			}
		}
	}

	// 转换为切片
	var fusedResults []*MatchResult
	for _, result := range resultMap {
		fusedResults = append(fusedResults, result)
	}

	// 最终过滤：移除分数过低的结果
	var filteredResults []*MatchResult
	minScore := 0.1 // 最低分数阈值
	for _, result := range fusedResults {
		if result.Score >= minScore {
			filteredResults = append(filteredResults, result)
		}
	}

	rm.logger.Debug("融合结果统计",
		slog.Int("融合前", len(fusedResults)),
		slog.Int("过滤后", len(filteredResults)))

	return filteredResults
}

// GetTopKResults 获取Top K结果
func (rm *RAGMatcher) GetTopKResults(results []*MatchResult, k int) []*MatchResult {
	if len(results) <= k {
		return results
	}

	// 排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results[:k]
}

// SmartMatcher 智能匹配器（简化版，用于快速匹配）
type SmartMatcher struct {
	logger *slog.Logger
}

// NewSmartMatcher 创建智能匹配器
func NewSmartMatcher(logger *slog.Logger) *SmartMatcher {
	if logger == nil {
		logger = slog.Default().With("component", "smart_matcher")
	}
	return &SmartMatcher{
		logger: logger.With("component", "smart_matcher"),
	}
}

// GetRelevantChunksWithSimilarity 获取相关文档块（带相似度）
func (sm *SmartMatcher) GetRelevantChunksWithSimilarity(ctx context.Context, query string, chunks []*schema.Document, topK int, threshold float64) []*schema.Document {
	if len(chunks) == 0 {
		return nil
	}
	var relevant []*schema.Document

	for _, chunk := range chunks {
		similarity := sm.calculateSimpleSimilarity(query, chunk.Content)
		if similarity >= threshold {
			relevant = append(relevant, chunk)
			if len(relevant) >= topK {
				break
			}
		}
	}

	return relevant
}

// calculateSimpleSimilarity 计算简单相似度
func (sm *SmartMatcher) calculateSimpleSimilarity(query, content string) float64 {
	queryWords := sm.extractWords(query)
	contentWords := sm.extractWords(content)

	if len(queryWords) == 0 {
		return 0.0
	}

	// 计算词频
	queryFreq := make(map[string]int)
	for _, word := range queryWords {
		queryFreq[word]++
	}

	contentFreq := make(map[string]int)
	for _, word := range contentWords {
		contentFreq[word]++
	}

	// 计算交集
	intersection := 0
	for word := range queryFreq {
		if _, exists := contentFreq[word]; exists {
			intersection++
		}
	}

	return float64(intersection) / float64(len(queryWords))
}

// extractWords 提取词语
func (sm *SmartMatcher) extractWords(text string) []string {
	var words []string
	word := ""

	for _, char := range text {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || (char >= '\u4e00' && char <= '\u9fa5') {
			word += string(char)
		} else {
			if word != "" {
				words = append(words, word)
				word = ""
			}
		}
	}

	if word != "" {
		words = append(words, word)
	}

	return words
}
