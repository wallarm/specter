package helpers

import (
	"bufio"
	"fmt"
	"path/filepath"

	"os"
	"strings"
)

const HTTPSchemeWithDash = "http://"
const HTTPSSchemeWithDash = "https://"

func findFile(fileName string) (string, error) {
	if _, err := os.Stat(fileName); err == nil {
		return fileName, nil
	}

	binPath := filepath.Join("bin", fileName)
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	return "", fmt.Errorf("file not found: %s", fileName)
}

func UpdateFiles(updateTarget string) error {
	loadYamlPath, err := findFile("load.yaml")
	if err != nil {
		return err
	}

	if err = updateLoadYaml(loadYamlPath, updateTarget); err != nil {
		return err
	}

	ammoJsonPath, err := findFile("ammo.json")
	if err != nil {
		return err
	}

	if err = updateAmmoJson(ammoJsonPath, updateTarget); err != nil {
		return err
	}

	return nil
}

func updateLoadYaml(filePath, updateTarget string) error {
	return updateFile(filePath, func(line string) string {
		if strings.Contains(line, "target:") {
			newTarget := strings.TrimPrefix(strings.TrimPrefix(updateTarget, HTTPSchemeWithDash), HTTPSSchemeWithDash)
			if !strings.Contains(newTarget, ":") {
				if strings.HasPrefix(updateTarget, HTTPSSchemeWithDash) {
					newTarget += ":443"
				} else {
					newTarget += ":80"
				}
			}
			return fmt.Sprintf("      target: %s", newTarget)
		}
		if strings.Contains(line, "ssl :") {
			sslValue := "false"
			if strings.HasPrefix(updateTarget, HTTPSSchemeWithDash) {
				sslValue = "true"
			}
			return fmt.Sprintf("      ssl : %s", sslValue)
		}
		return line
	})
}

func updateAmmoJson(filePath, updateTarget string) error {
	newHost := strings.TrimPrefix(strings.TrimPrefix(updateTarget, HTTPSchemeWithDash), HTTPSSchemeWithDash)

	return updateFile(filePath, func(line string) string {
		line = replaceHostValue(line, `"host":`, newHost)
		line = replaceHostValue(line, `"Host":`, newHost)
		return line
	})
}

func replaceHostValue(line, fieldName, newHost string) string {
	fieldIndex := strings.Index(line, fieldName)
	if fieldIndex != -1 {
		startValueIndex := fieldIndex + len(fieldName)
		startQuoteIndex := strings.Index(line[startValueIndex:], `"`) + startValueIndex + 1
		endQuoteIndex := strings.Index(line[startQuoteIndex+1:], `"`) + startQuoteIndex + 1

		if endQuoteIndex > startQuoteIndex {
			line = line[:startQuoteIndex] + newHost + line[endQuoteIndex:]
		}
	}
	return line
}

func updateFile(filePath string, updateFunc func(string) string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, updateFunc(scanner.Text()))
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}
