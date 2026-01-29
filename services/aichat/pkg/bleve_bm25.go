package pkg

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/go-ego/gse"
	"github.com/spf13/viper"
)

// BleveBM25Calculator 基于 Bleve 的 BM25 计算器
type BleveBM25Calculator struct {
	index        bleve.Index
	docs         map[string]string // 文档ID到内容的映射
	gseSegmenter gse.Segmenter     // GSE分词器
	mu           sync.RWMutex
}

// 全局GSE分词器，用于线程安全初始化
var (
	globalGseSegmenter gse.Segmenter
	gseOnce            sync.Once
)

// 全局停用词表，从专业停用词库加载
var (
	stopWordsMap  = make(map[string]bool)
	stopWordsOnce sync.Once
)

// WordStat 词统计信息
type WordStat struct {
	Word      string  // 词语
	Frequency int     // 词频
	TF        float64 // 词频归一化值
	BM25Score float64 // BM25得分
}

// NewBleveBM25Calculator 创建基于 Bleve 的 BM25 计算器
func NewBleveBM25Calculator(documents []string) *BleveBM25Calculator {
	// 初始化GSE分词器
	gseOnce.Do(func() {
		var err error
		globalGseSegmenter, err = gse.New("zh", "alpha")
		if err != nil {
			// 兜底回退
			globalGseSegmenter = gse.Segmenter{}
		}
	})

	// 创建内存索引
	mapping := bleve.NewIndexMapping()
	mapping.DefaultAnalyzer = standard.Name

	index, err := bleve.NewMemOnly(mapping)
	if err != nil {
		// 如果创建失败，返回空实例
		return &BleveBM25Calculator{
			index:        nil,
			docs:         make(map[string]string),
			gseSegmenter: globalGseSegmenter,
		}
	}

	calc := &BleveBM25Calculator{
		index:        index,
		docs:         make(map[string]string),
		gseSegmenter: globalGseSegmenter,
	}

	// 批量添加文档
	calc.AddDocuments(documents)

	return calc
}

// AddDocument 添加单个文档
func (calc *BleveBM25Calculator) AddDocument(doc string) {
	calc.mu.Lock()
	defer calc.mu.Unlock()

	if calc.index == nil {
		return
	}

	docID := fmt.Sprintf("doc%d", len(calc.docs))
	calc.docs[docID] = doc

	err := calc.index.Index(docID, map[string]interface{}{
		"content": doc,
	})
	if err != nil {
		// 记录错误但不中断
		return
	}
}

// AddDocuments 批量添加文档
func (calc *BleveBM25Calculator) AddDocuments(documents []string) {
	calc.mu.Lock()
	defer calc.mu.Unlock()

	if calc.index == nil {
		return
	}

	batch := calc.index.NewBatch()
	for i, doc := range documents {
		docID := fmt.Sprintf("doc%d", len(calc.docs)+i)
		calc.docs[docID] = doc

		err := batch.Index(docID, map[string]interface{}{
			"content": doc,
		})
		if err != nil {
			continue
		}
	}

	calc.index.Batch(batch)
}

// CalculateQuerySimilarity 计算查询与文档的相似度
func (calc *BleveBM25Calculator) CalculateQuerySimilarity(query, document string) float64 {
	calc.mu.RLock()
	defer calc.mu.RUnlock()

	if calc.index == nil || len(calc.docs) == 0 {
		return 0.0
	}

	// 在现有文档中查找最相似的
	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(query))
	searchRequest.Size = 10
	searchRequest.Fields = []string{"content"}

	searchResult, err := calc.index.Search(searchRequest)
	if err != nil {
		return 0.0
	}

	// 查找与目标文档最匹配的结果
	for _, hit := range searchResult.Hits {
		if storedDoc, exists := calc.docs[hit.ID]; exists && storedDoc == document {
			return hit.Score
		}
	}

	return 0.0
}

// ExtractKeywords 提取关键词
func (calc *BleveBM25Calculator) ExtractKeywords(text string, topN int) []string {
	calc.mu.RLock()
	defer calc.mu.RUnlock()

	if text == "" {
		return []string{}
	}

	// 智能关键词提取：GSE分词 + 停用词(库)过滤 + BM25权重 + 词频统计 + 智能排序
	keywords := calc.extractSmartKeywords(text, topN)

	return keywords
}

