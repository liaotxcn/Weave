package pkg

import "strings"

// ParseKeyValuePairs 解析键值对字符串 "key1:value1,key2:value2"
func ParseKeyValuePairs(input string) map[string]string {
	result := make(map[string]string)
	if input == "" {
		return result
	}

	// 按逗号分割键值对
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// 按冒号分割键值
		if idx := strings.IndexByte(pair, ':'); idx != -1 {
			key := strings.TrimSpace(pair[:idx])
			value := strings.TrimSpace(pair[idx+1:])
			if key != "" && value != "" {
				result[key] = value
			}
		}
	}

	return result
}
