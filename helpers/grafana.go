package helpers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobuffalo/envy"
)

func GenerateGrafanaLink() string {
	dashboardType := envy.Get("DEPLOY_TYPE", "none")
	grafanaBaseURL := envy.Get("OVERLOAD_GRAFANA_BASE_URL", "none")

	startTime, endTime, err := getTimeAndPastTime()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/d/perftest_%s/perftest-%s?orgId=1&from=%s&to=%s",
		grafanaBaseURL, dashboardType, dashboardType, startTime, endTime)
}

func getTimeAndPastTime() (string, string, error) {
	// Parse the duration string
	durationStrFromYAML, err := findDurationInYAML()
	if err != nil {
		return "", "", err
	}
	duration, err := time.ParseDuration(durationStrFromYAML)
	if err != nil {
		return "", "", err
	}

	// Current time in Unix milliseconds
	now := time.Now()
	currentMillis := now.UnixNano() / 1e6

	// Time ago in Unix milliseconds
	pastTime := now.Add(-duration)
	pastMillis := pastTime.UnixNano() / 1e6

	return fmt.Sprintf("%d", currentMillis), fmt.Sprintf("%d", pastMillis), nil
}

// findDurationInYAML searches the specified directories for a file named load.yaml and extracts the duration value.
func findDurationInYAML() (string, error) {
	var ConfigSearchDirs = []string{"./", "./config", "/etc/specter", "./../suite/mirroring", "./bin"}
	for _, dir := range ConfigSearchDirs {
		// Construct the path to the load.yaml file
		path := filepath.Join(dir, "load.yaml")

		// Attempt to open the file
		file, err := os.Open(path)
		if err != nil {
			if !os.IsNotExist(err) {
				// Report any error other than "file not found"
				return "", fmt.Errorf("error opening file at %s: %v", path, err)
			}
			// Continue to the next directory if file is not found
			continue
		}
		defer file.Close()

		// Create a scanner to read the file line by line
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			// Check if the line contains the keyword "duration"
			if strings.Contains(line, "duration:") {
				// Extract the value after the keyword, trim spaces
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					value := strings.TrimSpace(parts[1])
					return value, nil
				}
			}
		}

		// Check for errors encountered during scanning
		if err = scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading file at %s: %v", path, err)
		}
	}

	return "", fmt.Errorf("load.yaml not found in the search directories or does not contain 'duration'")
}
