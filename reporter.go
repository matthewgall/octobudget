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
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

// Reporter generates markdown reports from analysis results
type Reporter struct {
	logger *Logger
}

// NewReporter creates a new report generator
func NewReporter(logger *Logger) *Reporter {
	return &Reporter{
		logger: logger,
	}
}

// GenerateReport creates a markdown report from analysis results
func (r *Reporter) GenerateReport(result *AnalysisResult, outputPath string) error {
	r.logger.Info("Generating report")

	var writer io.Writer
	if outputPath == "" {
		writer = os.Stdout
	} else {
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create report file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	// Generate report content
	r.writeHeader(writer, result)
	r.writeSummary(writer, result)
	r.writePaymentAnalysis(writer, result)
	r.writeConsumptionAnalysis(writer, result)
	r.writeExportPerformance(writer, result)
	r.writeTariffInformation(writer, result)
	r.writeAnomalies(writer, result)
	r.writeTariffChanges(writer, result)
	r.writeRecommendations(writer, result)
	r.writeFooter(writer)

	if outputPath != "" {
		r.logger.Info("Report saved", "path", outputPath)
	}

	return nil
}

// writeHeader writes the report header
func (r *Reporter) writeHeader(w io.Writer, result *AnalysisResult) {
	fmt.Fprintf(w, "# Octopus Energy Budget Analysis Report\n\n")
	fmt.Fprintf(w, "**Generated:** %s\n\n", result.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "**Analysis Period:** %s to %s (%d days)\n\n",
		result.AnalysisPeriodStart.Format("2006-01-02"),
		result.AnalysisPeriodEnd.Format("2006-01-02"),
		result.AnalysisPeriodDays,
	)
	fmt.Fprintf(w, "**octobudget version:** %s\n\n", GetVersion())
	fmt.Fprintf(w, "---\n\n")
}

// writeSummary writes the summary section
func (r *Reporter) writeSummary(w io.Writer, result *AnalysisResult) {
	fmt.Fprintf(w, "## 📊 Summary\n\n")

	// Current balance with status indicator
	balanceIndicator := "✅"
	if result.CurrentBalance < -50 {
		balanceIndicator = "⚠️"
	} else if result.CurrentBalance < 0 {
		balanceIndicator = "⚡"
	}

	fmt.Fprintf(w, "**Current Account Balance:** %s %s\n\n",
		balanceIndicator,
		FormatCurrency(result.CurrentBalance),
	)

	// Average daily costs in table format
	fmt.Fprintf(w, "### 💷 Average Daily Costs\n\n")
	fmt.Fprintf(w, "| Item | Cost | Consumption |\n")
	fmt.Fprintf(w, "|------|------|-------------|\n")
	if result.AvgDailyCostElectricity > 0 {
		fmt.Fprintf(w, "| ⚡ Electricity Import | %s | %.2f kWh |\n",
			FormatCurrency(result.AvgDailyCostElectricity),
			result.AvgDailyElectricity,
		)
	}
	if result.AvgDailyEarningsExport > 0 {
		fmt.Fprintf(w, "| ☀️ Solar/Battery Export | -%s | %.2f kWh |\n",
			FormatCurrency(result.AvgDailyEarningsExport),
			result.AvgDailyExport,
		)
	}
	if result.AvgDailyCostGas > 0 {
		fmt.Fprintf(w, "| 🔥 Gas | %s | %.2f kWh |\n",
			FormatCurrency(result.AvgDailyCostGas),
			result.AvgDailyGas,
		)
	}
	fmt.Fprintf(w, "| **💰 Net Total** | **%s** | ",
		FormatCurrency(result.AvgDailyCostTotal),
	)
	if result.AvgDailyEarningsExport > 0 {
		netConsumption := result.AvgDailyElectricity - result.AvgDailyExport + result.AvgDailyGas
		fmt.Fprintf(w, "**%.2f kWh** |\n", netConsumption)
	} else {
		totalConsumption := result.AvgDailyElectricity + result.AvgDailyGas
		fmt.Fprintf(w, "**%.2f kWh** |\n", totalConsumption)
	}
	fmt.Fprintf(w, "\n")

	// Projected monthly cost in badge format
	fmt.Fprintf(w, "> **📅 Projected Monthly Cost:** %s\n\n",
		FormatCurrency(result.ProjectedMonthlyCost),
	)
}

// writePaymentAnalysis writes the payment analysis section
func (r *Reporter) writePaymentAnalysis(w io.Writer, result *AnalysisResult) {
	fmt.Fprintf(w, "## 💳 Payment Analysis\n\n")

	// Payment summary table
	fmt.Fprintf(w, "| Metric | Amount |\n")
	fmt.Fprintf(w, "|--------|--------|\n")
	fmt.Fprintf(w, "| 💰 Current Account Balance | %s |\n", FormatCurrency(result.CurrentBalance))
	fmt.Fprintf(w, "| 📊 Current Monthly Cost | %s |\n", FormatCurrency(result.ProjectedMonthlyCost))

	if result.CurrentDirectDebit > 0 {
		fmt.Fprintf(w, "| 📅 Current Direct Debit | %s |\n", FormatCurrency(result.CurrentDirectDebit))
		fmt.Fprintf(w, "| ✅ Recommended Direct Debit | %s |\n", FormatCurrency(result.RecommendedDirectDebit))

		difference := result.RecommendedDirectDebit - result.CurrentDirectDebit
		if math.Abs(difference) >= 5 {
			action := "↗️ Increase"
			if difference < 0 {
				action = "↘️ Decrease"
			}
			fmt.Fprintf(w, "| 🔄 Suggested Adjustment | %s by %s |\n", action, FormatCurrency(math.Abs(difference)))
		}
	} else {
		fmt.Fprintf(w, "| ✅ Recommended Direct Debit | %s |\n", FormatCurrency(result.RecommendedDirectDebit))
	}

	fmt.Fprintf(w, "\n")

	// Account balance analysis
	if result.CurrentBalance > 100 {
		monthsOfCredit := result.CurrentBalance / result.ProjectedMonthlyCost
		fmt.Fprintf(w, "### 💵 Credit Balance Analysis\n\n")
		fmt.Fprintf(w, "Your account holds **%s in credit**, equivalent to **%.1f months** of current usage.\n\n",
			FormatCurrency(result.CurrentBalance), monthsOfCredit)

		if result.CurrentBalance > 500 && monthsOfCredit > 6 {
			// Significant credit
			balanceReduction := result.CurrentBalance / 12
			optimalDD := result.ProjectedMonthlyCost - balanceReduction
			if optimalDD < 0 {
				optimalDD = 0
			}

			fmt.Fprintf(w, "**Options for managing your credit:**\n\n")
			fmt.Fprintf(w, "1. **Request a refund** of £%.0f (50%% of credit) and maintain current Direct Debit\n", result.CurrentBalance/2)
			fmt.Fprintf(w, "2. **Reduce Direct Debit** to £%.0f/month to gradually use credit over 12 months\n", optimalDD)
			fmt.Fprintf(w, "3. **Request full refund** of £%.0f and set new Direct Debit to £%.0f\n\n", result.CurrentBalance, result.RecommendedDirectDebit)
		}
	} else if result.CurrentBalance < -50 {
		fmt.Fprintf(w, "### ⚠️ Debit Balance Alert\n\n")
		fmt.Fprintf(w, "Your account has a **debit of %s**. Consider increasing your Direct Debit or making a one-time payment.\n\n",
			FormatCurrency(math.Abs(result.CurrentBalance)))
	}

	// Direct Debit calculation explanation
	fmt.Fprintf(w, "### 📐 How the Recommendation is Calculated\n\n")
	fmt.Fprintf(w, "The recommended Direct Debit accounts for:\n\n")
	fmt.Fprintf(w, "- **Current usage patterns** (£%.2f/day average)\n", result.AvgDailyCostTotal)
	fmt.Fprintf(w, "- **Seasonal variations** (winter: +40%%, spring/autumn: +20%%, summer: baseline)\n")
	fmt.Fprintf(w, "- **10%% buffer** for unexpected increases\n")
	fmt.Fprintf(w, "- **Year-round stability** to avoid large seasonal swings\n\n")

	currentMonth := time.Now().Month()
	if currentMonth >= 11 || currentMonth <= 2 {
		fmt.Fprintf(w, "> 🌡️ **Winter Period:** Currently in winter months when heating usage is typically 30-50%% higher. ")
		fmt.Fprintf(w, "The recommendation ensures you can cover peak winter costs while building modest credit in summer.\n\n")
	} else if currentMonth >= 5 && currentMonth <= 8 {
		fmt.Fprintf(w, "> ☀️ **Summer Period:** Currently in lower-usage summer months. ")
		fmt.Fprintf(w, "The recommendation is set to build credit now to cover higher winter costs later.\n\n")
	} else {
		fmt.Fprintf(w, "> 🍂 **Transition Period:** The recommendation balances seasonal changes ")
		fmt.Fprintf(w, "to provide stable payments year-round.\n\n")
	}
}

// writeConsumptionAnalysis writes the consumption analysis section
func (r *Reporter) writeConsumptionAnalysis(w io.Writer, result *AnalysisResult) {
	fmt.Fprintf(w, "## ⚡ Consumption Analysis\n\n")

	if result.AvgDailyElectricity == 0 && result.AvgDailyGas == 0 {
		fmt.Fprintf(w, "*No consumption data available for analysis.*\n\n")
		return
	}

	// Create metrics table
	fmt.Fprintf(w, "| Metric | Value |\n")
	fmt.Fprintf(w, "|--------|-------|\n")

	if result.AvgDailyElectricity > 0 {
		fmt.Fprintf(w, "| ⚡ Daily Electricity Import | %.2f kWh |\n", result.AvgDailyElectricity)
	}

	if result.AvgDailyExport > 0 {
		fmt.Fprintf(w, "| ☀️ Daily Solar/Battery Export | %.2f kWh |\n", result.AvgDailyExport)
		netElectricity := result.AvgDailyElectricity - result.AvgDailyExport
		fmt.Fprintf(w, "| 🔌 Net Electricity from Grid | %.2f kWh |\n", netElectricity)
	}

	if result.AvgDailyGas > 0 {
		fmt.Fprintf(w, "| 🔥 Daily Gas Usage | %.2f kWh |\n", result.AvgDailyGas)
	}

	fmt.Fprintf(w, "\n")
}

// writeExportPerformance writes the solar/battery export performance section
func (r *Reporter) writeExportPerformance(w io.Writer, result *AnalysisResult) {
	if result.AvgDailyExport == 0 {
		return // No export data, skip this section
	}

	fmt.Fprintf(w, "## ☀️ Solar/Battery Export Performance\n\n")

	// Calculate key metrics
	importTotal := result.AvgDailyElectricity
	exportTotal := result.AvgDailyExport
	netImport := importTotal - exportTotal
	exportRatio := 0.0
	if importTotal > 0 {
		exportRatio = (exportTotal / importTotal) * 100
	}

	importCost := result.AvgDailyCostElectricity
	exportEarnings := result.AvgDailyEarningsExport
	netCost := importCost - exportEarnings
	savingsRate := 0.0
	if importCost > 0 {
		savingsRate = (exportEarnings / importCost) * 100
	}

	gridDependencyRate := 0.0
	if importTotal > 0 {
		gridDependencyRate = (netImport / importTotal) * 100
	}

	// Performance summary table
	fmt.Fprintf(w, "### 📊 Performance Overview\n\n")
	fmt.Fprintf(w, "| Metric | Value |\n")
	fmt.Fprintf(w, "|--------|-------|\n")
	fmt.Fprintf(w, "| 📤 Daily Export | %.2f kWh |\n", exportTotal)
	fmt.Fprintf(w, "| 📥 Daily Import | %.2f kWh |\n", importTotal)
	fmt.Fprintf(w, "| 🔌 Net Grid Usage | %.2f kWh (%.1f%% of import) |\n", netImport, gridDependencyRate)
	fmt.Fprintf(w, "| ♻️ Export Ratio | %.1f%% of imports |\n", exportRatio)
	fmt.Fprintf(w, "\n")

	// Financial summary table
	fmt.Fprintf(w, "### 💰 Financial Impact\n\n")
	fmt.Fprintf(w, "| Period | Import Cost | Export Earnings | Net Cost | Savings |\n")
	fmt.Fprintf(w, "|--------|-------------|-----------------|----------|----------|\n")
	fmt.Fprintf(w, "| Daily | £%.2f | £%.2f | £%.2f | %.1f%% |\n",
		importCost, exportEarnings, netCost, savingsRate)
	fmt.Fprintf(w, "| Monthly | £%.2f | £%.2f | £%.2f | %.1f%% |\n",
		importCost*30, exportEarnings*30, netCost*30, savingsRate)
	fmt.Fprintf(w, "| Annual | £%.2f | £%.2f | £%.2f | %.1f%% |\n",
		importCost*365, exportEarnings*365, netCost*365, savingsRate)
	fmt.Fprintf(w, "\n")

	// Performance rating
	fmt.Fprintf(w, "### ⭐ Performance Rating\n\n")

	if exportRatio >= 50 {
		fmt.Fprintf(w, "**Excellent** ⭐⭐⭐⭐⭐\n\n")
		fmt.Fprintf(w, "Your export rate of %.1f%% is outstanding! You're exporting more than half of what you import from the grid.\n\n", exportRatio)
	} else if exportRatio >= 30 {
		fmt.Fprintf(w, "**Very Good** ⭐⭐⭐⭐\n\n")
		fmt.Fprintf(w, "Your export rate of %.1f%% shows strong system performance with good returns.\n\n", exportRatio)
	} else if exportRatio >= 15 {
		fmt.Fprintf(w, "**Good** ⭐⭐⭐\n\n")
		fmt.Fprintf(w, "Your export rate of %.1f%% indicates decent generation with room for optimization.\n\n", exportRatio)
	} else {
		fmt.Fprintf(w, "**Fair** ⭐⭐\n\n")
		fmt.Fprintf(w, "Your export rate of %.1f%% suggests either high self-consumption or potential for system improvements.\n\n", exportRatio)
	}

	if gridDependencyRate < 50 {
		fmt.Fprintf(w, "🏆 **Exceptional Grid Independence:** You're only %.1f%% dependent on the grid!\n\n", gridDependencyRate)
	} else if gridDependencyRate < 70 {
		fmt.Fprintf(w, "✅ **Strong Self-Sufficiency:** %.1f%% grid dependency shows good energy independence.\n\n", gridDependencyRate)
	}

	if savingsRate >= 40 {
		fmt.Fprintf(w, "💚 **High Financial Benefit:** Exports offset %.1f%% of your import costs - excellent ROI!\n\n", savingsRate)
	}
}

// writeTariffInformation writes the detected tariff information section
func (r *Reporter) writeTariffInformation(w io.Writer, result *AnalysisResult) {
	hasAnyTariff := len(result.ElectricityAgreements) > 0 ||
		len(result.ElectricityExportAgreements) > 0 ||
		len(result.GasAgreements) > 0

	if !hasAnyTariff {
		return // No tariff data, skip this section
	}

	fmt.Fprintf(w, "## 📋 Detected Tariffs\n\n")

	// Electricity Import Tariff
	if len(result.ElectricityAgreements) > 0 {
		fmt.Fprintf(w, "### ⚡ Electricity Import\n\n")

		// Show all agreements (current and upcoming)
		for i, agreement := range result.ElectricityAgreements {
			if i > 0 {
				fmt.Fprintf(w, "---\n\n")
			}

			fmt.Fprintf(w, "**Tariff:** %s\n\n", agreement.Tariff.DisplayName)

		if agreement.Tariff.FullName != "" && agreement.Tariff.FullName != agreement.Tariff.DisplayName {
			fmt.Fprintf(w, "**Full Name:** %s\n\n", agreement.Tariff.FullName)
		}

		fmt.Fprintf(w, "| Component | Rate |\n")
		fmt.Fprintf(w, "|-----------|------|\n")
		fmt.Fprintf(w, "| 💰 Standing Charge | %.2fp per day |\n", agreement.Tariff.StandingCharge)

		hasAnyRate := agreement.Tariff.UnitRate > 0 || agreement.Tariff.DayRate > 0 ||
			agreement.Tariff.NightRate > 0 || agreement.Tariff.OffPeakRate > 0

		if agreement.Tariff.UnitRate > 0 {
			fmt.Fprintf(w, "| ⚡ Unit Rate | %.2fp per kWh |\n", agreement.Tariff.UnitRate)
		}
		if agreement.Tariff.DayRate > 0 {
			fmt.Fprintf(w, "| 🌞 Day Rate | %.2fp per kWh |\n", agreement.Tariff.DayRate)
		}
		if agreement.Tariff.NightRate > 0 {
			fmt.Fprintf(w, "| 🌙 Night Rate | %.2fp per kWh |\n", agreement.Tariff.NightRate)
		}
		if agreement.Tariff.OffPeakRate > 0 {
			fmt.Fprintf(w, "| 🔋 Off-Peak Rate | %.2fp per kWh |\n", agreement.Tariff.OffPeakRate)
		}

			if !hasAnyRate {
				fmt.Fprintf(w, "| ⚡ Unit Rate | *Time-varying (see costs in analysis)* |\n")
			}

			fmt.Fprintf(w, "\n**Valid From:** %s", agreement.ValidFrom.Format("2006-01-02"))
			if agreement.ValidTo != nil {
				fmt.Fprintf(w, " **to** %s", agreement.ValidTo.Format("2006-01-02"))
			}
			fmt.Fprintf(w, "\n\n")
		}
	}

	// Electricity Export Tariff
	if len(result.ElectricityExportAgreements) > 0 {
		fmt.Fprintf(w, "### ☀️ Electricity Export\n\n")

		// Show all agreements (current and upcoming)
		for i, agreement := range result.ElectricityExportAgreements {
			if i > 0 {
				fmt.Fprintf(w, "---\n\n")
			}

			fmt.Fprintf(w, "**Tariff:** %s\n\n", agreement.Tariff.DisplayName)

		if agreement.Tariff.FullName != "" && agreement.Tariff.FullName != agreement.Tariff.DisplayName {
			fmt.Fprintf(w, "**Full Name:** %s\n\n", agreement.Tariff.FullName)
		}

		fmt.Fprintf(w, "| Component | Rate |\n")
		fmt.Fprintf(w, "|-----------|------|\n")

		hasAnyExportRate := agreement.Tariff.UnitRate > 0 || agreement.Tariff.DayRate > 0 ||
			agreement.Tariff.NightRate > 0 || agreement.Tariff.OffPeakRate > 0

		if agreement.Tariff.UnitRate > 0 {
			fmt.Fprintf(w, "| ☀️ Export Rate | %.2fp per kWh |\n", agreement.Tariff.UnitRate)
		}
		if agreement.Tariff.DayRate > 0 {
			fmt.Fprintf(w, "| 🌞 Day Export Rate | %.2fp per kWh |\n", agreement.Tariff.DayRate)
		}
		if agreement.Tariff.NightRate > 0 {
			fmt.Fprintf(w, "| 🌙 Night Export Rate | %.2fp per kWh |\n", agreement.Tariff.NightRate)
		}
		if agreement.Tariff.OffPeakRate > 0 {
			fmt.Fprintf(w, "| 🔋 Off-Peak Export Rate | %.2fp per kWh |\n", agreement.Tariff.OffPeakRate)
		}

			if !hasAnyExportRate {
				fmt.Fprintf(w, "| ☀️ Export Rate | *Time-varying (see earnings in analysis)* |\n")
			}

			fmt.Fprintf(w, "\n**Valid From:** %s", agreement.ValidFrom.Format("2006-01-02"))
			if agreement.ValidTo != nil {
				fmt.Fprintf(w, " **to** %s", agreement.ValidTo.Format("2006-01-02"))
			}
			fmt.Fprintf(w, "\n\n")
		}
	}

	// Gas Tariff
	if len(result.GasAgreements) > 0 {
		fmt.Fprintf(w, "### 🔥 Gas\n\n")

		// Show all agreements (current and upcoming)
		for i, agreement := range result.GasAgreements {
			if i > 0 {
				fmt.Fprintf(w, "---\n\n")
			}

			fmt.Fprintf(w, "**Tariff:** %s\n\n", agreement.Tariff.DisplayName)

		if agreement.Tariff.FullName != "" && agreement.Tariff.FullName != agreement.Tariff.DisplayName {
			fmt.Fprintf(w, "**Full Name:** %s\n\n", agreement.Tariff.FullName)
		}

		fmt.Fprintf(w, "| Component | Rate |\n")
		fmt.Fprintf(w, "|-----------|------|\n")
		fmt.Fprintf(w, "| 💰 Standing Charge | %.2fp per day |\n", agreement.Tariff.StandingCharge)

			if agreement.Tariff.UnitRate > 0 {
				fmt.Fprintf(w, "| 🔥 Unit Rate | %.2fp per kWh |\n", agreement.Tariff.UnitRate)
			}

			fmt.Fprintf(w, "\n**Valid From:** %s", agreement.ValidFrom.Format("2006-01-02"))
			if agreement.ValidTo != nil {
				fmt.Fprintf(w, " **to** %s", agreement.ValidTo.Format("2006-01-02"))
			}
			fmt.Fprintf(w, "\n\n")
		}
	}
}

// writeAnomalies writes the anomalies section (showing top 10 most significant)
func (r *Reporter) writeAnomalies(w io.Writer, result *AnalysisResult) {
	if len(result.Anomalies) == 0 {
		return
	}

	fmt.Fprintf(w, "## 🔍 Anomalies Detected\n\n")

	totalAnomalies := len(result.Anomalies)

	// Sort anomalies by deviation (most significant first)
	sortedAnomalies := make([]Anomaly, len(result.Anomalies))
	copy(sortedAnomalies, result.Anomalies)
	sort.Slice(sortedAnomalies, func(i, j int) bool {
		return math.Abs(sortedAnomalies[i].DeviationPercent) > math.Abs(sortedAnomalies[j].DeviationPercent)
	})

	// Limit to top 10 most significant anomalies
	displayCount := 10
	if totalAnomalies < displayCount {
		displayCount = totalAnomalies
	}

	if totalAnomalies > displayCount {
		fmt.Fprintf(w, "Found **%d anomalies** in your consumption data. Showing the **top %d most significant**:\n\n", totalAnomalies, displayCount)
	} else {
		fmt.Fprintf(w, "Found **%d anomalies** in your consumption data:\n\n", totalAnomalies)
	}

	// Create anomalies table
	fmt.Fprintf(w, "| Date | Fuel | Type | Actual | Expected | Deviation | Weather |\n")
	fmt.Fprintf(w, "|------|------|------|--------|----------|-----------|----------|\n")

	for i := 0; i < displayCount; i++ {
		anomaly := sortedAnomalies[i]

		// Determine icon and direction based on type
		typeIcon := "⚠️"
		direction := "↑"
		if anomaly.Type == "low_usage" {
			typeIcon = "🔵"
			direction = "↓"
		}

		// Determine fuel icon
		fuelIcon := "⚡"
		switch anomaly.FuelType {
		case "gas":
			fuelIcon = "🔥"
		case "export":
			fuelIcon = "☀️"
		}

		typeDesc := strings.ReplaceAll(anomaly.Type, "_", " ")

		weather := "-"
		if anomaly.Weather != nil {
			weather = fmt.Sprintf("%s, %.1f°C", anomaly.Weather.WeatherDesc, anomaly.Weather.TempMean)
			if anomaly.Weather.Precipitation > 0 {
				weather += fmt.Sprintf(", %.1fmm", anomaly.Weather.Precipitation)
			}
		}

		fmt.Fprintf(w, "| %s %s | %s | %s %s %s | %.2f kWh | %.2f kWh | %s | %s |\n",
			typeIcon,
			anomaly.Date.Format("2006-01-02"),
			fuelIcon,
			typeIcon,
			direction,
			typeDesc,
			anomaly.ActualValue,
			anomaly.ExpectedValue,
			FormatPercentage(anomaly.DeviationPercent),
			weather,
		)
	}
	fmt.Fprintf(w, "\n")
}

// writeTariffChanges writes the tariff changes section
func (r *Reporter) writeTariffChanges(w io.Writer, result *AnalysisResult) {
	if len(result.TariffChanges) == 0 {
		return
	}

	fmt.Fprintf(w, "## Tariff Changes\n\n")
	fmt.Fprintf(w, "Detected **%d tariff changes** during the analysis period:\n\n", len(result.TariffChanges))

	for _, change := range result.TariffChanges {
		changeIcon := "📈"
		if change.UnitRateChange < 0 {
			changeIcon = "📉"
		}

		fmt.Fprintf(w, "### %s %s - %s\n\n",
			changeIcon,
			change.ChangeDate.Format("2006-01-02"),
			strings.Title(change.FuelType),
		)
		fmt.Fprintf(w, "- **Old Tariff:** %s\n", change.OldTariffName)
		fmt.Fprintf(w, "- **New Tariff:** %s\n", change.NewTariffName)
		fmt.Fprintf(w, "- **Impact:** %s\n\n", change.ImpactDescription)
	}
}

// writeRecommendations writes the recommendations section
func (r *Reporter) writeRecommendations(w io.Writer, result *AnalysisResult) {
	if len(result.Insights) == 0 {
		return
	}

	fmt.Fprintf(w, "## Recommendations\n\n")

	// Separate export insights from general insights
	exportInsights := []Insight{}
	generalInsights := []Insight{}

	for _, insight := range result.Insights {
		if insight.Category == "export" {
			exportInsights = append(exportInsights, insight)
		} else {
			generalInsights = append(generalInsights, insight)
		}
	}

	// Write export insights first if any
	if len(exportInsights) > 0 {
		fmt.Fprintf(w, "### ☀️ Solar/Battery Export Insights\n\n")

		// Group export insights by priority
		highExport := []Insight{}
		mediumExport := []Insight{}
		lowExport := []Insight{}

		for _, insight := range exportInsights {
			switch insight.Priority {
			case "high":
				highExport = append(highExport, insight)
			case "medium":
				mediumExport = append(mediumExport, insight)
			case "low":
				lowExport = append(lowExport, insight)
			}
		}

		for _, insight := range highExport {
			r.writeInsight(w, insight)
		}
		for _, insight := range mediumExport {
			r.writeInsight(w, insight)
		}
		for _, insight := range lowExport {
			r.writeInsight(w, insight)
		}
	}

	// Write general insights grouped by priority
	if len(generalInsights) > 0 {
		if len(exportInsights) > 0 {
			fmt.Fprintf(w, "### 💡 General Insights\n\n")
		}

		// Group insights by priority
		highPriority := []Insight{}
		mediumPriority := []Insight{}
		lowPriority := []Insight{}

		for _, insight := range generalInsights {
			switch insight.Priority {
			case "high":
				highPriority = append(highPriority, insight)
			case "medium":
				mediumPriority = append(mediumPriority, insight)
			case "low":
				lowPriority = append(lowPriority, insight)
			}
		}

		// Write high priority insights
		if len(highPriority) > 0 {
			fmt.Fprintf(w, "#### 🔴 High Priority\n\n")
			for _, insight := range highPriority {
				r.writeInsight(w, insight)
			}
		}

		// Write medium priority insights
		if len(mediumPriority) > 0 {
			fmt.Fprintf(w, "#### 🟡 Medium Priority\n\n")
			for _, insight := range mediumPriority {
				r.writeInsight(w, insight)
			}
		}

		// Write low priority insights
		if len(lowPriority) > 0 {
			fmt.Fprintf(w, "#### 🔵 Low Priority\n\n")
			for _, insight := range lowPriority {
				r.writeInsight(w, insight)
			}
		}
	}
}

// writeInsight writes a single insight
func (r *Reporter) writeInsight(w io.Writer, insight Insight) {
	fmt.Fprintf(w, "#### %s\n\n", insight.Title)
	fmt.Fprintf(w, "%s\n\n", insight.Description)
	fmt.Fprintf(w, "**Recommended Action:** %s\n\n", insight.Action)
}

// writeFooter writes the report footer
func (r *Reporter) writeFooter(w io.Writer) {
	fmt.Fprintf(w, "---\n\n")
	fmt.Fprintf(w, "*This report is based on historical data and projections may vary based on seasonal changes, tariff adjustments, and usage patterns. Please review your actual bills and account statements for precise information.*\n\n")
	fmt.Fprintf(w, "*Generated by [octobudget](https://github.com/matthewgall/octobudget)*\n\n")
	fmt.Fprintf(w, "---\n\n")
	fmt.Fprintf(w, "This is an unofficial third-party application. \"Octopus Energy\" is a trademark of Octopus Energy Group Limited. This application is not affiliated with, endorsed by, or connected to Octopus Energy.\n")
}
