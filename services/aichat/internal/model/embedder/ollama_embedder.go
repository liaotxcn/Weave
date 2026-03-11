package embedder

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

type OllamaEmbedder struct {
	baseURL    string
	embedModel string
	client     *http.Client
}

type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func NewOllamaEmbedder(ctx context.Context) (embedding.Embedder, error) {
	baseURL := viper.GetString("AICHAT_OLLAMA_BASE_URL")
	embedModel := viper.GetString("AICHAT_OLLAMA_EMBED_MODEL")

	if baseURL == "" || embedModel == "" {
		return nil, fmt.Errorf("AICHAT_OLLAMA_BASE_URL 或 AICHAT_OLLAMA_EMBED_MODEL 未在 .env 文件中配置")
	}

	return &OllamaEmbedder{
		baseURL:    baseURL,
		embedModel: embedModel,
		client:     &http.Client{},
	}, nil
}

func (e *OllamaEmbedder) EmbedStrings(ctx context.Context, texts []string, options ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}

	embeddings := make([][]float64, 0, len(texts))
	for _, text := range texts {
		request := OllamaEmbedRequest{
			Model:  e.embedModel,
			Prompt: text,
		}

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

		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			return nil, fmt.Errorf("ollama API returned status %d: %s", response.StatusCode, string(body))
		}

		var embedResponse OllamaEmbedResponse
		if err := json.NewDecoder(response.Body).Decode(&embedResponse); err != nil {
			response.Body.Close()
			return nil, fmt.Errorf("decode response failed: %w", err)
		}

		response.Body.Close()

		if len(embedResponse.Embeddings) > 0 {
			embeddings = append(embeddings, embedResponse.Embeddings[0])
		}
	}

	return embeddings, nil
}
