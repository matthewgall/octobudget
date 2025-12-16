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
	"fmt"
	"math"
	"sort"
	"time"
)

// Analyzer performs statistical analysis on collected data
type Analyzer struct {
	config        *Config
	logger        *Logger
	weatherClient *WeatherClient
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(config *Config, logger *Logger) *Analyzer {
	return &Analyzer{
		config:        config,
		logger:        logger,
		weatherClient: NewWeatherClient(logger),
	}
}

// Analyze performs complete analysis on collected data
func (a *Analyzer) Analyze(data *CollectedData) (*AnalysisResult, error) {
	a.logger.Info("Starting analysis")

	if data.Account == nil {
		return nil, &DataError{
			DataType: "account",
			Message:  "account data is required for analysis",
		}
	}

	result := &AnalysisResult{
		GeneratedAt:                 time.Now(),
		CurrentBalance:              data.Account.Balance,
		CurrentDirectDebit:          a.config.DirectDebitAmount,
		ElectricityAgreements:       data.ElectricityAgreements,
		ElectricityExportAgreements: data.ElectricityExportAgreements,
		GasAgreements:               data.GasAgreements,
	}

	// Calculate analysis period
	if len(data.ElectricityConsumption) > 0 || len(data.GasConsumption) > 0 {
		result.AnalysisPeriodDays = a.config.AnalysisPeriodDays
		result.AnalysisPeriodEnd = time.Now()
		result.AnalysisPeriodStart = result.AnalysisPeriodEnd.AddDate(0, 0, -a.config.AnalysisPeriodDays)
	}

	// Analyze electricity consumption
	if len(data.ElectricityConsumption) > 0 {
		a.logger.LogAnalysisStage("electricity_consumption")
		result.AvgDailyElectricity = a.calculateAverageConsumption(data.ElectricityConsumption)
		result.AvgDailyCostElectricity = a.calculateAverageCost(data.ElectricityConsumption)

		// Detect electricity anomalies
		anomalies := a.detectAnomalies(data.ElectricityConsumption, "electricity")
		result.Anomalies = append(result.Anomalies, anomalies...)
	}

	// Analyze electricity exports (solar/battery)
	if len(data.ElectricityExport) > 0 {
		a.logger.LogAnalysisStage("electricity_export")
		result.AvgDailyExport = a.calculateAverageConsumption(data.ElectricityExport)
		result.AvgDailyEarningsExport = a.calculateAverageCost(data.ElectricityExport)
		a.logger.Info("Export analysis",
			"avg_daily_export_kwh", result.AvgDailyExport,
			"avg_daily_earnings", result.AvgDailyEarningsExport,
		)
	}

	// Analyze gas consumption
	if len(data.GasConsumption) > 0 {
		a.logger.LogAnalysisStage("gas_consumption")
		result.AvgDailyGas = a.calculateAverageConsumption(data.GasConsumption)
		result.AvgDailyCostGas = a.calculateAverageCost(data.GasConsumption)

		// Detect gas anomalies
		anomalies := a.detectAnomalies(data.GasConsumption, "gas")
		result.Anomalies = append(result.Anomalies, anomalies...)
	}

	// Calculate total average daily cost (import - export + gas)
	result.AvgDailyCostTotal = result.AvgDailyCostElectricity - result.AvgDailyEarningsExport + result.AvgDailyCostGas

	// Calculate projected monthly cost
	result.ProjectedMonthlyCost = result.AvgDailyCostTotal * 30

	// Calculate Direct Debit recommendation
	a.logger.LogAnalysisStage("direct_debit_recommendation")
	result.RecommendedDirectDebit = a.calculateRecommendedDirectDebit(result.AvgDailyCostTotal)
	result.PaymentStatus = a.determinePaymentStatus(result.RecommendedDirectDebit, result.CurrentDirectDebit)

	// Detect tariff changes
	a.logger.LogAnalysisStage("tariff_changes")
	if len(data.ElectricityAgreements) > 1 {
		changes := a.detectTariffChanges(data.ElectricityAgreements, "electricity")
		result.TariffChanges = append(result.TariffChanges, changes...)
	}
	if len(data.GasAgreements) > 1 {
		changes := a.detectTariffChanges(data.GasAgreements, "gas")
		result.TariffChanges = append(result.TariffChanges, changes...)
	}

	// Enrich anomalies with weather data
	if len(result.Anomalies) > 0 {
		a.logger.LogAnalysisStage("weather_enrichment")
		a.enrichAnomaliesWithWeather(result.Anomalies)

		// Filter out weather-expected anomalies
		result.Anomalies = a.filterWeatherExpectedAnomalies(result.Anomalies)
	}

	// Generate insights
	a.logger.LogAnalysisStage("insights_generation")
	result.Insights = a.generateInsights(result, data)

	a.logger.Info("Analysis completed",
		"anomalies", len(result.Anomalies),
		"tariff_changes", len(result.TariffChanges),
		"insights", len(result.Insights),
	)

	return result, nil
}

// calculateAverageConsumption calculates average daily consumption in kWh
func (a *Analyzer) calculateAverageConsumption(consumptions []Consumption) float64 {
	if len(consumptions) == 0 {
		return 0
	}

	total := 0.0
	for _, c := range consumptions {
		total += c.Value
	}

	// Divide by number of days in analysis period, not number of records
	return total / float64(a.config.AnalysisPeriodDays)
}

// calculateAverageCost calculates average daily cost in pounds
func (a *Analyzer) calculateAverageCost(consumptions []Consumption) float64 {
	if len(consumptions) == 0 {
		return 0
	}

	total := 0.0
	for _, c := range consumptions {
		total += c.Cost
	}

	// Convert pence to pounds and divide by number of days in analysis period
	return (total / 100.0) / float64(a.config.AnalysisPeriodDays)
}

// calculateRecommendedDirectDebit calculates the recommended monthly Direct Debit amount
// This considers seasonal variations and provides a year-round stable payment
func (a *Analyzer) calculateRecommendedDirectDebit(avgDailyCost float64) float64 {
	// Base monthly cost from current daily average
	baseMonthlyCost := avgDailyCost * 30

	// Apply seasonal adjustment based on current month
	currentMonth := time.Now().Month()
	seasonalMultiplier := 1.0

	// Winter months (Nov-Feb): expect 30-50% higher usage due to heating
	if currentMonth >= 11 || currentMonth <= 2 {
		seasonalMultiplier = 1.40 // Assume 40% increase for winter
	} else if currentMonth >= 3 && currentMonth <= 4 {
		// Spring transition (Mar-Apr): moderate increase
		seasonalMultiplier = 1.20
	} else if currentMonth >= 9 && currentMonth <= 10 {
		// Autumn transition (Sep-Oct): moderate increase
		seasonalMultiplier = 1.20
	} else {
		// Summer months (May-Aug): lower usage, base rate
		seasonalMultiplier = 1.0
	}

	// Calculate seasonally-adjusted annual cost
	// This averages out the seasonal variation for stable monthly payments
	annualEstimate := baseMonthlyCost * 12 * seasonalMultiplier
	recommendedMonthly := annualEstimate / 12

	// Add 10% buffer for unexpected variations
	recommendedMonthly *= 1.10

	// Round to nearest £5 for practical payment amounts
	return math.Round(recommendedMonthly/5) * 5
}

// determinePaymentStatus determines if the user is underpaying, overpaying, or balanced
func (a *Analyzer) determinePaymentStatus(recommended, current float64) string {
	if current == 0 {
		return "Unknown"
	}

	difference := math.Abs(recommended - current)

	if difference < 5 {
		return "Balanced"
	}

	if recommended > current {
		return "Underpaying"
	}

	return "Overpaying"
}

// detectAnomalies detects unusual consumption patterns on daily aggregated data
func (a *Analyzer) detectAnomalies(consumptions []Consumption, fuelType string) []Anomaly {
	if len(consumptions) < 7 {
		// Need at least a week of data for meaningful anomaly detection
		return nil
	}

	// Aggregate consumption to daily totals
	dailyConsumption := aggregateToDaily(consumptions)

	if len(dailyConsumption) < 7 {
		return nil
	}

	var anomalies []Anomaly

	// Extract daily values for statistical analysis
	values := make([]float64, 0, len(dailyConsumption))
	for _, daily := range dailyConsumption {
		values = append(values, daily.Value)
	}

	// Calculate mean and standard deviation
	mean := calculateMean(values)
	stdDev := calculateStdDev(values, mean)

	// Detect anomalies in daily data
	for _, daily := range dailyConsumption {
		// Check for very low usage (< 10% of mean)
		if daily.Value < mean*0.1 && mean > 0 {
			anomalies = append(anomalies, Anomaly{
				Date:             daily.StartAt,
				FuelType:         fuelType,
				Type:             "low_usage",
				Description:      fmt.Sprintf("Unusually low %s usage for this day", fuelType),
				ActualValue:      daily.Value,
				ExpectedValue:    mean,
				DeviationPercent: ((daily.Value - mean) / mean) * 100,
			})
			continue
		}

		// Check for consumption spike (> 2 standard deviations above mean)
		// Note: We'll filter out weather-expected spikes after enriching with weather data
		if daily.Value > mean+2*stdDev && stdDev > 0 {
			deviation := ((daily.Value - mean) / mean) * 100
			if deviation > a.config.AnomalyThreshold {
				a.logger.LogAnomalyDetected(daily.StartAt.Format("2006-01-02"), "consumption_spike", deviation)
				anomalies = append(anomalies, Anomaly{
					Date:             daily.StartAt,
					FuelType:         fuelType,
					Type:             "consumption_spike",
					Description:      fmt.Sprintf("Unusually high %s consumption", fuelType),
					ActualValue:      daily.Value,
					ExpectedValue:    mean,
					DeviationPercent: deviation,
				})
			}
		}
	}

	return anomalies
}

// aggregateToDaily aggregates half-hourly consumption data to daily totals
func aggregateToDaily(consumptions []Consumption) []Consumption {
	if len(consumptions) == 0 {
		return nil
	}

	dailyMap := make(map[string]*Consumption)

	for _, c := range consumptions {
		// Get date key (YYYY-MM-DD)
		dateKey := c.StartAt.Format("2006-01-02")

		if existing, exists := dailyMap[dateKey]; exists {
			// Add to existing day
			existing.Value += c.Value
			existing.Cost += c.Cost
		} else {
			// Create new day entry
			dayStart := time.Date(c.StartAt.Year(), c.StartAt.Month(), c.StartAt.Day(), 0, 0, 0, 0, c.StartAt.Location())
			dailyMap[dateKey] = &Consumption{
				StartAt: dayStart,
				EndAt:   dayStart.Add(24 * time.Hour),
				Value:   c.Value,
				Cost:    c.Cost,
			}
		}
	}

	// Convert map to slice
	result := make([]Consumption, 0, len(dailyMap))
	for _, daily := range dailyMap {
		result = append(result, *daily)
	}

	// Sort by date
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartAt.Before(result[j].StartAt)
	})

	return result
}

