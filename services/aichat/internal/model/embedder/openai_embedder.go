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

type OpenAIEmbedder struct {
	apiKey     string
	embedModel string
	client     *http.Client
}

type OpenAIEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type OpenAIEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func NewOpenAIEmbedder(ctx context.Context) (embedding.Embedder, error) {
	apiKey := viper.GetString("AICHAT_OPENAI_API_KEY")
	embedModel := viper.GetString("AICHAT_OPENAI_EMBED_MODEL")

	if apiKey == "" {
		return nil, fmt.Errorf("AICHAT_OPENAI_API_KEY 未在 .env 文件中配置")
	}

	return &OpenAIEmbedder{
		apiKey:     apiKey,
		embedModel: embedModel,
		client:     &http.Client{},
	}, nil
}

func (e *OpenAIEmbedder) EmbedStrings(ctx context.Context, texts []string, options ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}

	request := OpenAIEmbedRequest{
		Model: e.embedModel,
		Input: texts,
	}

	url := "https://api.openai.com/v1/embeddings"
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	response, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("openai API returned status %d: %s", response.StatusCode, string(body))
	}

	var embedResponse OpenAIEmbedResponse
	if err := json.NewDecoder(response.Body).Decode(&embedResponse); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	embeddings := make([][]float64, 0, len(embedResponse.Data))
	for _, item := range embedResponse.Data {
		embeddings = append(embeddings, item.Embedding)
	}

	return embeddings, nil
}
