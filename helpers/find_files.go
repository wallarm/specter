package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindFile(fileName string) (string, error) {
	if _, err := os.Stat(fileName); err == nil {
		return fileName, nil
	}

	binPath := filepath.Join("bin", fileName)
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	return "", fmt.Errorf("file not found: %s", fileName)
}
