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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	// Octopus Energy credentials
	AccountID string `yaml:"account_id"`
	APIKey    string `yaml:"api_key"`

	// Meter identifiers
	ElectricityMPAN   string `yaml:"electricity_mpan"`
	ElectricitySerial string `yaml:"electricity_serial"`
	GasMPRN           string `yaml:"gas_mprn"`
	GasSerial         string `yaml:"gas_serial"`

	// Analysis settings
	AnalysisPeriodDays int     `yaml:"analysis_period_days"`
	TargetDailySpend   float64 `yaml:"target_daily_spend"`
	AnomalyThreshold   float64 `yaml:"anomaly_threshold"`
	DirectDebitAmount  float64 `yaml:"direct_debit_amount"`

	// Storage
	StoragePath string `yaml:"storage_path"`

	// Debugging
	Debug bool `yaml:"debug"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Set defaults
	config := &Config{
		AnalysisPeriodDays: 90,
		AnomalyThreshold:   50.0,
		StoragePath:        getDefaultStoragePath(),
		Debug:              false,
	}

	// If no path provided, return defaults with env var overrides
	if path == "" {
		config.applyEnvironmentVariables()
		return config, nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	config.applyEnvironmentVariables()

	return config, nil
}

// getDefaultStoragePath returns the default storage path
func getDefaultStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".octobudget"
	}
	return filepath.Join(home, ".config", "octobudget")
}

// applyEnvironmentVariables overrides config with environment variables
func (c *Config) applyEnvironmentVariables() {
	if val := os.Getenv("OCTOPUS_ACCOUNT_ID"); val != "" {
		c.AccountID = val
	}
	if val := os.Getenv("OCTOPUS_API_KEY"); val != "" {
		c.APIKey = val
	}
	if val := os.Getenv("OCTOPUS_ELECTRICITY_MPAN"); val != "" {
		c.ElectricityMPAN = val
	}
	if val := os.Getenv("OCTOPUS_ELECTRICITY_SERIAL"); val != "" {
		c.ElectricitySerial = val
	}
	if val := os.Getenv("OCTOPUS_GAS_MPRN"); val != "" {
		c.GasMPRN = val
	}
	if val := os.Getenv("OCTOPUS_GAS_SERIAL"); val != "" {
		c.GasSerial = val
	}
	if val := os.Getenv("OCTOPUS_STORAGE_PATH"); val != "" {
		c.StoragePath = val
	}
	if val := os.Getenv("OCTOPUS_DEBUG"); val == "true" || val == "1" {
		c.Debug = true
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errors []string

	// Required fields
	if c.AccountID == "" {
		errors = append(errors, "account_id is required")
	} else if !strings.HasPrefix(c.AccountID, "A-") {
		errors = append(errors, "account_id must start with 'A-'")
	}

	if c.APIKey == "" {
		errors = append(errors, "api_key is required")
	} else if len(c.APIKey) < 20 {
		errors = append(errors, "api_key appears to be invalid (too short)")
	}

	// Validate analysis period
	if c.AnalysisPeriodDays < 1 || c.AnalysisPeriodDays > 365 {
		errors = append(errors, "analysis_period_days must be between 1 and 365")
	}

	// Validate anomaly threshold
	if c.AnomalyThreshold < 0 || c.AnomalyThreshold > 100 {
		errors = append(errors, "anomaly_threshold must be between 0 and 100")
	}

	// Set default storage path if empty
	if c.StoragePath == "" {
		c.StoragePath = getDefaultStoragePath()
	}

	// Meter configuration is now optional - meters will be auto-discovered from account
	// No validation needed

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
