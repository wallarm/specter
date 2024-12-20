package helpers

import (
	"bufio"
	"fmt"
	"hash/adler32"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobuffalo/envy"
	"github.com/sirupsen/logrus"
)

func GenerateGrafanaLink(versions []string) string {

	var (
		err          error
		ciPipelineID string
		endTime      string
		startTime    string
	)

	dashboardType := envy.Get("DEPLOY_TYPE", "none")
	if dashboardType == "nginx-node-docker" {
		dashboardType = "docker"
	}

	grafanaBaseURL := envy.Get("OVERLOAD_GRAFANA_BASE_URL", "none")
	ciPipelineID, err = envy.MustGet("CI_PIPELINE_ID")
	if err != nil {
		logrus.Fatal("CI_PIPELINE_ID is not set")
	}

	endTime, startTime, err = getTimeAndPastTime()
	if err != nil {
		logrus.Errorf("Error getting time and past time: %s", err)
		return ""
	}

	logrus.Printf("startTime: %s, endTime: %s", startTime, endTime)
	logrus.Printf("versions: %v", versions)

	var runIds []string
	if len(versions) > 0 {
		for _, version := range versions {
			hash := adler32.Checksum([]byte(version))
			deployKey := fmt.Sprintf("%08x", hash)
			runId := deployKey + ciPipelineID
			runIds = append(runIds, runId)
			logrus.Printf("Appending runId: %s", runId)
		}
	} else {
		logrus.Print("No versions provided, runIds will be empty")
	}
	logrus.Printf("Final runIds: %v", runIds)

	if len(runIds) != 0 {
		var runIDsEndpoint string
		for _, runId := range runIds {
			runIDsEndpoint += "&var-run_id=" + runId
		}
		grafanaLink := fmt.Sprintf("%s/d/perftest_%s+"+
			"/perftest-%s?orgId=1&from=%s&to=%s%s",
			grafanaBaseURL, dashboardType, dashboardType, startTime, endTime, runIDsEndpoint)
		logrus.Printf("Grafana link: %s", grafanaLink)
		return grafanaLink
	}
	return fmt.Sprintf("%s/d/perftest_%s/perftest-%s?orgId=1&from=%s&to=%s",
		grafanaBaseURL, dashboardType, dashboardType, startTime, endTime)
}

func getTimeAndPastTime() (string, string, error) {
	// Parse the duration string
	durationStrFromYAML, err := findDurationInScriptJS()
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

//// findDurationInYAML searches the specified directories for a file named load.yaml and extracts the duration value.
//func findDurationInYAML() (string, error) {
//	var ConfigSearchDirs = []string{"./", "./config", "/etc/specter", "./../suite/mirroring", "./bin"}
//	for _, dir := range ConfigSearchDirs {
//		// Construct the path to the load.yaml file
//		path := filepath.Join(dir, "load.yaml")
//
//		// Attempt to open the file
//		file, err := os.Open(path)
//		if err != nil {
//			if !os.IsNotExist(err) {
//				// Report any error other than "file not found"
//				return "", fmt.Errorf("error opening file at %s: %v", path, err)
//			}
//			// Continue to the next directory if file is not found
//			continue
//		}
//		defer file.Close()
//
//		// Create a scanner to read the file line by line
//		scanner := bufio.NewScanner(file)
//		for scanner.Scan() {
//			line := scanner.Text()
//
//			// Check if the line contains the keyword "duration"
//			if strings.Contains(line, "duration:") {
//				// Extract the value after the keyword, trim spaces
//				parts := strings.Split(line, ":")
//				if len(parts) > 1 {
//					value := strings.TrimSpace(parts[1])
//					return value, nil
//				}
//			}
//		}
//
//		// Check for errors encountered during scanning
//		if err = scanner.Err(); err != nil {
//			return "", fmt.Errorf("error reading file at %s: %v", path, err)
//		}
//	}
//
//	return "", fmt.Errorf("load.yaml not found in the search directories or does not contain 'duration'")
//}

func findDurationInScriptJS() (string, error) {
	duration := os.Getenv("K6_DURATION")
	if duration != "" {
		return duration, nil
	}

	ConfigSearchDirs := []string{"/scripts", "./scripts", "./../scripts"}

	for _, dir := range ConfigSearchDirs {
		path := filepath.Join(dir, "test.js")
		file, err := os.Open(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("error opening file at %s: %v", path, err)
			}
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, "duration:") {
				lineWithoutComments := strings.SplitN(line, "//", 2)[0]
				lineWithoutComments = strings.ReplaceAll(lineWithoutComments, ",", "")
				parts := strings.SplitN(lineWithoutComments, ":", 2)
				if len(parts) < 2 {
					continue
				}

				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, `"'`)

				if value == "" {
					return "", fmt.Errorf("empty duration value in file %s", path)
				}

				return value, nil
			}
		}

		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading file at %s: %v", path, err)
		}
	}
	return "", fmt.Errorf("test.js not found in the search directories or does not contain 'duration'")
}
