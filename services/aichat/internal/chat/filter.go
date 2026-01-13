package chat

import (
	"strings"

	"github.com/spf13/viper"
)

// SensitiveFilter 敏感内容过滤器
type SensitiveFilter struct {
	sensitiveWords    map[string]bool
	maliciousPatterns map[string]bool
	injectionPatterns map[string]bool
}

// NewSensitiveFilter 创建新的敏感内容过滤器
func NewSensitiveFilter() *SensitiveFilter {
	return &SensitiveFilter{
		sensitiveWords:    loadSensitiveWords(),
		maliciousPatterns: loadMaliciousPatterns(),
		injectionPatterns: loadInjectionPatterns(),
	}
}

// loadSensitiveWords 加载敏感词列表
func loadSensitiveWords() map[string]bool {
	// 从配置文件获取敏感词列表
	sensitiveWordsList := viper.GetString("AICHAT_SENSITIVE_WORDS")
	sensitiveWords := make(map[string]bool)

	// 解析配置文件中的敏感词列表
	if sensitiveWordsList != "" {
		for _, word := range strings.Split(sensitiveWordsList, ",") {
			if word = strings.TrimSpace(word); word != "" {
				sensitiveWords[word] = true
			}
		}
	}

	return sensitiveWords
}

// loadMaliciousPatterns 加载恶意输入模式
func loadMaliciousPatterns() map[string]bool {
	// 从配置文件获取恶意输入模式列表
	maliciousPatternsList := viper.GetString("AICHAT_MALICIOUS_PATTERNS")
	maliciousPatterns := make(map[string]bool)

	// 解析配置文件中的恶意输入模式列表
	if maliciousPatternsList != "" {
		for _, pattern := range strings.Split(maliciousPatternsList, ",") {
			if pattern = strings.TrimSpace(pattern); pattern != "" {
				maliciousPatterns[pattern] = true
			}
		}
	}

	return maliciousPatterns
}

// loadInjectionPatterns 加载提示注入模式
func loadInjectionPatterns() map[string]bool {
	injectionPatterns := make(map[string]bool)

	// 从配置文件获取自定义提示注入模式列表
	injectionPatternsList := viper.GetString("AICHAT_INJECTION_PATTERNS")
	if injectionPatternsList != "" {
		for _, pattern := range strings.Split(injectionPatternsList, ",") {
			if pattern = strings.TrimSpace(pattern); pattern != "" {
				injectionPatterns[pattern] = true
			}
		}
	}

	return injectionPatterns
}

// ContainsSensitiveContent 检查是否包含敏感内容
func (f *SensitiveFilter) ContainsSensitiveContent(input string) bool {
	inputLower := strings.ToLower(input)
	for word := range f.sensitiveWords {
		if strings.Contains(inputLower, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

// ContainsMaliciousInput 检查是否包含恶意输入
func (f *SensitiveFilter) ContainsMaliciousInput(input string) bool {
	inputLower := strings.ToLower(input)
	for pattern := range f.maliciousPatterns {
		if strings.Contains(inputLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// ContainsInjectionPattern 检查是否包含提示注入模式
func (f *SensitiveFilter) ContainsInjectionPattern(input string) bool {
	inputLower := strings.ToLower(input)
	for pattern := range f.injectionPatterns {
		if strings.Contains(inputLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// FilterSensitiveContent 过滤敏感内容
func (f *SensitiveFilter) FilterSensitiveContent(input string) string {
	result := input
	for word := range f.sensitiveWords {
		// 替换敏感词为星号
		replacement := strings.Repeat("*", len(word))
		result = strings.ReplaceAll(result, word, replacement)
		// 不区分大小写替换
		result = strings.ReplaceAll(strings.ToLower(result), strings.ToLower(word), replacement)
	}
	return result
}

// ValidateInput 验证输入是否安全
func (f *SensitiveFilter) ValidateInput(input string) (bool, string) {
	if input == "" {
		return false, "输入内容不能为空"
	}

	if f.ContainsMaliciousInput(input) || f.ContainsInjectionPattern(input) || f.ContainsSensitiveContent(input) {
		return false, "输入包含不安全内容"
	}

	return true, ""
}
