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
	"strings"
	"time"
)

// Collector orchestrates data collection from the Octopus Energy API
type Collector struct {
	client  *OctopusClient
	config  *Config
	storage *Storage
	logger  *Logger
}

// NewCollector creates a new data collector
func NewCollector(client *OctopusClient, config *Config, storage *Storage, logger *Logger) *Collector {
	return &Collector{
		client:  client,
		config:  config,
		storage: storage,
		logger:  logger,
	}
}

// CollectAll fetches all required data from the API
func (c *Collector) CollectAll() (*CollectedData, error) {
	c.logger.Info("Starting data collection")

	data := &CollectedData{
		FetchedAt: time.Now(),
	}

	// Fetch account details (try cache first - cache for 1 hour)
	c.logger.Info("Fetching account details")
	cacheKey := fmt.Sprintf("account_%s", c.config.AccountID)
	var account *Account
	cached, err := c.storage.LoadCache(cacheKey, &account)
	if err != nil {
		c.logger.Warn("Failed to load account from cache", "error", err)
	}

	if !cached {
		account, err = c.client.FetchAccountDetails()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch account details: %w", err)
		}
		// Cache account details for 1 hour
		if err := c.storage.SaveCache(cacheKey, account, 1*time.Hour); err != nil {
			c.logger.Warn("Failed to cache account details", "error", err)
		}
	} else {
		c.logger.Info("Loaded account details from cache")
	}
	data.Account = account

	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -c.config.AnalysisPeriodDays)

	c.logger.Info("Analysis period",
		"start", startDate.Format("2006-01-02"),
		"end", endDate.Format("2006-01-02"),
		"days", c.config.AnalysisPeriodDays,
	)

	// Auto-discover meters from account if not explicitly configured
	exportMPAN, exportSerials, exportAgreements := c.discoverMeters(account)

	// Fetch electricity consumption if configured or discovered
	if c.config.ElectricityMPAN != "" && c.config.ElectricitySerial != "" {
		c.logger.Info("Fetching electricity consumption data")
		consumptions, _, err := c.client.FetchElectricityConsumption(
			c.config.ElectricityMPAN,
			c.config.ElectricitySerial,
			startDate,
			endDate,
		)
		if err != nil {
			c.logger.Warn("Failed to fetch electricity consumption", "error", err)
			// Continue with other data collection
		} else {
			// Get agreements from account details
			var agreements []Agreement
			for _, prop := range account.Properties {
				for _, emp := range prop.ElectricityMeterPoints {
					if emp.MPAN == c.config.ElectricityMPAN {
						agreements = emp.Agreements
						break
					}
				}
			}

			// Detect if user configured an export meter instead of import meter
			if len(agreements) > 0 {
				tariffName := agreements[0].Tariff.DisplayName
				if containsIgnoreCase(tariffName, "export") {
					c.logger.Warn("⚠️  CONFIGURATION WARNING: You have configured an EXPORT meter",
						"mpan", c.config.ElectricityMPAN,
						"tariff", tariffName,
					)
					c.logger.Warn("Export meters track solar/battery exports to the grid, not consumption")
					c.logger.Warn("Please update electricity_mpan in config.yaml to your IMPORT meter for consumption tracking")
					c.logger.Warn("Check your account details for the meter with 'Import' or consumption tariff")
				}
			}

			// Calculate costs using tariff data
			if len(agreements) > 0 {
				tariffName := agreements[0].Tariff.DisplayName

				// Try to fetch time-varying rates from Products API for more accurate costs
				productCode, err := c.fetchProductCodeCached(tariffName)
				if err == nil {
					c.logger.Info("Fetching time-varying electricity rates from Products API")
					rates, err := c.fetchTariffRatesCached(productCode, startDate, endDate)
					if err == nil {
						consumptions = CalculateConsumptionCostsWithRates(consumptions, rates)
						c.logger.Info("Calculated electricity costs using time-varying rates", "rates_count", len(rates))
					} else {
						c.logger.Warn("Failed to fetch tariff rates, using simple tariff calculation", "error", err)
						consumptions = CalculateConsumptionCosts(consumptions, agreements)
						c.logger.Info("Calculated electricity costs from tariff data")
					}
				} else {
					c.logger.Warn("Failed to fetch product code, using simple tariff calculation", "error", err)
					consumptions = CalculateConsumptionCosts(consumptions, agreements)
					c.logger.Info("Calculated electricity costs from tariff data")
				}
			}

			data.ElectricityConsumption = consumptions
			data.ElectricityAgreements = agreements
			c.logger.Info("Electricity data collected",
				"consumptions", len(consumptions),
				"agreements", len(agreements),
			)

			// Save to storage
		}
	} else {
		c.logger.Info("Skipping electricity consumption (not configured)")
	}

	// Fetch export data if export meter was discovered
	if exportMPAN != "" && len(exportSerials) > 0 {
		c.logger.Info("Fetching solar/battery export data", "serials_to_try", len(exportSerials))

		var exports []Consumption
		var fetchErr error

		// Try each serial number until we find one with data
		for _, serial := range exportSerials {
			exports, _, fetchErr = c.client.FetchElectricityConsumption(
				exportMPAN,
				serial,
				startDate,
				endDate,
			)
			if fetchErr == nil && len(exports) > 0 {
				c.logger.Info("Found export data", "serial", serial, "records", len(exports))
				break
			}
		}

		if fetchErr != nil {
			c.logger.Warn("Failed to fetch export data", "error", fetchErr)
		} else if len(exports) > 0 {
			// Calculate export earnings using tariff rates
			if len(exportAgreements) > 0 {
				tariffName := exportAgreements[0].Tariff.DisplayName

				// Try to fetch time-varying rates from Products API
				productCode, err := c.fetchProductCodeCached(tariffName)
				if err == nil {
					c.logger.Info("Fetching time-varying export rates from Products API")
					rates, err := c.fetchTariffRatesCached(productCode, startDate, endDate)
					if err == nil {
						exports = CalculateConsumptionCostsWithRates(exports, rates)
						c.logger.Info("Calculated export earnings using time-varying rates", "rates_count", len(rates))
					} else {
						c.logger.Warn("Failed to fetch export tariff rates", "error", err)
					}
				}
			}

			data.ElectricityExport = exports
			data.ElectricityExportAgreements = exportAgreements
			c.logger.Info("Export data collected",
				"exports", len(exports),
				"agreements", len(exportAgreements),
			)
		} else {
			c.logger.Warn("No export data found for any serial number")
		}
	}

	// Fetch gas consumption if configured
	if c.config.GasMPRN != "" && c.config.GasSerial != "" {
		c.logger.Info("Fetching gas consumption data")
		consumptions, _, err := c.client.FetchGasConsumption(
			c.config.GasMPRN,
			c.config.GasSerial,
			startDate,
			endDate,
		)
		if err != nil {
			c.logger.Warn("Failed to fetch gas consumption", "error", err)
			// Continue with other data collection
		} else {
			// Get agreements from account details
			var agreements []Agreement
			for _, prop := range account.Properties {
				for _, gmp := range prop.GasMeterPoints {
					if gmp.MPRN == c.config.GasMPRN {
						agreements = gmp.Agreements
						break
					}
				}
			}

			// Calculate costs using tariff data
			if len(agreements) > 0 {
				consumptions = CalculateConsumptionCosts(consumptions, agreements)
				c.logger.Info("Calculated gas costs from tariff data")
			}

			data.GasConsumption = consumptions
			data.GasAgreements = agreements
			c.logger.Info("Gas data collected",
				"consumptions", len(consumptions),
				"agreements", len(agreements),
			)

			// Save to storage
		}
	} else {
		c.logger.Info("Skipping gas consumption (not configured)")
	}

	// Save complete collected data

	c.logger.Info("Data collection completed successfully")
	return data, nil
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// fetchProductCodeCached fetches product code with caching (cache for 24 hours)
func (c *Collector) fetchProductCodeCached(tariffName string) (string, error) {
	cacheKey := fmt.Sprintf("product_code_%s", strings.ReplaceAll(tariffName, " ", "_"))
	var productCode string
	cached, err := c.storage.LoadCache(cacheKey, &productCode)
	if err != nil {
		c.logger.Warn("Failed to load product code from cache", "error", err)
	}

	if !cached {
		productCode, err = c.client.FetchProductCode(tariffName)
		if err != nil {
			return "", err
		}
		// Cache product code for 24 hours (tariff names rarely change)
		if err := c.storage.SaveCache(cacheKey, productCode, 24*time.Hour); err != nil {
			c.logger.Warn("Failed to cache product code", "error", err)
		}
	} else {
		c.logger.Debug("Loaded product code from cache", "tariff", tariffName, "code", productCode)
	}

	return productCode, nil
}

