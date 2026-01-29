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

	"weave/services/aichat/pkg"

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
	maxSummaryLength int                      // 摘要最大长度
	minMessageCount  int                      // 生成摘要的最小消息数
	bm25Calculator   *pkg.BleveBM25Calculator // Bleve BM25计算器
	updateStrategy   *UpdateStrategy          // 智能更新策略
}

// UpdateStrategy 摘要更新策略
type UpdateStrategy struct {
	MinRoundsForUpdate int // 最少对话轮次才更新（默认：2）
	MaxRoundsInterval  int // 最大更新间隔轮次（默认：5）
	LastUpdateRound    int // 上次更新的轮次
}

// NewBM25SummaryGenerator 创建BM25摘要生成器
func NewBM25SummaryGenerator(conversationHistory []string) *SimpleSummaryGenerator {
	bm25Calculator := pkg.NewBleveBM25Calculator(conversationHistory)

	return &SimpleSummaryGenerator{
		maxSummaryLength: 400,
		minMessageCount:  3,
		bm25Calculator:   bm25Calculator,
		updateStrategy: &UpdateStrategy{
			MinRoundsForUpdate: 2, // 最少2轮对话才更新
			MaxRoundsInterval:  5, // 每5轮对话更新一次
			LastUpdateRound:    0, // 初始为0
		},
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

	// 智能选择对话轮次（最多保留2轮）
	selectedRounds := sg.selectConversationRounds(messages, 2)

	// 生成基于多轮对话的摘要
	if len(selectedRounds) > 0 {
		sg.generateMultiRoundSummary(&summaryBuilder, selectedRounds)
	}

	// 限制摘要长度
	summary := summaryBuilder.String()
	if len(summary) > sg.maxSummaryLength {
		summary = summary[:sg.maxSummaryLength-3] + "..."
	}

	return summary, nil
}

// UpdateSummary 智能更新对话摘要
func (sg *SimpleSummaryGenerator) UpdateSummary(ctx context.Context, existingSummary string, newMessages []*schema.Message) (string, error) {
	if len(newMessages) == 0 {
		return existingSummary, nil
	}

	// 计算当前对话轮次
	currentRound := sg.calculateCurrentRound(newMessages)

	// 智能判断是否需要更新
	if !sg.shouldUpdateSummary(currentRound, newMessages) {
		return existingSummary, nil // 不满足更新条件，返回原摘要
	}

	// 增量更新BM25词汇表
	for _, msg := range newMessages {
		if msg.Content != "" {
			sg.bm25Calculator.AddDocument(msg.Content)
		}
	}

	// 更新策略记录
	sg.updateStrategy.LastUpdateRound = currentRound

	return sg.GenerateSummary(ctx, newMessages)
}

// ExtractKeywords 提取关键词
func (sg *SimpleSummaryGenerator) ExtractKeywords(text string, topN int) []string {
	if text == "" {
		return []string{}
	}

	// 使用BM25计算器提取关键词
	keywords := sg.bm25Calculator.ExtractKeywords(text, topN)

	// 过滤关键词
	return filterKeywords(keywords)
}

// selectConversationRounds 智能选择对话轮次
func (sg *SimpleSummaryGenerator) selectConversationRounds(messages []*schema.Message, maxRounds int) []ConversationRound {
	var rounds []ConversationRound
	var currentRound ConversationRound

	for _, msg := range messages {
		if msg.Content == "" {
			continue
		}

		if msg.Role == schema.User {
			// 如果是新的一轮对话，保存当前轮次
			if currentRound.AssistantAnswer != "" {
				rounds = append(rounds, currentRound)
				currentRound = ConversationRound{}
			}
			currentRound.UserQuestion = msg.Content
		} else if msg.Role == schema.Assistant && currentRound.UserQuestion != "" {
			currentRound.AssistantAnswer = msg.Content
		}
	}

	// 添加最后一轮（如果完整）
	if currentRound.UserQuestion != "" && currentRound.AssistantAnswer != "" {
		rounds = append(rounds, currentRound)
	}

	// 返回最近的maxRounds轮
	if len(rounds) > maxRounds {
		return rounds[len(rounds)-maxRounds:]
	}
	return rounds
}

// generateMultiRoundSummary 生成多轮对话摘要
func (sg *SimpleSummaryGenerator) generateMultiRoundSummary(builder *strings.Builder, rounds []ConversationRound) {
	if len(rounds) == 0 {
		return
	}

	// 单轮对话
	if len(rounds) == 1 {
		round := rounds[0]
		builder.WriteString("用户提问：")
		builder.WriteString(sg.truncateText(round.UserQuestion, 60))
		builder.WriteString("。助手回答：")
		builder.WriteString(sg.truncateText(round.AssistantAnswer, 100))
		builder.WriteString("。")
		return
	}

	// 多轮对话
	builder.WriteString("多轮对话摘要：")
	for i, round := range rounds {
		if i > 0 {
			builder.WriteString("；")
		}

		// 提取每轮的核心内容
		questionKeywords := sg.ExtractKeywords(round.UserQuestion, 3)
		if len(questionKeywords) > 0 {
			builder.WriteString("用户询问")
			builder.WriteString(strings.Join(questionKeywords, "、"))
			builder.WriteString("，助手提供相关解答")
		} else {
			builder.WriteString("用户讨论")
			builder.WriteString(sg.truncateText(round.UserQuestion, 30))
			builder.WriteString("等话题")
		}
	}
	builder.WriteString("。")
}

// truncateText 截断文本
func (sg *SimpleSummaryGenerator) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// calculateCurrentRound 计算当前对话轮次
func (sg *SimpleSummaryGenerator) calculateCurrentRound(messages []*schema.Message) int {
	rounds := sg.selectConversationRounds(messages, 100) // 获取所有轮次
	return len(rounds)
}

// shouldUpdateSummary 智能判断是否需要更新摘要
func (sg *SimpleSummaryGenerator) shouldUpdateSummary(currentRound int, newMessages []*schema.Message) bool {
	strategy := sg.updateStrategy

	// 规则1：至少完成最少轮次对话才更新
	if currentRound < strategy.MinRoundsForUpdate {
		return false
	}

	// 规则2：达到最大间隔轮次时更新
	if currentRound-strategy.LastUpdateRound >= strategy.MaxRoundsInterval {
		return true
	}

	// 规则3：检测到话题切换时更新
	if sg.isTopicChanged(newMessages) {
		return true
	}

	// 规则4：长时间对话后恢复时更新
	if sg.isLongPauseResumed(newMessages) {
		return true
	}

	return false
}

// isTopicChanged 检测话题是否切换
func (sg *SimpleSummaryGenerator) isTopicChanged(newMessages []*schema.Message) bool {
	if len(newMessages) < 2 {
		return false
	}

	// 话题切换检测：比较最近消息的关键词差异
	lastMsg := newMessages[len(newMessages)-1]
	secondLastMsg := newMessages[len(newMessages)-2]

	lastKeywords := sg.ExtractKeywords(lastMsg.Content, 3)
	secondLastKeywords := sg.ExtractKeywords(secondLastMsg.Content, 3)

	// 如果关键词完全不同，认为话题切换
	keywordOverlap := 0
	for _, kw1 := range lastKeywords {
		for _, kw2 := range secondLastKeywords {
			if kw1 == kw2 {
				keywordOverlap++
				break
			}
		}
	}

	// 重叠关键词少于1个，认为话题切换
	return keywordOverlap < 1
}

// isLongPauseResumed 检测长时间暂停后恢复
func (sg *SimpleSummaryGenerator) isLongPauseResumed(newMessages []*schema.Message) bool {
	// 如果新消息数量较多（>3），认为可能是长时间暂停后恢复
	return len(newMessages) > 3
}

// ConversationRound 对话轮次结构
type ConversationRound struct {
	UserQuestion    string
	AssistantAnswer string
}

// filterKeywords 过滤关键词，去除停用词和无效词
func filterKeywords(words []string) []string {
	var filtered []string

	for _, word := range words {
		// 保留长度大于1的词（BM25已处理停用词过滤）
		if len(word) > 1 {
			filtered = append(filtered, word)
		}
	}

	return filtered
}