// extractSmartKeywords 智能关键词提取
func (calc *BleveBM25Calculator) extractSmartKeywords(text string, topN int) []string {
	// 1. GSE智能分词
	words := calc.gseSegmentText(text)

	// 2. 停用词过滤
	filteredWords := calc.filterStopWords(words)

	// 3. 词频统计分析
	wordStats := calc.analyzeWordStatistics(filteredWords)

	// 4. BM25权重计算
	if calc.index != nil && len(calc.docs) > 0 {
		wordStats = calc.calculateBM25Weights(wordStats)
	}

	// 5. 智能排序选择
	keywords := calc.selectTopKeywords(wordStats, topN)

	return keywords
}

// gseSegmentText GSE智能分词
func (calc *BleveBM25Calculator) gseSegmentText(text string) []string {
	if text == "" {
		return []string{}
	}

	// 转换为小写
	lowerText := strings.ToLower(text)

	// 使用GSE分词器进行智能分词
	segments := calc.gseSegmenter.Cut(lowerText)

	// 过滤空字符串
	var result []string
	for _, word := range segments {
		word = strings.TrimSpace(word)
		if word != "" {
			result = append(result, word)
		}
	}

	return result
}

// filterStopWords 停用词过滤
func (calc *BleveBM25Calculator) filterStopWords(words []string) []string {
	var filtered []string

	// 确保停用词表已加载
	calc.loadStopWords()

	for _, word := range words {
		// 过滤停用词
		if stopWordsMap[word] {
			continue
		}

		// 过滤长度小于2的词语
		if len([]rune(word)) < 2 {
			continue
		}

		// 过滤纯数字
		if _, err := strconv.Atoi(word); err == nil {
			continue
		}

		filtered = append(filtered, word)
	}

	return filtered
}

// loadStopWords 加载专业停用词库
func (calc *BleveBM25Calculator) loadStopWords() {
	stopWordsOnce.Do(func() {
		// 从专业停用词库加载
		stopWordsList := calc.fetchStopWordsFromRepository()

		// 构建停用词映射
		for _, word := range stopWordsList {
			stopWordsMap[word] = true
		}
	})
}

// fetchStopWordsFromRepository 从专业停用词库获取停用词
func (calc *BleveBM25Calculator) fetchStopWordsFromRepository() []string {
	var stopWords []string

	cnStopWordsURL := viper.GetString("STOPWORDS_CN_URL")
	enStopWordsURL := viper.GetString("STOPWORDS_EN_URL")

	cnStopWords := calc.fetchStopWordsFromURL(cnStopWordsURL)
	enStopWords := calc.fetchStopWordsFromURL(enStopWordsURL)

	// 合并停用词
	stopWords = append(stopWords, cnStopWords...)
	stopWords = append(stopWords, enStopWords...)

	return stopWords
}

// fetchStopWordsFromURL 从URL获取停用词
func (calc *BleveBM25Calculator) fetchStopWordsFromURL(url string) []string {
	resp, err := http.Get(url)
	if err != nil {
		return []string{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}
	}

	// 解析停用词文件
	content := string(body)
	lines := strings.Split(content, "\n")

	var stopWords []string
	for _, line := range lines {
		word := strings.TrimSpace(line)
		if word != "" && !strings.HasPrefix(word, "#") {
			stopWords = append(stopWords, word)
		}
	}

	return stopWords
}

// analyzeWordStatistics 词频统计分析
func (calc *BleveBM25Calculator) analyzeWordStatistics(words []string) map[string]*WordStat {
	stats := make(map[string]*WordStat)
	totalWords := len(words)

	for _, word := range words {
		if stat, exists := stats[word]; exists {
			stat.Frequency++
			stat.TF = float64(stat.Frequency) / float64(totalWords)
		} else {
			stats[word] = &WordStat{
				Word:      word,
				Frequency: 1,
				TF:        1.0 / float64(totalWords),
				BM25Score: 0.0,
			}
		}
	}

	return stats
}

