package chat

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"

	"weave/services/aichat/pkg"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
	"github.com/go-ego/gse"
)

// 全局分词器，sync.Once线程安全初始化
var (
	gseSegmenter  gse.Segmenter
	segmenterOnce sync.Once
)

// scoredMessage 带分数的消息结构体
type scoredMessage struct {
	message *schema.Message
	score   float64
	index   int // 原始索引用于保持时间顺序
}

// initGse 初始化分词器
func initGse() {
	segmenterOnce.Do(func() {
		var err error
		gseSegmenter, err = gse.New("zh", "alpha")
		if err != nil {
			// 兜底回退
			gseSegmenter = gse.Segmenter{}
		}
	})
}

// segmentText 对文本进行分词处理，支持中英文混合
func SegmentText(text string) []string {
	if text == "" {
		return []string{}
	}

	// 初始化分词器
	initGse()

	// 转换为小写
	lowerText := strings.ToLower(text)

	// gse智能分词
	segments := gseSegmenter.Cut(lowerText)

	// 过滤空字符串
	var words []string
	for _, word := range segments {
		word = strings.TrimSpace(word)
		if word != "" {
			words = append(words, word)
		}
	}

	return words
}

// calculateRecencyWeight 计算消息的时间权重
func calculateRecencyWeight(chatHistory []*schema.Message, index int) float64 {
	return float64(len(chatHistory)-index) * 0.01
}

// selectAndOrderMessages 选择并按时间顺序排序消息
func selectAndOrderMessages(scoredMessages []scoredMessage, maxHistory int, chatHistory []*schema.Message, startIndex int) []*schema.Message {
	// 按分数降序排序（分数相同时按时间升序）
	sort.Slice(scoredMessages, func(i, j int) bool {
		if scoredMessages[i].score != scoredMessages[j].score {
			return scoredMessages[i].score > scoredMessages[j].score
		}
		return scoredMessages[i].index < scoredMessages[j].index
	})

	// 选择前N条消息
	selectedCount := maxHistory
	if selectedCount > len(scoredMessages) {
		selectedCount = len(scoredMessages)
	}
	selectedMessages := scoredMessages[:selectedCount]

	// 按原始时间顺序排序
	sort.Slice(selectedMessages, func(i, j int) bool {
		return selectedMessages[i].index < selectedMessages[j].index
	})

	// 提取消息
	var relevant []*schema.Message
	for _, scoredMsg := range selectedMessages {
		relevant = append(relevant, scoredMsg.message)
	}

	// 如果没有选中任何消息，返回最近的消息
	if len(relevant) == 0 {
		return chatHistory[startIndex:]
	}

	return relevant
}

// FilterRelevantHistoryWithBM25 使用BM25关键词匹配的对话历史过滤
func FilterRelevantHistoryWithBM25(chatHistory []*schema.Message, currentQuestion string, maxHistory int, bm25Calculator *pkg.BleveBM25Calculator) []*schema.Message {
	// 基本参数检查
	if len(chatHistory) == 0 || maxHistory <= 0 || bm25Calculator == nil {
		return []*schema.Message{}
	}

	if maxHistory > len(chatHistory) {
		maxHistory = len(chatHistory)
	}

	// 使用BM25提取当前问题的关键词
	keywords := bm25Calculator.ExtractKeywords(currentQuestion, 3)
	if len(keywords) == 0 {
		return []*schema.Message{}
	}

	var scoredMessages []scoredMessage

	// 为每条历史消息计算BM25关键词匹配分数
	for i, msg := range chatHistory {
		if msg.Content == "" {
			continue
		}

		// BM25关键词匹配分数
		keywordScore := calculateBM25MatchScore(msg.Content, keywords, bm25Calculator)

		// 时间权重
		recencyWeight := calculateRecencyWeight(chatHistory, i)

		// 综合分数 = BM25关键词分数 + 时间权重
		finalScore := keywordScore + recencyWeight

		scoredMessages = append(scoredMessages, scoredMessage{
			message: msg,
			score:   finalScore,
			index:   i,
		})
	}

	return selectAndOrderMessages(scoredMessages, maxHistory, chatHistory, 0)
}

