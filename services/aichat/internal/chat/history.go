package chat

import (
	"sort"
	"strings"

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
			// 中文字符，先处理当前单词（如果有），然后添加中文字符
			if currentWord.Len() > 0 {
				word := currentWord.String()
				currentWord.Reset()
				if word != "" && !stopwords[word] {
					words = append(words, word)
				}
			}
			char := string(r)
			if char != "" && !stopwords[char] {
				words = append(words, char)
			}
		} else {
			// 其他字符（如标点符号）作为分隔符
			if currentWord.Len() > 0 {
				word := currentWord.String()
				currentWord.Reset()
				if word != "" && !stopwords[word] {
					words = append(words, word)
				}
			}
		}
	}

	// 处理最后一个单词
	if currentWord.Len() > 0 {
		word := currentWord.String()
		if word != "" && !stopwords[word] {
			words = append(words, word)
		}
	}

	return words
}

// FilterRelevantHistory 过滤与当前问题相关的对话历史，支持中英文混合
func FilterRelevantHistory(chatHistory []*schema.Message, currentQuestion string, maxHistory int) []*schema.Message {
	var relevant []*schema.Message

	// 如果历史记录为空或最大保留数量为0，返回空切片
	if len(chatHistory) == 0 || maxHistory <= 0 {
		return relevant
	}

	// 对当前问题进行分词
	questionWords := segmentText(currentQuestion)

	// 如果没有提取到关键词，返回最近的几条消息
	if len(questionWords) == 0 {
		startIndex := len(chatHistory) - maxHistory
		if startIndex < 0 {
			startIndex = 0
		}
		return chatHistory[startIndex:]
	}

	// 为每条历史消息计算相关性分数
	type scoredMessage struct {
		message *schema.Message
		score   int
		index   int // 原始索引，用于保持时间顺序
	}
	var scoredMessages []scoredMessage

	for i, msg := range chatHistory {
		msgContent := msg.Content
		if msgContent == "" {
			continue
		}

		// 对消息内容进行分词
		msgWords := segmentText(msgContent)
		if len(msgWords) == 0 {
			continue
		}

		// 计算关键词重叠度
		wordMap := make(map[string]bool)
		for _, word := range msgWords {
			wordMap[word] = true
		}

		var overlapCount int
		for _, word := range questionWords {
			if wordMap[word] {
				overlapCount++
			}
		}

		// 如果没有重叠，跳过该消息
		if overlapCount == 0 {
			continue
		}

		// 计算相关性分数：
		// - 基础分数：重叠词数量
		// - 时间权重：越新的消息权重越高
		// - 内容长度权重：更长的消息（提供更多上下文）获得轻微加分
		recencyWeight := len(chatHistory) - i       // 时间权重
		contentLengthWeight := len(msgWords)/10 + 1 // 内容长度权重
		score := overlapCount * recencyWeight * contentLengthWeight

		scoredMessages = append(scoredMessages, scoredMessage{
			message: msg,
			score:   score,
			index:   i,
		})
	}

	// 按分数降序排序（分数相同时按时间升序）
	sort.Slice(scoredMessages, func(i, j int) bool {
		if scoredMessages[i].score != scoredMessages[j].score {
			return scoredMessages[i].score > scoredMessages[j].score
		}
		return scoredMessages[i].index < scoredMessages[j].index
	})

	// 选择前N条消息并保持时间顺序
	selectedIndices := make(map[int]bool)
	for i := 0; i < len(scoredMessages) && i < maxHistory; i++ {
		selectedIndices[scoredMessages[i].index] = true
	}

	// 按原始时间顺序收集选中的消息
	for i, msg := range chatHistory {
		if selectedIndices[i] {
			relevant = append(relevant, msg)
		}
	}

	// 如果相关消息不足，添加最近的消息补充
	if len(relevant) < maxHistory {
		var recentMessages []*schema.Message
		for i := len(chatHistory) - 1; i >= 0 && len(recentMessages) < maxHistory-len(relevant); i-- {
			if !selectedIndices[i] {
				recentMessages = append([]*schema.Message{chatHistory[i]}, recentMessages...)
			}
		}
		relevant = append(relevant, recentMessages...)
	}

	// 确保不超过最大保留数量
	if len(relevant) > maxHistory {
		relevant = relevant[len(relevant)-maxHistory:]
	}

	return relevant
}
