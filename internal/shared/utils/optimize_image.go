package utils

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

func OptimizeImage(srcPath string) (string, int64, error) {
	srcImg, err := imaging.Open(srcPath)
	if err != nil {
		return "", 0, err
	}

	dstImg := imaging.Resize(srcImg, 0, 2048, imaging.Lanczos)

	ext := filepath.Ext(srcPath)
	nameWithoutExt := strings.TrimSuffix(srcPath, ext)
	outputPath := nameWithoutExt + "_opt.jpg"

	f, err := os.Create(outputPath)
	if err != nil {
		return "", 0, err
	}

	err = jpeg.Encode(f, dstImg, &jpeg.Options{Quality: 40})
	f.Close()
	if err != nil {
		return "", 0, err
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return "", 0, err
	}

	os.Remove(srcPath)

	return outputPath, info.Size(), nil
}
