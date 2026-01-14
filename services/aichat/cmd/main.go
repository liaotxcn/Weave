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
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"weave/pkg"
	"weave/services/aichat/internal/api"
	"weave/services/aichat/internal/service"

	"go.uber.org/zap"
)

func main() {
	// 解析命令行参数
	apiMode := flag.Bool("api", false, "启动 aichat 服务器模式")
	apiAddr := flag.String("api-addr", ":8080", "aichat 服务器监听地址")
	flag.Parse()

	ctx := context.Background()

	// 创建日志实例
	logger := pkg.GetLogger()

	// 创建服务
	chatService := service.NewChatService()

	// 初始化服务
	if err := chatService.Initialize(ctx); err != nil {
		logger.Error("创建智能助手失败", zap.Error(err))
		fmt.Println("抱歉，创建智能助手失败，请检查服务配置和连接。")
		return
	}
	defer chatService.Close(ctx)

	if *apiMode {
		// 创建并启动 aichat 服务器
		apiServer := api.NewAPIServer(chatService, *apiAddr)
		if err := apiServer.Start(); err != nil {
			logger.Fatal("aichat 服务器启动失败", zap.Error(err))
		}
		return
	}

	// 模拟用户ID（实际应用中从认证系统获取）
	userID := "default_user"

	// 输出欢迎信息
	fmt.Println("欢迎使用 PaiChat 智能助手")
	fmt.Println("你可以输入任何问题，输入 'exit' 或 'quit' 退出程序。")
	fmt.Println("在对话生成回复时，输入 'pause' 暂停，输入 'continue' 继续，输入 'stop' 停止。")
	fmt.Println(strings.Repeat("=", 50))

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

		// 生成回复（使用流式输出）
		logger.Info("开始生成回复", zap.String("user_id", userID))
		fmt.Print("PaiChat: ")

		// 控制信号变量
		var isPaused bool
		var isStopped bool
		var mu sync.Mutex
		var wg sync.WaitGroup
		pauseChan := make(chan bool, 1)
		stopChan := make(chan bool, 1)
		doneChan := make(chan bool)

		// 启动命令监听goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 创建带有超时的命令监听
			for {
				select {
				case <-doneChan:
					return
				default:
					// 设置一个短超时，定期检查doneChan
					select {
					case <-doneChan:
						return
					case <-time.After(100 * time.Millisecond):
						// 非阻塞地读取输入
						if os.Stdin != nil {
							os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
							reader := bufio.NewReader(os.Stdin)
							cmd, err := reader.ReadString('\n')
							if err == nil {
								cmd = strings.TrimSpace(strings.ToLower(cmd))
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
				}
			}
		}()

		// 控制回调函数
		controlCallback := func() (bool, bool) {
			mu.Lock()
			defer mu.Unlock()
			return isPaused, isStopped
		}

		// 流式回调函数
		streamCallback := func(content string, isToolCall bool) error {
			fmt.Print(content)
			return nil
		}

		// 处理控制信号
		go func() {
			for {
				select {
				case <-doneChan:
					return
				case <-stopChan:
					mu.Lock()
					isStopped = true
					mu.Unlock()
					return
				case pauseStatus := <-pauseChan:
					mu.Lock()
					isPaused = pauseStatus
					mu.Unlock()
				}
			}
		}()

		// 使用服务层处理用户输入
		_, err := chatService.ProcessUserInputStream(ctx, userInput, userID, streamCallback, controlCallback)
		if err != nil {
			logger.Error("生成回复失败", zap.Error(err), zap.String("user_id", userID))
			fmt.Println("抱歉，生成回复失败，请稍后重试。")
			continue
		}

		// 关闭命令监听goroutine
		close(doneChan)
		wg.Wait()

		fmt.Println() // 换行
		fmt.Println(strings.Repeat("=", 50))
	}

	// 处理可能的错误
	if err := scanner.Err(); err != nil {
		logger.Fatal("读取输入失败", zap.Error(err))
	}
}
