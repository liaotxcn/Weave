/*
 * Copyright 2024 CloudWeGo Authors
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

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"weave/services/aichat/internal/cache"
	"weave/services/aichat/internal/chat"
	"weave/services/aichat/internal/model"
	"weave/services/aichat/internal/stream"
	"weave/services/aichat/internal/template"

	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	// 创建llm
	log.Printf("===create llm===")
	// cm, err := model.CreateOpenAIChatModel(ctx)
	cm, err := model.CreateOllamaChatModel(ctx)
	if err != nil {
		log.Printf("创建模型失败: %v\n", err)
		fmt.Println("抱歉，创建模型失败，请检查服务配置和连接。")
		return
	}
	log.Printf("create llm success\n\n")

	// 初始化缓存
	var chatCache cache.Cache
	var errCache error

	// 尝试创建Redis缓存
	chatCache, errCache = cache.NewRedisClient(ctx)
	if errCache != nil {
		log.Printf("Redis连接失败，将使用内存缓存: %v\n", errCache)
		chatCache = cache.NewInMemoryCache()
	}
	defer chatCache.Close()

	// 模拟用户ID（实际应用中从认证系统获取）
	userID := "default_user"

	// 加载对话历史
	chatHistory, err := chatCache.LoadChatHistory(ctx, userID)
	if err != nil {
		log.Printf("加载对话历史失败，将使用空历史: %v\n", err)
		chatHistory = []*schema.Message{}
	}
	log.Printf("已加载%d条对话历史记录\n", len(chatHistory))

	// 输出欢迎信息
	fmt.Println("欢迎使用 PaiChat 智能助手")
	fmt.Println("你可以输入任何问题，输入 'exit' 或 'quit' 退出程序。")
	fmt.Println("在对话生成回复时，输入 'pause' 暂停，输入 'continue' 继续，输入 'stop' 停止。")
	fmt.Println(strings.Repeat("=", 50))

	// 创建模板
	template := template.CreateTemplate()

	// 读取用户输入
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// 获取用户输入
		fmt.Print("你: ")
		scanner.Scan()
		userInput := scanner.Text()

		// 检查退出条件
		if strings.ToLower(userInput) == "exit" || strings.ToLower(userInput) == "quit" {
			fmt.Println("再见！期待下次相遇！")
			break
		}

		// 检查空输入，如果用户没有输入内容，则重新等待输入
		if strings.TrimSpace(userInput) == "" {
			continue
		}

		// 使用模板生成消息
		// 过滤与当前问题相关的对话历史
		filteredHistory := chat.FilterRelevantHistory(chatHistory, userInput, 50)
		messages, err := template.Format(ctx, map[string]any{
			"role":         "PaiChat",
			"style":        "积极、温暖且专业",
			"question":     userInput,
			"chat_history": filteredHistory,
		})
		if err != nil {
			log.Printf("format template failed: %v\n", err)
			continue
		}

		// 生成回复（使用流式输出）
		log.Printf("===llm stream generate===")
		fmt.Print("PaiChat: ")

		// 使用流式生成并实时输出
		streamReader, err := stream.Stream(ctx, cm, messages)
		if err != nil {
			log.Printf("生成回复失败: %v\n", err)
			fmt.Println("抱歉，生成回复失败，请稍后重试。")
			continue
		}
		// 标记streamReader是否已经关闭
		streamClosed := false

		// 实时处理流式输出
		var fullContent strings.Builder
		var wg sync.WaitGroup
		var isPaused bool
		var mu sync.Mutex
		pauseChan := make(chan bool)
		stopChan := make(chan bool)
		doneChan := make(chan bool)

		// 启动命令监听goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmdScanner := bufio.NewScanner(os.Stdin)
			for {
				select {
				case <-doneChan:
					return
				default:
					if cmdScanner.Scan() {
						cmd := strings.ToLower(cmdScanner.Text())
						if cmd == "pause" {
							mu.Lock()
							isPaused = true
							mu.Unlock()
							pauseChan <- true
							fmt.Println("\n[已暂停生成]")
						} else if cmd == "continue" {
							mu.Lock()
							isPaused = false
							mu.Unlock()
							pauseChan <- false
							fmt.Println("\n[已继续生成]")
						} else if cmd == "stop" {
							fmt.Println("\n[已停止生成]")
							stopChan <- true
							return
						}
					}
				}
			}
		}()

		// 处理流式输出
		for {
			// 优先检查暂停状态
			mu.Lock()
			paused := isPaused
			mu.Unlock()

			if paused {
				// 处于暂停状态，等待继续或停止信号
				select {
				case <-stopChan:
					// 关闭流式读取器以停止生成
					if !streamClosed {
						streamReader.Close()
						streamClosed = true
					}
					goto endStream
				case pauseStatus := <-pauseChan:
					// 更新暂停状态
					mu.Lock()
					isPaused = pauseStatus
					mu.Unlock()
					// 如果是继续信号，跳出暂停等待
					if !pauseStatus {
						continue
					}
				}
			} else {
				// 正常生成状态，同时监听停止和暂停信号
				select {
				case <-stopChan:
					// 关闭流式读取器以停止生成
					if !streamClosed {
						streamReader.Close()
						streamClosed = true
					}
					goto endStream
				case pauseStatus := <-pauseChan:
					// 更新暂停状态
					mu.Lock()
					isPaused = pauseStatus
					mu.Unlock()
				default:
					// 执行正常的流式输出
					message, err := streamReader.Recv()
					if err != nil {
						if err == io.EOF {
							break
						}
						log.Printf("流式接收失败: %v\n", err)
						break
					}

					// 输出当前片段
					fmt.Print(message.Content)
					fullContent.WriteString(message.Content)
				}
			}
		}

	endStream:

		// 确保streamReader被正确关闭（如果尚未关闭）
		if !streamClosed {
			streamReader.Close()
			streamClosed = true
		}

		// 关闭命令监听goroutine
		close(doneChan)
		wg.Wait()

		fmt.Println() // 换行
		log.Printf("stream result: %+v\n\n", fullContent.String())

		// 更新对话历史
		resultContent := fullContent.String()

		// 更新对话历史
		chatHistory = append(chatHistory,
			schema.UserMessage(userInput),
			schema.AssistantMessage(resultContent, nil),
		)

		// 保存对话历史到缓存
		err = chatCache.SaveChatHistory(ctx, userID, chatHistory)
		if err != nil {
			log.Printf("保存对话历史失败: %v\n", err)
			// 保存失败不影响后续对话
		}

		fmt.Println(strings.Repeat("=", 50))
	}

	// 处理可能的错误
	if err := scanner.Err(); err != nil {
		log.Fatalf("读取输入失败: %v\n", err)
	}
}
