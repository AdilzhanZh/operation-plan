package files

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

const maxFileSizeBytes int64 = 20 << 20

var allowedExtensions = map[string]bool{
	".pdf":  true,
	".docx": true,
	".xlsx": true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

func ValidateUpload(fileHeader *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[ext] {
		return fmt.Errorf("unsupported extension: %s", ext)
	}

	if fileHeader.Size > maxFileSizeBytes {
		return fmt.Errorf("file exceeds %d MB", maxFileSizeBytes>>20)
	}

	return nil
}
