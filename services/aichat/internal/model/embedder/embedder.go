package embedder

import (
	"context"
	"fmt"

	"weave/services/aichat/internal/cache"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

const (
	EmbedModelOllama     = "ollama"
	EmbedModelOpenAI     = "openai"
	EmbedModelModelScope = "modelscope"
)

type CachedEmbedder struct {
	embedder       embedding.Embedder
	embeddingCache cache.EmbeddingCache
}

func NewCachedEmbedder(ctx context.Context, embedder embedding.Embedder, redisClient *redis.Client) embedding.Embedder {
	if redisClient == nil {
		return embedder
	}

	embeddingCache := cache.NewRedisEmbeddingCache(ctx, redisClient)
	return &CachedEmbedder{
		embedder:       embedder,
		embeddingCache: embeddingCache,
	}
}

func (e *CachedEmbedder) EmbedStrings(ctx context.Context, texts []string, options ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}

	if len(texts) == 1 {
		cachedEmbeddings, err := e.embeddingCache.Get(ctx, texts[0])
		if err == nil && cachedEmbeddings != nil && len(cachedEmbeddings) == 1 {
			return cachedEmbeddings, nil
		}
	}

	embeddings, err := e.embedder.EmbedStrings(ctx, texts, options...)
	if err != nil {
		return nil, err
	}

	if len(texts) == 1 && len(embeddings) == 1 {
		_ = e.embeddingCache.Set(ctx, texts[0], embeddings)
	}

	return embeddings, nil
}

func NewEmbedder(ctx context.Context) (embedding.Embedder, error) {
	modelType := viper.GetString("AICHAT_EMBED_MODEL_TYPE")
	var embedder embedding.Embedder
	var err error

	switch modelType {
	case EmbedModelOllama:
		embedder, err = NewOllamaEmbedder(ctx)
	case EmbedModelOpenAI:
		embedder, err = NewOpenAIEmbedder(ctx)
	case EmbedModelModelScope:
		embedder, err = NewModelScopeEmbedder(ctx)
	default:
		return nil, fmt.Errorf("不支持的嵌入模型类型: %s", modelType)
	}

	if err != nil {
		return nil, err
	}

	return embedder, nil
}