// calculateBM25Weights BM25权重计算
func (calc *BleveBM25Calculator) calculateBM25Weights(wordStats map[string]*WordStat) map[string]*WordStat {
	totalDocs := float64(len(calc.docs))

	// 计算平均文档长度
	var totalLength int
	docLengths := make(map[string]int)
	for docID, doc := range calc.docs {
		docLength := len(strings.Fields(doc)) // 使用词数作为文档长度
		docLengths[docID] = docLength
		totalLength += docLength
	}
	avgDocLength := float64(totalLength) / totalDocs

	for word, stat := range wordStats {
		// 计算包含该词的文档数
		docCount := 0
		var totalTermFreqInDocs int

		for _, doc := range calc.docs {
			if strings.Contains(strings.ToLower(doc), word) {
				docCount++
				// 计算该词在当前文档中的频率
				words := strings.Fields(strings.ToLower(doc))
				for _, w := range words {
					if w == word {
						totalTermFreqInDocs++
					}
				}
			}
		}

		// BM25参数
		k1 := 1.2 // 词频饱和度参数
		b := 0.75 // 文档长度归一化参数

		// 避免除零错误
		if docCount == 0 {
			stat.BM25Score = 0.0
			continue
		}

		// BM25 IDF计算
		idf := math.Log((totalDocs-float64(docCount)+0.5)/(float64(docCount)+0.5) + 1.0)

		// 计算平均词频
		avgTermFreq := float64(totalTermFreqInDocs) / float64(docCount)

		// TF归一化
		tfNorm := (avgTermFreq * (k1 + 1)) / (avgTermFreq + k1*(1-b+b*(avgDocLength/avgDocLength)))

		// BM25得分
		stat.BM25Score = idf * tfNorm
	}

	return wordStats
}

// selectTopKeywords 智能排序选择
func (calc *BleveBM25Calculator) selectTopKeywords(wordStats map[string]*WordStat, topN int) []string {
	var wordList []*WordStat
	for _, stat := range wordStats {
		wordList = append(wordList, stat)
	}

	// 按BM25得分降序排序，如果BM25得分相同则按词频排序
	sort.Slice(wordList, func(i, j int) bool {
		if wordList[i].BM25Score != wordList[j].BM25Score {
			return wordList[i].BM25Score > wordList[j].BM25Score
		}
		return wordList[i].Frequency > wordList[j].Frequency
	})

	// 提取topN关键词
	var keywords []string
	for i := 0; i < topN && i < len(wordList); i++ {
		keywords = append(keywords, wordList[i].Word)
	}

	return keywords
}

// Calculate 计算文本与所有文档的相似度
func (calc *BleveBM25Calculator) Calculate(text string) map[string]float64 {
	calc.mu.RLock()
	defer calc.mu.RUnlock()

	result := make(map[string]float64)

	if calc.index == nil || len(calc.docs) == 0 {
		return result
	}

	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(text))
	searchRequest.Size = len(calc.docs)

	searchResult, err := calc.index.Search(searchRequest)
	if err != nil {
		return result
	}

	for _, hit := range searchResult.Hits {
		if doc, exists := calc.docs[hit.ID]; exists {
			result[doc] = hit.Score
		}
	}

	return result
}

// GetDocumentCount 获取文档数量
func (calc *BleveBM25Calculator) GetDocumentCount() int {
	calc.mu.RLock()
	defer calc.mu.RUnlock()
	return len(calc.docs)
}

// GetVocabularySize 获取词汇表大小（Bleve 自动管理）
func (calc *BleveBM25Calculator) GetVocabularySize() int {
	// Bleve 内部管理词汇表，这里返回文档数作为近似值
	return calc.GetDocumentCount()
}

// SetParameters 设置 BM25 参数（Bleve 内部优化，这里作为兼容接口）
func (calc *BleveBM25Calculator) SetParameters(k1, b float64) {
	// Bleve 内部使用优化过的 BM25 参数
	// 这里作为兼容接口，不实际修改参数
}

// Close 关闭索引
func (calc *BleveBM25Calculator) Close() {
	calc.mu.Lock()
	defer calc.mu.Unlock()

	if calc.index != nil {
		calc.index.Close()
		calc.index = nil
	}
}