// detectTariffChanges identifies tariff changes within the analysis period
func (a *Analyzer) detectTariffChanges(agreements []Agreement, fuelType string) []TariffChange {
	var changes []TariffChange

	for i := 1; i < len(agreements); i++ {
		prevAgreement := agreements[i-1]
		currAgreement := agreements[i]

		rateChange := currAgreement.Tariff.UnitRate - prevAgreement.Tariff.UnitRate

		// Only report changes > 0.1p/kWh
		if math.Abs(rateChange) > 0.1 {
			impact := "increased"
			if rateChange < 0 {
				impact = "decreased"
			}

			changes = append(changes, TariffChange{
				ChangeDate:        currAgreement.ValidFrom,
				FuelType:          fuelType,
				OldTariffName:     prevAgreement.Tariff.DisplayName,
				NewTariffName:     currAgreement.Tariff.DisplayName,
				UnitRateChange:    rateChange,
				ImpactDescription: fmt.Sprintf("Unit rate %s by %.2fp/kWh", impact, math.Abs(rateChange)),
			})
		}
	}

	return changes
}

// generateInsights creates actionable recommendations
func (a *Analyzer) generateInsights(result *AnalysisResult, data *CollectedData) []Insight {
	var insights []Insight

	// Payment status insights
	if result.CurrentDirectDebit > 0 {
		if result.PaymentStatus == "Underpaying" {
			difference := result.RecommendedDirectDebit - result.CurrentDirectDebit
			insights = append(insights, Insight{
				Category:    "payment",
				Priority:    "high",
				Title:       "Direct Debit Increase Recommended",
				Description: fmt.Sprintf("Your current Direct Debit (£%.2f) is lower than recommended (£%.2f). You may build up debt over time.", result.CurrentDirectDebit, result.RecommendedDirectDebit),
				Action:      fmt.Sprintf("Consider increasing your Direct Debit by £%.2f per month", difference),
			})
		} else if result.PaymentStatus == "Overpaying" {
			difference := result.CurrentDirectDebit - result.RecommendedDirectDebit
			insights = append(insights, Insight{
				Category:    "payment",
				Priority:    "medium",
				Title:       "Direct Debit Decrease Possible",
				Description: fmt.Sprintf("Your current Direct Debit (£%.2f) is higher than needed (£%.2f). You're building up credit.", result.CurrentDirectDebit, result.RecommendedDirectDebit),
				Action:      fmt.Sprintf("Consider decreasing your Direct Debit by £%.2f per month", difference),
			})
		} else {
			insights = append(insights, Insight{
				Category:    "payment",
				Priority:    "low",
				Title:       "Direct Debit Well Balanced",
				Description: fmt.Sprintf("Your current Direct Debit (£%.2f) is appropriate for your usage", result.CurrentDirectDebit),
				Action:      "No action needed - continue monitoring your usage",
			})
		}
	}

	// Balance insights with Direct Debit context
	if result.CurrentBalance < -50 {
		insights = append(insights, Insight{
			Category:    "payment",
			Priority:    "high",
			Title:       "Account in Debit",
			Description: fmt.Sprintf("Your account has a debit balance of £%.2f", math.Abs(result.CurrentBalance)),
			Action:      "Consider making a payment or increasing your Direct Debit to clear the debt",
		})
	} else if result.CurrentBalance > 100 {
		// Calculate months of credit at current projected cost
		monthsOfCredit := result.CurrentBalance / result.ProjectedMonthlyCost

		// Suggest optimal Direct Debit with balance management
		optimalDD := result.RecommendedDirectDebit
		if result.CurrentBalance > 500 && monthsOfCredit > 6 {
			// Significant credit - suggest reducing to burn down over 12 months
			balanceReduction := result.CurrentBalance / 12 // Spread over a year
			optimalDD = result.ProjectedMonthlyCost - balanceReduction
			if optimalDD < 0 {
				optimalDD = 0
			}

			insights = append(insights, Insight{
				Category:    "payment",
				Priority:    "high",
				Title:       "High Credit Balance - Payment Adjustment Recommended",
				Description: fmt.Sprintf("Your account has £%.2f credit (%.1f months at current usage). This credit should be utilized rather than held.", result.CurrentBalance, monthsOfCredit),
				Action:      fmt.Sprintf("Consider reducing Direct Debit to £%.0f/month to gradually use your credit over 12 months, or request a partial refund of £%.0f", optimalDD, result.CurrentBalance/2),
			})
		} else {
			insights = append(insights, Insight{
				Category:    "payment",
				Priority:    "medium",
				Title:       "Credit Balance Available",
				Description: fmt.Sprintf("Your account has a credit balance of £%.2f (%.1f months coverage)", result.CurrentBalance, monthsOfCredit),
				Action:      "Consider requesting a refund or slightly reducing your Direct Debit while maintaining seasonal coverage",
			})
		}
	}

	// Recent anomaly warnings
	recentAnomalies := 0
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	for _, anomaly := range result.Anomalies {
		if anomaly.Date.After(sevenDaysAgo) && anomaly.Type != "zero_usage" {
			recentAnomalies++
		}
	}

	if recentAnomalies > 0 {
		insights = append(insights, Insight{
			Category:    "usage",
			Priority:    "medium",
			Title:       "Recent Unusual Usage Detected",
			Description: fmt.Sprintf("Detected %d unusual consumption patterns in the last 7 days", recentAnomalies),
			Action:      "Review your recent energy usage to identify any changes in consumption patterns",
		})
	}

	// Seasonal insights (winter months: November to February)
	currentMonth := time.Now().Month()
	if currentMonth >= 11 || currentMonth <= 2 {
		insights = append(insights, Insight{
			Category:    "seasonal",
			Priority:    "low",
			Title:       "Winter Usage Period",
			Description: "Currently in winter months when energy usage typically increases",
			Action:      "Monitor your usage closely as heating costs may be higher than summer averages",
		})
	}

	// Solar/Battery Export insights
	if result.AvgDailyExport > 0 {
		exportInsights := a.generateExportInsights(result, data)
		insights = append(insights, exportInsights...)
	}

	return insights
}

