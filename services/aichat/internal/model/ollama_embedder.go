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

package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/spf13/viper"
)

// OllamaEmbedder 使用 Ollama API 生成嵌入客户端
type OllamaEmbedder struct {
	baseURL    string
	embedModel string
	client     *http.Client
}

// OllamaEmbedRequest Ollama 嵌入请求结构
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse Ollama 嵌入响应结构
type OllamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

// NewOllamaEmbedder 创建新的 Ollama 嵌入客户端
func NewOllamaEmbedder(ctx context.Context) (embedding.Embedder, error) {
	baseURL := viper.GetString("OLLAMA_BASE_URL")
	embedModel := viper.GetString("OLLAMA_EMBED_MODEL")

	if baseURL == "" || embedModel == "" {
		return nil, fmt.Errorf("OLLAMA_BASE_URL 或 OLLAMA_EMBED_MODEL 未在 .env 文件中配置")
	}

	return &OllamaEmbedder{
		baseURL:    baseURL,
		embedModel: embedModel,
		client:     &http.Client{},
	}, nil
}

// EmbedStrings 实现 embedding.Embedder 接口
func (e *OllamaEmbedder) EmbedStrings(ctx context.Context, texts []string, options ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}

	// Ollama API 嵌入处理
	embeddings := make([][]float64, 0, len(texts))
	for _, text := range texts {
		// 构建请求
		request := OllamaEmbedRequest{
			Model:  e.embedModel,
			Prompt: text,
		}

		// 发送请求
		url := fmt.Sprintf("%s/api/embeddings", e.baseURL)
		reqBytes, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("marshal request failed: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBytes))
		if err != nil {
			return nil, fmt.Errorf("create request failed: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		response, err := e.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("send request failed: %w", err)
		}

		// 检查响应状态
		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			return nil, fmt.Errorf("ollama API returned status %d: %s", response.StatusCode, string(body))
		}

		// 解析响应
		var embedResponse OllamaEmbedResponse
		if err := json.NewDecoder(response.Body).Decode(&embedResponse); err != nil {
			response.Body.Close()
			return nil, fmt.Errorf("decode response failed: %w", err)
		}

		response.Body.Close()

		// 将嵌入结果添加到列表中
		if len(embedResponse.Embeddings) > 0 {
			embeddings = append(embeddings, embedResponse.Embeddings[0])
		}
	}

	return embeddings, nil
}
