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
		// 选择最近的3个问题
		questionCount := 3
		if questionCount > len(userQuestions) {
			questionCount = len(userQuestions)
		}
		recentQuestions := userQuestions[len(userQuestions)-questionCount:]

		summaryBuilder.WriteString("用户询问了关于")
		for i, question := range recentQuestions {
			if i > 0 {
				summaryBuilder.WriteString("和")
			}
			// 提取问题的关键词
			keywords := extractKeywords(question)
			summaryBuilder.WriteString(keywords)
		}
		summaryBuilder.WriteString("的问题。")
	}

	// 包含助手的主要回答
	if len(assistantAnswers) > 0 {
		// 选择最近的1个回答
		answer := assistantAnswers[len(assistantAnswers)-1]
		// 提取回答的核心内容
		coreContent := extractCoreContent(answer)
		if coreContent != "" {
			summaryBuilder.WriteString("助手的回答是：")
			summaryBuilder.WriteString(coreContent)
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
	words := strings.Fields(text)
	if len(words) > 5 {
		words = words[:5]
	}
	return strings.Join(words, " ")
}

// extractCoreContent 提取文本的核心内容
func extractCoreContent(text string) string {
	sentences := strings.Split(text, ".")
	if len(sentences) > 2 {
		sentences = sentences[:2]
	}
	coreContent := strings.Join(sentences, ".")
	// 限制长度
	if len(coreContent) > 100 {
		coreContent = coreContent[:100] + "..."
	}
	return coreContent
}
