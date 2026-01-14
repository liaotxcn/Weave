package service

import (
	"log/slog"
	"regexp"
	"strings"
)

// ContentFilter 内容过滤器
type ContentFilter struct {
	logger *slog.Logger
}

// NewContentFilter 创建内容过滤器
func NewContentFilter(logger *slog.Logger) *ContentFilter {
	if logger == nil {
		logger = slog.Default().With("component", "content_filter")
	}
	return &ContentFilter{
		logger: logger.With("component", "content_filter"),
	}
}

// FilterAllImages 过滤所有图片内容
func (cf *ContentFilter) FilterAllImages(content string) string {
	// 匹配 Markdown 图片语法
	imageRegex := regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	filteredContent := imageRegex.ReplaceAllString(content, "")

	// 匹配 HTML 图片标签
	htmlImageRegex := regexp.MustCompile(`<img[^>]*>`)
	filteredContent = htmlImageRegex.ReplaceAllString(filteredContent, "")

	// 清理多余的空行
	filteredContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(filteredContent, "\n\n")

	return strings.TrimSpace(filteredContent)
}

// GetImageCount 获取图片数量
func (cf *ContentFilter) GetImageCount(content string) int {
	// 匹配 Markdown 图片语法
	imageRegex := regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	markdownImages := imageRegex.FindAllString(content, -1)

	// 匹配 HTML 图片标签
	htmlImageRegex := regexp.MustCompile(`<img[^>]*>`)
	htmlImages := htmlImageRegex.FindAllString(content, -1)

	return len(markdownImages) + len(htmlImages)
}

// FilterSensitiveContent 过滤敏感内容
func (cf *ContentFilter) FilterSensitiveContent(content string) string {
	// 这里可以添加敏感内容过滤逻辑
	// 例如：敏感词替换、违法内容检测等
	return content
}

// IsContentSafe 检查内容是否安全
func (cf *ContentFilter) IsContentSafe(content string) bool {
	// 这里可以添加内容安全检查逻辑
	// 例如：检测违法内容、色情内容等
	return true
}