// generateExportInsights generates insights specific to solar/battery export performance
func (a *Analyzer) generateExportInsights(result *AnalysisResult, data *CollectedData) []Insight {
	var insights []Insight

	// Calculate key export metrics
	importTotal := result.AvgDailyElectricity
	exportTotal := result.AvgDailyExport
	netImport := importTotal - exportTotal

	// Export efficiency (% of import that's exported)
	exportRatio := 0.0
	if importTotal > 0 {
		exportRatio = (exportTotal / importTotal) * 100
	}

	// Note: Self-sufficiency rate calculation would require total generation data,
	// which we don't have directly. Export only shows excess after self-consumption.

	// Calculate financial metrics
	importCost := result.AvgDailyCostElectricity
	exportEarnings := result.AvgDailyEarningsExport
	netCost := importCost - exportEarnings
	savingsRate := 0.0
	if importCost > 0 {
		savingsRate = (exportEarnings / importCost) * 100
	}

	// Insight 1: Export Performance Summary
	if exportRatio >= 50 {
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "high",
			Title:       "Excellent Export Performance",
			Description: fmt.Sprintf("You're exporting %.1f%% of your imported electricity (%.1f kWh/day). Your solar/battery system is performing very well!", exportRatio, exportTotal),
			Action:      fmt.Sprintf("You're earning £%.2f/day from exports, offsetting %.1f%% of your import costs. Consider if you can time more usage during generation periods to increase self-consumption.", exportEarnings, savingsRate),
		})
	} else if exportRatio >= 30 {
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "medium",
			Title:       "Good Export Performance",
			Description: fmt.Sprintf("You're exporting %.1f%% of your imported electricity (%.1f kWh/day). Your system is providing good returns.", exportRatio, exportTotal),
			Action:      fmt.Sprintf("Earning £%.2f/day from exports (%.1f%% of import costs). Look for opportunities to shift more usage to daylight hours to maximize self-consumption.", exportEarnings, savingsRate),
		})
	} else {
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "medium",
			Title:       "Export Performance Review",
			Description: fmt.Sprintf("You're exporting %.1f kWh/day (%.1f%% of imports). This may indicate high self-consumption or limited generation.", exportTotal, exportRatio),
			Action:      fmt.Sprintf("Earning £%.2f/day from exports. Review if generation is meeting expectations or if system maintenance is needed.", exportEarnings),
		})
	}

	// Insight 2: Net Import Analysis
	if netImport < 5 {
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "high",
			Title:       "Near Energy Independence",
			Description: fmt.Sprintf("Your net grid import is only %.1f kWh/day! Your exports (%.1f kWh) nearly match your imports (%.1f kWh).", netImport, exportTotal, importTotal),
			Action:      "Excellent self-sufficiency! Consider battery storage optimization to further reduce grid dependency, especially during peak rate periods.",
		})
	} else if netImport < 10 {
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "medium",
			Title:       "Strong Energy Self-Sufficiency",
			Description: fmt.Sprintf("Your net grid import is %.1f kWh/day. Exports offset a significant portion of your consumption.", netImport),
			Action:      fmt.Sprintf("With %.1f kWh/day net import at £%.2f/day net cost, you're achieving good grid independence. Review battery charging patterns to reduce peak-time imports.", netImport, netCost),
		})
	}

	// Insight 3: Financial Optimization
	monthlyExportEarnings := exportEarnings * 30
	annualExportEarnings := exportEarnings * 365
	if exportEarnings > 0.50 {
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "medium",
			Title:       "Strong Export Earnings",
			Description: fmt.Sprintf("Your exports are generating £%.2f/day (£%.2f/month, ~£%.0f/year)", exportEarnings, monthlyExportEarnings, annualExportEarnings),
			Action:      fmt.Sprintf("Export earnings offset %.1f%% of your import costs. Review your export tariff rate to ensure you're getting the best rate available.", savingsRate),
		})
	}

	// Insight 4: Seasonal Considerations
	currentMonth := time.Now().Month()
	if currentMonth >= 10 || currentMonth <= 3 {
		// Winter months - lower solar generation expected
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "low",
			Title:       "Winter Export Performance",
			Description: fmt.Sprintf("Winter months typically see 50-70%% lower solar generation. Your current %.1f kWh/day export is expected to increase in spring/summer.", exportTotal),
			Action:      "Track your export performance over the coming months. Spring/summer exports should significantly increase if your system is working optimally.",
		})
	} else if currentMonth >= 4 && currentMonth <= 9 {
		// Summer months - higher solar generation expected
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "low",
			Title:       "Peak Solar Season Performance",
			Description: fmt.Sprintf("Currently in peak solar season. Your %.1f kWh/day export represents optimal generation conditions.", exportTotal),
			Action:      "This is your baseline for optimal performance. Compare winter exports to this rate to gauge seasonal variations.",
		})
	}

	// Insight 5: Grid Dependency Analysis
	gridDependencyRate := (netImport / importTotal) * 100
	if gridDependencyRate < 50 {
		insights = append(insights, Insight{
			Category:    "export",
			Priority:    "high",
			Title:       "Exceptional Grid Independence",
			Description: fmt.Sprintf("You're only %.1f%% grid-dependent! Your generation and exports mean you're mostly energy independent.", gridDependencyRate),
			Action:      "Outstanding performance! Share your setup and optimizations with the community. Consider whether additional battery capacity could reduce grid dependency further.",
		})
	}

	return insights
}

