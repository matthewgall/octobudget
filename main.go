// Copyright 2025 Matthew Gall <me@matthewgall.dev>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	accountID := flag.String("account", "", "Octopus Energy Account ID (overrides config)")
	apiKey := flag.String("key", "", "Octopus Energy API Key (overrides config)")
	outputPath := flag.String("output", "", "Output file for report (default: stdout)")
	htmlOutput := flag.Bool("html", false, "Generate HTML report instead of Markdown")
	debug := flag.Bool("debug", false, "Enable debug logging")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("octobudget %s\n", GetVersion())
		os.Exit(0)
	}

	// Initialize logger
	logger := NewLogger(*debug)
	logger.Info("Starting octobudget", "version", GetVersion())

	// Check for updates (non-blocking)
	go CheckForUpdates(logger)

	// Load configuration
	logger.Info("Loading configuration", "config_file", *configPath)
	config, err := LoadConfig(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Override with command-line flags
	if *accountID != "" {
		config.AccountID = *accountID
	}
	if *apiKey != "" {
		config.APIKey = *apiKey
	}
	if *debug {
		config.Debug = true
		// Recreate logger with debug enabled
		logger = NewLogger(true)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully")

	// Initialize storage
	logger.Info("Initializing storage", "path", config.StoragePath)
	storage, err := NewStorage(config.StoragePath, config.AccountID, logger)
	if err != nil {
		logger.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer storage.Close()

	// Create GraphQL client
	logger.Info("Creating API client")
	client := NewOctopusClient(config.AccountID, config.APIKey, logger)

	// Create data collector
	logger.Info("Initializing data collector")
	collector := NewCollector(client, config, storage, logger)

	// Fetch all data from API
	logger.Info("Collecting data from Octopus Energy API")
	data, err := collector.CollectAll()
	if err != nil {
		logger.Error("Failed to collect data", "error", err)
		os.Exit(1)
	}

	// Create analyzer
	logger.Info("Initializing analyzer")
	analyzer := NewAnalyzer(config, logger)

	// Perform analysis
	logger.Info("Performing analysis")
	result, err := analyzer.Analyze(data)
	if err != nil {
		logger.Error("Failed to perform analysis", "error", err)
		os.Exit(1)
	}

	// Save analysis results
	logger.Info("Saving analysis results")
	if err := storage.SaveAnalysisResult(result, config.AccountID); err != nil {
		logger.Warn("Failed to save analysis results", "error", err)
	}

	// Generate report (HTML or Markdown)
	if *htmlOutput {
		logger.Info("Generating HTML report")
		htmlReporter := NewHTMLReporter(logger)
		if err := htmlReporter.GenerateHTMLReport(result, *outputPath); err != nil {
			logger.Error("Failed to generate HTML report", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Info("Generating Markdown report")
		reporter := NewReporter(logger)
		if err := reporter.GenerateReport(result, *outputPath); err != nil {
			logger.Error("Failed to generate report", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Analysis completed successfully")
}