// fetchTariffRatesCached fetches tariff rates with caching (cache for 6 hours)
func (c *Collector) fetchTariffRatesCached(productCode string, startDate, endDate time.Time) ([]TariffRate, error) {
	// Cache key includes date range since rates are time-specific
	cacheKey := fmt.Sprintf("tariff_rates_%s_%s_%s",
		productCode,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	)

	var rates []TariffRate
	cached, err := c.storage.LoadCache(cacheKey, &rates)
	if err != nil {
		c.logger.Warn("Failed to load tariff rates from cache", "error", err)
	}

	if !cached {
		rates, err = c.client.FetchElectricityTariffRates(productCode, startDate, endDate)
		if err != nil {
			return nil, err
		}
		// Cache tariff rates for 6 hours (rates update periodically)
		if err := c.storage.SaveCache(cacheKey, rates, 6*time.Hour); err != nil {
			c.logger.Warn("Failed to cache tariff rates", "error", err)
		}
	} else {
		c.logger.Debug("Loaded tariff rates from cache", "product", productCode, "count", len(rates))
	}

	return rates, nil
}

// discoverMeters auto-discovers meters from account details if not explicitly configured
// Returns export meter details (MPAN, serials, agreements) if found
func (c *Collector) discoverMeters(account *Account) (string, []string, []Agreement) {
	if account == nil || len(account.Properties) == 0 {
		return "", nil, nil
	}

	property := account.Properties[0] // Use first property

	// Auto-discover electricity meters if not configured
	if c.config.ElectricityMPAN == "" && len(property.ElectricityMeterPoints) > 0 {
		// Look for import meter first (primary consumption meter)
		var importMeter *ElectricityMeterPoint
		var exportMeter *ElectricityMeterPoint

		for i := range property.ElectricityMeterPoints {
			emp := &property.ElectricityMeterPoints[i]
			if len(emp.Agreements) > 0 {
				tariffName := emp.Agreements[0].Tariff.DisplayName
				if containsIgnoreCase(tariffName, "import") {
					importMeter = emp
				} else if containsIgnoreCase(tariffName, "export") {
					exportMeter = emp
				}
			}
		}

		// Prefer import meter for consumption analysis
		if importMeter != nil {
			c.config.ElectricityMPAN = importMeter.MPAN
			if len(importMeter.Meters) > 0 {
				c.config.ElectricitySerial = importMeter.Meters[0].SerialNumber
			}
			c.logger.Info("Auto-discovered electricity import meter",
				"mpan", c.config.ElectricityMPAN,
				"serial", c.config.ElectricitySerial,
				"tariff", importMeter.Agreements[0].Tariff.DisplayName,
			)
		} else if len(property.ElectricityMeterPoints) > 0 {
			// Fallback to first meter if no import meter found
			emp := &property.ElectricityMeterPoints[0]
			c.config.ElectricityMPAN = emp.MPAN
			if len(emp.Meters) > 0 {
				c.config.ElectricitySerial = emp.Meters[0].SerialNumber
			}
			c.logger.Info("Auto-discovered electricity meter",
				"mpan", c.config.ElectricityMPAN,
				"serial", c.config.ElectricitySerial,
			)
		}

		// Return export meter details if found
		if exportMeter != nil {
			// Collect ALL serial numbers from the export meter
			// Some meters have multiple serials and we need to try them all
			exportSerials := make([]string, 0, len(exportMeter.Meters))
			for _, meter := range exportMeter.Meters {
				if meter.SerialNumber != "" {
					exportSerials = append(exportSerials, meter.SerialNumber)
				}
			}
			c.logger.Info("Detected solar/battery export meter",
				"mpan", exportMeter.MPAN,
				"serials", exportSerials,
				"tariff", exportMeter.Agreements[0].Tariff.DisplayName,
			)
			// Will be used to fetch export data
			defer func() {}() // Prevent issues with return in the middle
			exportMPAN := exportMeter.MPAN
			exportAgmts := exportMeter.Agreements

			// Auto-discover gas meter if not configured
			if c.config.GasMPRN == "" && len(property.GasMeterPoints) > 0 {
				gmp := &property.GasMeterPoints[0]
				c.config.GasMPRN = gmp.MPRN
				if len(gmp.Meters) > 0 {
					c.config.GasSerial = gmp.Meters[0].SerialNumber
				}
				c.logger.Info("Auto-discovered gas meter",
					"mprn", c.config.GasMPRN,
					"serial", c.config.GasSerial,
				)
			}

			return exportMPAN, exportSerials, exportAgmts
		}
	}

	// Auto-discover gas meter if not configured
	if c.config.GasMPRN == "" && len(property.GasMeterPoints) > 0 {
		gmp := &property.GasMeterPoints[0]
		c.config.GasMPRN = gmp.MPRN
		if len(gmp.Meters) > 0 {
			c.config.GasSerial = gmp.Meters[0].SerialNumber
		}
		c.logger.Info("Auto-discovered gas meter",
			"mprn", c.config.GasMPRN,
			"serial", c.config.GasSerial,
		)
	}

	return "", nil, nil
}