// FilterRelevantHistory 过滤与当前问题相关的对话历史，支持中英文混合
func FilterRelevantHistory(ctx context.Context, embedder embedding.Embedder, chatHistory []*schema.Message, currentQuestion string, maxHistory int) []*schema.Message {
	// 如果历史记录为空或最大保留数量为0，返回空切片
	if len(chatHistory) == 0 || maxHistory <= 0 {
		return []*schema.Message{}
	}

	// 确保maxHistory不超过历史记录总数
	if maxHistory > len(chatHistory) {
		maxHistory = len(chatHistory)
	}

	// 优先获取最近的消息作为基础
	startIndex := len(chatHistory) - maxHistory
	if startIndex < 0 {
		startIndex = 0
	}
	recentMessages := chatHistory[startIndex:]

	// 如果不需要更复杂的相关性过滤，直接返回最近的消息
	if currentQuestion == "" {
		return recentMessages
	}

	// 如果嵌入器不可用，使用关键词匹配作为备选
	if embedder == nil {
		return filterRelevantHistoryByKeywords(chatHistory, currentQuestion, maxHistory)
	}

	var scoredMessages []scoredMessage

	// 生成当前问题的向量
	currentQuestionEmbedding, err := embedder.EmbedStrings(ctx, []string{currentQuestion})
	if err != nil || len(currentQuestionEmbedding) == 0 {
		// 如果向量化失败，使用关键词匹配作为备选
		return filterRelevantHistoryByKeywords(chatHistory, currentQuestion, maxHistory)
	}
	questionVector := currentQuestionEmbedding[0]

	// 准备所有需要向量化的历史消息内容
	var historyContents []string
	var validIndices []int
	for i, msg := range chatHistory {
		if msg.Content != "" {
			historyContents = append(historyContents, msg.Content)
			validIndices = append(validIndices, i)
		}
	}

	// 批量生成历史消息的向量
	var historyEmbeddings [][]float64
	if len(historyContents) > 0 {
		historyEmbeddings, err = embedder.EmbedStrings(ctx, historyContents)
		if err != nil || len(historyEmbeddings) == 0 {
			// 如果向量化失败，使用关键词匹配作为备选
			return filterRelevantHistoryByKeywords(chatHistory, currentQuestion, maxHistory)
		}
	}

	// 为每条历史消息计算相似度分数
	for i, embedding := range historyEmbeddings {
		msgIndex := validIndices[i]
		msg := chatHistory[msgIndex]

		// 计算余弦相似度
		similarity := cosineSimilarity(questionVector, embedding)

		// 即使相似度较低，也为最近的消息赋予基础分数
		baseScore := 0.0
		if msgIndex >= startIndex {
			// 最近的消息有基础分数，确保它们至少有机会被选中
			baseScore = 0.1
		}

		// 计算相关性分数：
		// - 基础分数：确保最近的消息有机会被选中
		// - 相似度分数：余弦相似度
		// - 时间权重：越新的消息权重越高
		recencyWeight := calculateRecencyWeight(chatHistory, msgIndex)
		score := baseScore + similarity + recencyWeight

		scoredMessages = append(scoredMessages, scoredMessage{
			message: msg,
			score:   score,
			index:   msgIndex,
		})
	}

	// 选择并按时间顺序排序消息
	return selectAndOrderMessages(scoredMessages, maxHistory, chatHistory, startIndex)
}

// calculateBM25MatchScore 计算BM25关键词匹配分数
func calculateBM25MatchScore(content string, keywords []string, calculator *pkg.BleveBM25Calculator) float64 {
	if len(keywords) == 0 {
		return 0.0
	}

	// 使用BM25计算内容与关键词的相似度
	matchScore := 0.0
	for _, keyword := range keywords {
		// 计算单个关键词与内容的BM25相似度
		score := calculator.CalculateQuerySimilarity(keyword, content)
		matchScore += score
	}

	// 归一化处理
	return matchScore / float64(len(keywords))
}

// EnhanceHistorySelection 基于BM25关键词重新排序历史
func EnhanceHistorySelection(chatHistory []*schema.Message, currentQuestion string, calculator *pkg.BleveBM25Calculator) []*schema.Message {
	if calculator == nil || len(chatHistory) <= 5 {
		return chatHistory
	}

	keywords := calculator.ExtractKeywords(currentQuestion, 3)
	if len(keywords) == 0 {
		return chatHistory
	}

	// 基于BM25关键词重新排序历史
	sort.Slice(chatHistory, func(i, j int) bool {
		scoreI := calculateBM25MatchScore(chatHistory[i].Content, keywords, calculator)
		scoreJ := calculateBM25MatchScore(chatHistory[j].Content, keywords, calculator)
		return scoreI > scoreJ
	})

	return chatHistory
}

