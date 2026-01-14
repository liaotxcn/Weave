package chat

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
)

// 中英文停用词表
var stopwords = map[string]bool{
	// 中文常用停用词
	"的": true, "了": true, "是": true, "在": true, "我": true, "有": true, "和": true,
	"就": true, "不": true, "人": true, "都": true, "一": true, "一个": true, "上": true,
	"也": true, "很": true, "到": true, "说": true, "要": true, "去": true, "你": true,
	"会": true, "着": true, "没有": true, "看": true, "好": true, "自己": true, "这": true,
	"那": true, "他": true, "她": true, "它": true, "们": true, "来": true, "做": true,
	// 英文常用停用词
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true, "is": true,
	"are": true, "was": true, "were": true, "in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true, "this": true, "that": true, "i": true,
	"you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
}

// scoredMessage 带分数的消息结构体
type scoredMessage struct {
	message *schema.Message
	score   float64
	index   int // 原始索引用于保持时间顺序
}

// addWord 处理并添加单词到结果列表
func addWord(wordBuilder *strings.Builder, words *[]string) {
	if wordBuilder.Len() > 0 {
		word := wordBuilder.String()
		wordBuilder.Reset()
		if word != "" && !stopwords[word] {
			*words = append(*words, word)
		}
	}
}

// segmentText 对文本进行分词处理，支持中英文混合
func segmentText(text string) []string {
	if text == "" {
		return []string{}
	}

	// 转换为小写
	lowerText := strings.ToLower(text)

	// 按字符拆分中文，按空格拆分英文
	var words []string
	var currentWord strings.Builder

	for _, r := range lowerText {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			// 英文字母或数字，继续构建当前单词
			currentWord.WriteRune(r)
		} else if r >= 0x4e00 && r <= 0x9fff {
			// 中文字符，先处理当前单词，然后添加中文字符
			addWord(&currentWord, &words)
			char := string(r)
			if char != "" && !stopwords[char] {
				words = append(words, char)
			}
		} else {
			// 其他字符（如标点符号）作为分隔符
			addWord(&currentWord, &words)
		}
	}

	// 处理最后一个单词
	addWord(&currentWord, &words)

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

// filterRelevantHistoryByKeywords 使用关键词匹配过滤相关历史消息
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
	questionWords := segmentText(currentQuestion)
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
		msgWords := segmentText(msg.Content)
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