// Statistical helper functions

// calculateMean calculates the mean of a slice of float64 values
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// calculateStdDev calculates the standard deviation
func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values))
	return math.Sqrt(variance)
}

// enrichAnomaliesWithWeather fetches weather data for anomaly dates and adds context
func (a *Analyzer) enrichAnomaliesWithWeather(anomalies []Anomaly) {
	// Extract unique dates from anomalies
	dates := make([]time.Time, len(anomalies))
	for i, anomaly := range anomalies {
		dates[i] = anomaly.Date
	}

	// Fetch weather data for all anomaly dates
	weatherMap, err := a.weatherClient.FetchWeatherForDates(dates)
	if err != nil || weatherMap == nil {
		// Non-fatal - continue without weather data
		return
	}

	// Enrich each anomaly with weather data
	for i := range anomalies {
		dateKey := anomalies[i].Date.Format("2006-01-02")
		if weather, found := weatherMap[dateKey]; found {
			anomalies[i].Weather = weather
		}
	}
}

// filterWeatherExpectedAnomalies removes anomalies that are explained by weather conditions
func (a *Analyzer) filterWeatherExpectedAnomalies(anomalies []Anomaly) []Anomaly {
	filtered := make([]Anomaly, 0, len(anomalies))

	for _, anomaly := range anomalies {
		// Only filter consumption spikes, not low usage
		if anomaly.Type != "consumption_spike" {
			filtered = append(filtered, anomaly)
			continue
		}

		// If no weather data, keep the anomaly
		if anomaly.Weather == nil {
			filtered = append(filtered, anomaly)
			continue
		}

		// Check if gas consumption spike is explained by cold weather
		if anomaly.FuelType == "gas" {
			// Gas usage naturally increases in cold weather
			// If temp < 10°C, this is expected behavior, not an anomaly
			if anomaly.Weather.TempMean < 10.0 {
				a.logger.Debug("Filtering weather-expected gas spike",
					"date", anomaly.Date.Format("2006-01-02"),
					"temp", anomaly.Weather.TempMean,
					"usage", anomaly.ActualValue)
				continue // Skip this anomaly
			}
		}

		// Check if electricity consumption spike is explained by extreme weather
		if anomaly.FuelType == "electricity" {
			// Electricity may spike during very cold weather (heating) or very hot (cooling)
			// Only filter if temperature is extreme AND usage increase is proportional
			if anomaly.Weather.TempMean < 5.0 || anomaly.Weather.TempMean > 28.0 {
				// Allow moderate increases for extreme weather, but flag massive spikes
				if anomaly.DeviationPercent < 100.0 {
					a.logger.Debug("Filtering weather-expected electricity spike",
						"date", anomaly.Date.Format("2006-01-02"),
						"temp", anomaly.Weather.TempMean,
						"deviation", anomaly.DeviationPercent)
					continue // Skip this anomaly
				}
			}
		}

		// Keep this anomaly
		filtered = append(filtered, anomaly)
	}

	if len(filtered) < len(anomalies) {
		a.logger.Info("Filtered weather-expected anomalies",
			"original", len(anomalies),
			"filtered", len(filtered),
			"removed", len(anomalies)-len(filtered))
	}

	return filtered
}

// FormatCurrency formats a value as currency
func FormatCurrency(value float64) string {
	return fmt.Sprintf("£%.2f", value)
}

// FormatPercentage formats a value as a percentage
func FormatPercentage(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}