// filterRelevantHistoryByKeywords 基于关键词匹配过滤相关历史
func filterRelevantHistoryByKeywords(chatHistory []*schema.Message, currentQuestion string, maxHistory int) []*schema.Message {
	// 确保maxHistory不超过历史记录总数
	if maxHistory > len(chatHistory) {
		maxHistory = len(chatHistory)
	}

	// 优先获取最近的消息作为基础
	startIndex := len(chatHistory) - maxHistory
	if startIndex < 0 {
		startIndex = 0
	}

	var scoredMessages []scoredMessage

	// 分词当前问题
	questionWords := SegmentText(currentQuestion)
	if len(questionWords) == 0 {
		// 如果问题分词后为空，返回最近的消息
		return chatHistory[startIndex:]
	}

	// 统计问题中的词频
	questionWordFreq := make(map[string]int)
	for _, word := range questionWords {
		questionWordFreq[word]++
	}

	// 计算每条历史消息的相关性分数
	for i, msg := range chatHistory {
		if msg.Content == "" {
			continue
		}

		// 分词历史消息
		msgWords := SegmentText(msg.Content)
		if len(msgWords) == 0 {
			continue
		}

		// 统计历史消息中的词频
		msgWordFreq := make(map[string]int)
		for _, word := range msgWords {
			msgWordFreq[word]++
		}

		// 计算关键词匹配分数
		matchCount := 0
		for word := range questionWordFreq {
			if _, exists := msgWordFreq[word]; exists {
				matchCount++
			}
		}

		// 计算匹配度
		matchScore := float64(matchCount) / float64(len(questionWordFreq))

		// 时间权重
		recencyWeight := calculateRecencyWeight(chatHistory, i)

		// 基础分数（确保最近的消息有机会被选中）
		baseScore := 0.0
		if i >= startIndex {
			baseScore = 0.1
		}

		totalScore := baseScore + matchScore + recencyWeight

		scoredMessages = append(scoredMessages, scoredMessage{
			message: msg,
			score:   totalScore,
			index:   i,
		})
	}

	// 选择并按时间顺序排序消息
	return selectAndOrderMessages(scoredMessages, maxHistory, chatHistory, startIndex)
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float64) float64 {
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

// rffFusion RFF (Reciprocal Rank Fusion) 多路召回结果融合
// k: RFF参数，控制排名衰减速度
func rffFusion(rankings [][]int, k float64, maxHistory int) []int {
	if len(rankings) == 0 {
		return nil
	}

	// 统计每个文档的RFF得分
	scores := make(map[int]float64)

	for _, ranking := range rankings {
		for rank, docID := range ranking {
			// RFF: score = Σ(1.0 / (k + rank))
			scores[docID] += 1.0 / (k + float64(rank+1))
		}
	}

	// 转换为切片排序
	type docScore struct {
		id    int
		score float64
	}

	results := make([]docScore, 0, len(scores))
	for id, score := range scores {
		results = append(results, docScore{id: id, score: score})
	}

	// 按RFF得分降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// 截取前maxHistory个
	if len(results) > maxHistory {
		results = results[:maxHistory]
	}

	// 提取ID
	finalRanking := make([]int, len(results))
	for i, r := range results {
		finalRanking[i] = r.id
	}

	return finalRanking
}

// recallWithBM25 A路召回：BM25关键词召回，返回文档ID排序列表
func recallWithBM25(chatHistory []*schema.Message, currentQuestion string, calculator *pkg.BleveBM25Calculator, maxHistory int) []int {
	if calculator == nil || len(chatHistory) == 0 {
		return nil
	}

	keywords := calculator.ExtractKeywords(currentQuestion, 5)
	if len(keywords) == 0 {
		return nil
	}

	type docScore struct {
		id    int
		score float64
	}
	scores := make([]docScore, 0, len(chatHistory))

	for i, msg := range chatHistory {
		if msg.Content == "" {
			continue
		}
		score := calculateBM25MatchScore(msg.Content, keywords, calculator)
		if score > 0 {
			scores = append(scores, docScore{id: i, score: score})
		}
	}

	// 按BM25得分降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 截取前maxHistory*2个（给RFF留空间）
	limit := maxHistory * 2
	if limit > len(scores) {
		limit = len(scores)
	}

	ranking := make([]int, limit)
	for i := 0; i < limit; i++ {
		ranking[i] = scores[i].id
	}

	return ranking
}

// recallWithEmbedding B路召回：Embedding向量语义召回，返回文档ID排序列表
func recallWithEmbedding(ctx context.Context, embedder embedding.Embedder, chatHistory []*schema.Message, currentQuestion string, maxHistory int) []int {
	if embedder == nil || len(chatHistory) == 0 {
		return nil
	}

	// 获取问题向量
	questionEmbeddings, err := embedder.EmbedStrings(ctx, []string{currentQuestion})
	if err != nil || len(questionEmbeddings) == 0 {
		return nil
	}
	questionVector := questionEmbeddings[0]

	// 准备历史消息
	var contents []string
	var indices []int
	for i, msg := range chatHistory {
		if msg.Content != "" {
			contents = append(contents, msg.Content)
			indices = append(indices, i)
		}
	}

	if len(contents) == 0 {
		return nil
	}

	// 批量获取向量
	historyEmbeddings, err := embedder.EmbedStrings(ctx, contents)
	if err != nil || len(historyEmbeddings) != len(contents) {
		return nil
	}

	type docScore struct {
		id    int
		score float64
	}
	scores := make([]docScore, 0, len(historyEmbeddings))

	for i, emb := range historyEmbeddings {
		similarity := cosineSimilarity(questionVector, emb)
		if similarity > 0 {
			scores = append(scores, docScore{id: indices[i], score: similarity})
		}
	}

	// 按相似度降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 截取前maxHistory*2个
	limit := maxHistory * 2
	if limit > len(scores) {
		limit = len(scores)
	}

	ranking := make([]int, limit)
	for i := 0; i < limit; i++ {
		ranking[i] = scores[i].id
	}

	return ranking
}

// FilterRelevantHistoryHybrid 多路召回+RFF排序融合
// A路：BM25关键词召回  B路：Embedding语义召回
func FilterRelevantHistoryHybrid(ctx context.Context, embedder embedding.Embedder, calculator *pkg.BleveBM25Calculator, chatHistory []*schema.Message, currentQuestion string, maxHistory int) []*schema.Message {
	if len(chatHistory) == 0 || maxHistory <= 0 {
		return []*schema.Message{}
	}

	if maxHistory > len(chatHistory) {
		maxHistory = len(chatHistory)
	}

	// 空问题直接返回最近消息
	if currentQuestion == "" {
		start := len(chatHistory) - maxHistory
		if start < 0 {
			start = 0
		}
		return chatHistory[start:]
	}

	// 两路并行召回
	var rankingA, rankingB []int
	var wg sync.WaitGroup

	// A路：BM25召回
	wg.Add(1)
	go func() {
		defer wg.Done()
		rankingA = recallWithBM25(chatHistory, currentQuestion, calculator, maxHistory)
	}()

	// B路：Embedding召回
	wg.Add(1)
	go func() {
		defer wg.Done()
		rankingB = recallWithEmbedding(ctx, embedder, chatHistory, currentQuestion, maxHistory)
	}()

	wg.Wait()

	// 收集有效召回结果
	var rankings [][]int
	if len(rankingA) > 0 {
		rankings = append(rankings, rankingA)
	}
	if len(rankingB) > 0 {
		rankings = append(rankings, rankingB)
	}

	// 如果没有召回结果，返回最近消息
	if len(rankings) == 0 {
		start := len(chatHistory) - maxHistory
		if start < 0 {
			start = 0
		}
		return chatHistory[start:]
	}

	// RFF融合排序
	finalRanking := rffFusion(rankings, 60.0, maxHistory)

	// 按最终排序提取消息
	result := make([]*schema.Message, len(finalRanking))
	for i, idx := range finalRanking {
		if idx >= 0 && idx < len(chatHistory) {
			result[i] = chatHistory[idx]
		}
	}

	return result
}
