package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"weave/services/rag/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// 加载 .env 文件
	if err := godotenv.Load("../.env"); err != nil {
		logger.Warn("未找到 .env 文件")
	}

	viper.SetConfigFile("../.env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		logger.Warn("viper配置读取失败", slog.Any("error", err))
	}

	// 初始化配置
	config := service.NewRAGServiceConfig()
	config.DocumentDir = getEnv("RAG_DOCUMENT_DIR", "")
	config.MaxChunkSize = 8000
	config.ChunkOverlap = 200
	config.TopKResults = 3
	config.SimilarityThreshold = 0.2

	// 初始化嵌入模型
	var embedder embedding.Embedder
	embedModelType := getEnv("RAG_EMBED_MODEL_TYPE", "")
	logger.Info("初始化嵌入模型", slog.String("modelType", embedModelType))

	switch embedModelType {
	case "ollama":
		embedder = service.NewOllamaEmbedder(
			getEnv("RAG_OLLAMA_BASE_URL", ""),
			getEnv("RAG_OLLAMA_EMBED_MODEL", ""),
		)
	case "modelscope":
		embedder = service.NewModelScopeEmbedder(
			getEnv("RAG_MODELSCOPE_API_KEY", ""),
			getEnv("RAG_MODELSCOPE_EMBED_MODEL", ""),
			getEnv("RAG_MODELSCOPE_BASE_URL", ""),
		)
	default:
		logger.Error("不支持的嵌入模型类型", slog.String("type", embedModelType))
		os.Exit(1)
	}

	// 初始化 LLM 模型
	ctx := context.Background()
	llm, err := initLLMModel(ctx, logger)
	if err != nil {
		logger.Error("初始化 LLM 模型失败", slog.Any("error", err))
		os.Exit(1)
	}

	// 创建并初始化 RAG 服务
	rragService := service.NewRAGService(embedder, llm, config, logger)
	if err := rragService.Initialize(ctx); err != nil {
		logger.Error("初始化服务失败", slog.Any("error", err))
		os.Exit(1)
	}

	// 处理命令行参数
	if len(os.Args) > 1 {
		// 获取查询参数
		query := os.Args[1]

		// 执行查询
		result, err := rragService.Query(ctx, query)
		if err != nil {
			logger.Error("查询执行失败", slog.Any("error", err))
			fmt.Printf("查询失败: %v\n", err)
		} else {

			fmt.Println("\n=== 查询结果 ===")
			fmt.Println(result)
			fmt.Println("=== 查询完成 ===\n")
		}

		// 关闭服务
		if err := rragService.Close(ctx); err != nil {

		}

		return
	}

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 关闭服务

	if err := rragService.Close(ctx); err != nil {

	}

}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// initLLMModel 根据配置初始化 LLM 模型
func initLLMModel(ctx context.Context, logger *slog.Logger) (model.BaseChatModel, error) {
	// 获取模型类型
	modelType := getEnv("RAG_MODEL_TYPE", "")

	switch modelType {
	case "ollama":
		return initOllamaModel(ctx)
	case "modelscope":
		return initModelScopeModel(ctx)
	default:
		return nil, fmt.Errorf("RAG_MODEL_TYPE 未在 .env 文件中配置或值无效")
	}
}

// initOllamaModel 初始化 Ollama 模型
func initOllamaModel(ctx context.Context) (model.BaseChatModel, error) {
	baseURL := getEnv("RAG_OLLAMA_BASE_URL", "")
	model := getEnv("RAG_OLLAMA_MODEL", "")

	if baseURL == "" || model == "" {
		return nil, fmt.Errorf("RAG_OLLAMA_BASE_URL 或 RAG_OLLAMA_MODEL 未在 .env 文件中配置")
	}

	return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   model,
	})
}

// initModelScopeModel 初始化 ModelScope 模型
func initModelScopeModel(ctx context.Context) (model.BaseChatModel, error) {
	apiKey := getEnv("RAG_MODELSCOPE_API_KEY", "")
	modelName := getEnv("RAG_MODELSCOPE_MODEL_NAME", "")
	baseURL := getEnv("RAG_MODELSCOPE_BASE_URL", "")

	if apiKey == "" || modelName == "" || baseURL == "" {
		return nil, fmt.Errorf("RAG_MODELSCOPE_API_KEY、RAG_MODELSCOPE_MODEL_NAME 或 RAG_MODELSCOPE_BASE_URL 未在 .env 文件中配置")
	}

	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  apiKey,
	})
}
