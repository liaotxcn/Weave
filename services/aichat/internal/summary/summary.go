/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package summary

import (
	"context"
	"strings"

	"weave/services/aichat/internal/chat"

	"github.com/cloudwego/eino/schema"
)

// SummaryGenerator 摘要生成器接口
type SummaryGenerator interface {
	// GenerateSummary 生成对话摘要
	GenerateSummary(ctx context.Context, messages []*schema.Message) (string, error)
	// UpdateSummary 更新对话摘要
	UpdateSummary(ctx context.Context, existingSummary string, newMessages []*schema.Message) (string, error)
}

// SimpleSummaryGenerator 摘要生成器实现
type SimpleSummaryGenerator struct {
	// 可配置参数
	maxSummaryLength int // 摘要最大长度
	minMessageCount  int // 生成摘要的最小消息数
}

// NewSimpleSummaryGenerator 创建简单摘要生成器
func NewSimpleSummaryGenerator() *SimpleSummaryGenerator {
	return &SimpleSummaryGenerator{
		maxSummaryLength: 200, // 摘要最大长度为200个字符
		minMessageCount:  3,   // 至少3条消息才生成摘要
	}
}

// GenerateSummary 生成对话摘要
func (sg *SimpleSummaryGenerator) GenerateSummary(ctx context.Context, messages []*schema.Message) (string, error) {
	// 检查消息数量
	if len(messages) < sg.minMessageCount {
		return "", nil
	}

	// 提取关键信息
	var userQuestions []string
	var assistantAnswers []string

	for _, msg := range messages {
		if msg.Content == "" {
			continue
		}

		if msg.Role == schema.User {
			userQuestions = append(userQuestions, msg.Content)
		} else if msg.Role == schema.Assistant {
			assistantAnswers = append(assistantAnswers, msg.Content)
		}
	}

	// 生成摘要
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString("对话摘要：")

	// 包含用户的主要问题
	if len(userQuestions) > 0 {
		// 合并所有问题提取关键词
		var allQuestions strings.Builder
		for _, q := range userQuestions {
			allQuestions.WriteString(q)
			allQuestions.WriteString(" ")
		}

		// 提取问题的关键词
		keywords := extractKeywords(allQuestions.String())
		if keywords != "" {
			summaryBuilder.WriteString("用户询问了关于")
			summaryBuilder.WriteString(keywords)
			summaryBuilder.WriteString("的问题。")
		}
	}

	// 包含助手的主要回答
	if len(assistantAnswers) > 0 {
		// 选择最近的2个回答
		recentAnswers := getRecentAnswers(assistantAnswers, 2)
		// 提取回答的核心内容
		var answersContent strings.Builder
		for i, answer := range recentAnswers {
			coreContent := extractCoreContent(answer)
			if coreContent != "" {
				if i > 0 {
					answersContent.WriteString("；")
				}
				answersContent.WriteString(coreContent)
			}
		}

		if answersContent.Len() > 0 {
			summaryBuilder.WriteString("助手的回答是：")
			summaryBuilder.WriteString(answersContent.String())
			summaryBuilder.WriteString("。")
		}
	}

	// 限制摘要长度
	summary := summaryBuilder.String()
	if len(summary) > sg.maxSummaryLength {
		summary = summary[:sg.maxSummaryLength-3] + "..."
	}

	return summary, nil
}

// UpdateSummary 更新对话摘要
func (sg *SimpleSummaryGenerator) UpdateSummary(ctx context.Context, existingSummary string, newMessages []*schema.Message) (string, error) {
	// 如果没有新消息，返回现有摘要
	if len(newMessages) == 0 {
		return existingSummary, nil
	}

	// 生成新摘要
	return sg.GenerateSummary(ctx, newMessages)
}

// extractKeywords 提取文本中的关键词
func extractKeywords(text string) string {
	words := chat.SegmentText(text)

	// 过滤停用词
	filteredWords := filterKeywords(words)

	// 限制关键词数量，避免摘要过长
	if len(filteredWords) > 5 {
		filteredWords = filteredWords[:5]
	}

	return strings.Join(filteredWords, " ")
}

// filterKeywords 过滤关键词，去除停用词和无效词
func filterKeywords(words []string) []string {
	var filtered []string

	stopwords := getStopwords()

	for _, word := range words {
		// 过滤停用词、单字词、过短词
		if !isStopword(word, stopwords) && len(word) > 1 {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// getStopwords 停用词表
func getStopwords() map[string]bool {
	return map[string]bool{
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
}

// isStopword 判断是否为停用词
func isStopword(word string, stopwords map[string]bool) bool {
	return stopwords[word]
}

// getRecentAnswers 获取最近的N个回答
func getRecentAnswers(answers []string, count int) []string {
	if len(answers) <= count {
		return answers
	}
	return answers[len(answers)-count:]
}

// extractCoreContent 提取文本的核心内容
func extractCoreContent(text string) string {
	// 分割句子（支持多种标点）
	sentences := splitSentences(text)
	if len(sentences) > 2 {
		sentences = sentences[:2]
	}
	coreContent := strings.Join(sentences, "。")
	// 限制长度
	if len(coreContent) > 100 {
		coreContent = coreContent[:100-3] + "..."
	}
	return coreContent
}

// splitSentences 分割句子
func splitSentences(text string) []string {
	var sentences []string
	var currentSentence strings.Builder

	for _, r := range text {
		if r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' {
			// 句子结束符
			if currentSentence.Len() > 0 {
				sentences = append(sentences, strings.TrimSpace(currentSentence.String()))
				currentSentence.Reset()
			}
		} else {
			currentSentence.WriteRune(r)
		}
	}

	// 处理最后一个句子
	if currentSentence.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(currentSentence.String()))
	}

	return sentences
}
