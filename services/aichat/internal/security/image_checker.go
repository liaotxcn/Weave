package security

import (
	"bytes"
	"encoding/base64"
	"errors"
)

// ImageFormat 图片格式信息
type ImageFormat struct {
	Name       string
	Extension  string
	MagicBytes []byte
}

// 支持图片格式
var supportedImageFormats = []ImageFormat{
	{"JPEG", "jpg", []byte{0xFF, 0xD8, 0xFF}},
	{"PNG", "png", []byte{0x89, 0x50, 0x4E, 0x47}},
	{"GIF", "gif", []byte{0x47, 0x49, 0x46, 0x38}},
	{"WebP", "webp", []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}},
	{"BMP", "bmp", []byte{0x42, 0x4D}},
}

// CheckBase64Image 检查 Base64 编码的图片是否合法
func CheckBase64Image(base64Str string) (ImageFormat, error) {
	// 验证 Base64 编码合法性
	decoded, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return ImageFormat{}, errors.New("invalid base64 encoding")
	}

	// 检查图片大小（可选，根据实际需求调整）
	if len(decoded) == 0 {
		return ImageFormat{}, errors.New("empty image data")
	}

	if len(decoded) > 5*1024*1024 { // 5MB 限制
		return ImageFormat{}, errors.New("image too large")
	}

	// 检查文件头，判断是否为支持的图片格式
	for _, format := range supportedImageFormats {
		if len(decoded) >= len(format.MagicBytes) && bytes.HasPrefix(decoded, format.MagicBytes) {
			return format, nil
		}
	}

	return ImageFormat{}, errors.New("unsupported image format")
}

// IsValidBase64Image 快速检查 Base64 图片是否有效
func IsValidBase64Image(base64Str string) error {
	_, err := CheckBase64Image(base64Str)
	return err
}

// GetImageFormatFromData 根据图片数据判断图片格式
func GetImageFormatFromData(data []byte) (ImageFormat, error) {
	if len(data) == 0 {
		return ImageFormat{}, errors.New("empty image data")
	}

	for _, format := range supportedImageFormats {
		if len(data) >= len(format.MagicBytes) && bytes.HasPrefix(data, format.MagicBytes) {
			return format, nil
		}
	}

	return ImageFormat{}, errors.New("unknown image format")
}
