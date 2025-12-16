// Copyright 2025 Matthew Gall <me@matthewgall.dev>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

var (
	version = "dev"
	commit  = "unknown"
)

// GetVersion returns the application version
func GetVersion() string {
	if version != "dev" {
		return version
	}

	// Try to get version from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
		// Look for vcs.revision
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				if len(setting.Value) > 7 {
					return setting.Value[:7]
				}
				return setting.Value
			}
		}
	}

	if commit != "unknown" {
		if len(commit) > 7 {
			return commit[:7]
		}
		return commit
	}

	return "dev"
}

// GetUserAgent returns the user agent string for API requests
func GetUserAgent() string {
	return fmt.Sprintf("matthewgall/octobudget %s", GetVersion())
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

// CheckForUpdates checks if a newer version is available on GitHub
func CheckForUpdates(logger *Logger) {
	currentVersion := GetVersion()

	// Skip update check for development builds
	if currentVersion == "dev" || !strings.HasPrefix(currentVersion, "v") {
		logger.Debug("Skipping update check for development build")
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://api.github.com/repos/matthewgall/octobudget/releases/latest")
	if err != nil {
		logger.Debug("Failed to check for updates", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Debug("Failed to check for updates", "status", resp.StatusCode)
		return
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		logger.Debug("Failed to parse update response", "error", err)
		return
	}

	if release.TagName == "" {
		return
	}

	// Simple version comparison (assumes semantic versioning)
	if release.TagName != currentVersion && isNewerVersion(release.TagName, currentVersion) {
		logger.UserMessage("\n╔══════════════════════════════════════════════════════════════╗")
		logger.UserMessage("║  A new version of octobudget is available!                  ║")
		logger.UserMessage("║  Current: %-48s║", currentVersion)
		logger.UserMessage("║  Latest:  %-48s║", release.TagName)
		logger.UserMessage("║                                                              ║")
		logger.UserMessage("║  Download: %-49s║", release.HTMLURL)
		logger.UserMessage("╚══════════════════════════════════════════════════════════════╝\n")
	}
}

// isNewerVersion performs a simple semantic version comparison
func isNewerVersion(latest, current string) bool {
	// Remove 'v' prefix if present
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	// Split versions by '.'
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	// Compare each part
	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	// If all parts are equal, check if latest has more parts
	return len(latestParts) > len(currentParts)
}
