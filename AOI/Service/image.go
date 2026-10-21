package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
)

// convertToJPG nhận vào đường dẫn file gốc và lưu thành file .jpg
func convertToJPG(inputPath string, quality int) error {
	nameWithoutExt := strings.TrimSuffix(inputPath, filepath.Ext(inputPath))
	outputPath := nameWithoutExt + ".jpg"

	// 1. Mở file ảnh gốc
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("không thể mở file: %v", err)
	}
	defer file.Close()

	// Tự động giải mã (phụ thuộc vào việc đã import vế trên)
	img, ext, err := image.Decode(file)
	if err != nil {
		// Nếu tự động decode lỗi với bmp, thử ép kiểu giải mã bằng thư viện bmp
		if strings.ToLower(filepath.Ext(inputPath)) == ".bmp" {
			file.Seek(0, 0) // Quay lại đầu file
			img, err = bmp.Decode(file)
		}
		if err != nil {
			return fmt.Errorf("không thể giải mã ảnh (%s): %v", ext, err)
		}
	}

	// 2. Xử lý nền trắng cho ảnh trong suốt (nếu là PNG)
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	// Tô nền trắng toàn bộ ảnh mới
	draw.Draw(newImg, bounds, &image.Uniform{color.White}, image.Point{}, draw.Src)
	// Đè ảnh cũ lên trên nền trắng đó
	draw.Draw(newImg, bounds, img, bounds.Min, draw.Over)

	// 3. Tạo file JPG đầu ra
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("không thể tạo file đầu ra: %v", err)
	}
	defer outFile.Close()

	// 4. Ghi dữ liệu dạng JPEG với chất lượng cấu hình (1-100)
	options := jpeg.Options{Quality: quality}
	err = jpeg.Encode(outFile, newImg, &options)
	if err != nil {
		return fmt.Errorf("không thể mã hóa mã JPEG: %v", err)
	}

	return nil
}
