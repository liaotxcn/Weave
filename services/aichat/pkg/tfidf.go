package pkg

import (
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/jdkato/prose/v2"
)

// 基于TF-IDF算法的文本关键词提取和语义分析

// TFIDFCalculator TF-IDF计算器
type TFIDFCalculator struct {
	documents  []string
	vocabulary map[string]int
	idfCache   map[string]float64
	mu         sync.RWMutex
}

// NewTFIDFCalculator 创建TF-IDF计算器
func NewTFIDFCalculator(documents []string) *TFIDFCalculator {
	calc := &TFIDFCalculator{
		documents:  documents,
		vocabulary: make(map[string]int),
		idfCache:   make(map[string]float64),
	}
	calc.buildVocabulary()
	return calc
}

// Calculate 计算文本的TF-IDF分数
func (calc *TFIDFCalculator) Calculate(text string) map[string]float64 {
	words := calc.tokenize(text)
	tf := calc.calculateTF(words)

	calc.mu.RLock()
	defer calc.mu.RUnlock()

	result := make(map[string]float64)
	for word, tfScore := range tf {
		if idfScore, exists := calc.idfCache[word]; exists {
			result[word] = tfScore * idfScore
		}
	}
	return result
}

// ExtractKeywords 提取关键词（按TF-IDF值排序）
func (calc *TFIDFCalculator) ExtractKeywords(text string, topN int) []string {
	scores := calc.Calculate(text)

	type keywordScore struct {
		word  string
		score float64
	}

	var keywordScores []keywordScore
	for word, score := range scores {
		keywordScores = append(keywordScores, keywordScore{word, score})
	}

	// 按分数降序排序
	sort.Slice(keywordScores, func(i, j int) bool {
		return keywordScores[i].score > keywordScores[j].score
	})

	// 返回前N个关键词
	var result []string
	for i := 0; i < topN && i < len(keywordScores); i++ {
		result = append(result, keywordScores[i].word)
	}
	return result
}

// AddDocument 添加新文档（增量更新）
func (calc *TFIDFCalculator) AddDocument(doc string) {
	calc.mu.Lock()
	defer calc.mu.Unlock()

	calc.documents = append(calc.documents, doc)
	words := calc.tokenize(doc)

	// 更新词汇表
	for _, word := range words {
		calc.vocabulary[word]++
	}

	// 清除相关IDF缓存
	for _, word := range words {
		delete(calc.idfCache, word)
	}
}

// tokenize 使用prose进行分词
func (calc *TFIDFCalculator) tokenize(text string) []string {
	doc, err := prose.NewDocument(text)
	if err != nil {
		// prose分词失败，返回空切片
		return []string{}
	}

	var words []string
	for _, tok := range doc.Tokens() {
		word := strings.ToLower(strings.TrimSpace(tok.Text))
		if len(word) > 1 { // 过滤单字符词
			words = append(words, word)
		}
	}
	return words
}

// calculateTF 计算词频
func (calc *TFIDFCalculator) calculateTF(words []string) map[string]float64 {
	tf := make(map[string]float64)
	total := len(words)

	if total == 0 {
		return tf
	}

	for _, word := range words {
		tf[word]++
	}

	// 归一化
	for word := range tf {
		tf[word] = tf[word] / float64(total)
	}

	return tf
}

// buildVocabulary 构建词汇表和IDF缓存
func (calc *TFIDFCalculator) buildVocabulary() {
	totalDocs := len(calc.documents)

	// 统计每个词出现的文档数
	docFreq := make(map[string]int)
	for _, doc := range calc.documents {
		words := calc.tokenize(doc)
		uniqueWords := make(map[string]bool)
		for _, word := range words {
			uniqueWords[word] = true
		}
		for word := range uniqueWords {
			docFreq[word]++
		}
	}

	// 计算IDF并缓存
	for word, freq := range docFreq {
		calc.idfCache[word] = math.Log(float64(totalDocs) / float64(freq+1))
		calc.vocabulary[word] = freq
	}
}

// GetVocabularySize 获取词汇表大小
func (calc *TFIDFCalculator) GetVocabularySize() int {
	calc.mu.RLock()
	defer calc.mu.RUnlock()
	return len(calc.vocabulary)
}

// GetDocumentCount 获取文档数量
func (calc *TFIDFCalculator) GetDocumentCount() int {
	calc.mu.RLock()
	defer calc.mu.RUnlock()
	return len(calc.documents)
}
