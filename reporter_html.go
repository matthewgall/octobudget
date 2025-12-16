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
	"html"
	"io"
	"math"
	"os"
	"sort"
	"time"
)

// HTMLReporter generates HTML reports from analysis results
type HTMLReporter struct {
	logger *Logger
}

// NewHTMLReporter creates a new HTML report generator
func NewHTMLReporter(logger *Logger) *HTMLReporter {
	return &HTMLReporter{
		logger: logger,
	}
}

// GenerateHTMLReport generates an HTML report
func (r *HTMLReporter) GenerateHTMLReport(result *AnalysisResult, outputPath string) error {
	r.logger.Info("Generating HTML report")

	var writer io.Writer
	if outputPath == "" {
		writer = os.Stdout
	} else {
		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create HTML report file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	// Generate HTML report content
	r.writeHTMLHeader(writer, result)
	r.writeHTMLSummary(writer, result)
	r.writeHTMLPaymentAnalysis(writer, result)
	r.writeHTMLConsumptionAnalysis(writer, result)
	r.writeHTMLExportPerformance(writer, result)
	r.writeHTMLTariffInformation(writer, result)
	r.writeHTMLAnomalies(writer, result)
	r.writeHTMLRecommendations(writer, result)
	r.writeHTMLFooter(writer)

	if outputPath != "" {
		r.logger.Info("HTML report saved", "path", outputPath)
	}

	return nil
}

func (r *HTMLReporter) writeHTMLHeader(w io.Writer, result *AnalysisResult) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Octopus Energy Budget Analysis Report</title>
    <style>
        :root {
            --primary-color: #FF006E;
            --secondary-color: #00C896;
            --warning-color: #FFB800;
            --danger-color: #FF006E;
            --success-color: #00C896;
            --bg-color: #0A0F1E;
            --card-bg: #1A2332;
            --text-color: #E8EAF6;
            --text-muted: #9FA8DA;
            --border-color: #2A3550;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: var(--bg-color);
            color: var(--text-color);
            line-height: 1.6;
            padding: 20px;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        
        header {
            background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
            padding: 40px;
            border-radius: 16px;
            margin-bottom: 30px;
            box-shadow: 0 8px 32px rgba(255, 0, 110, 0.2);
        }
        
        h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
            font-weight: 700;
        }
        
        .subtitle {
            color: rgba(255, 255, 255, 0.9);
            font-size: 1.1em;
        }
        
        .card {
            background: var(--card-bg);
            border-radius: 12px;
            padding: 30px;
            margin-bottom: 30px;
            border: 1px solid var(--border-color);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
        }
        
        h2 {
            color: var(--primary-color);
            margin-bottom: 20px;
            font-size: 1.8em;
            border-bottom: 2px solid var(--border-color);
            padding-bottom: 10px;
        }
        
        h3 {
            color: var(--secondary-color);
            margin: 25px 0 15px 0;
            font-size: 1.4em;
        }
        
        h4 {
            color: var(--text-color);
            margin: 20px 0 10px 0;
            font-size: 1.2em;
        }
        
        table {
            width: 100%%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
        }
        
        th {
            background: rgba(255, 0, 110, 0.1);
            color: var(--primary-color);
            font-weight: 600;
        }
        
        tr:hover {
            background: rgba(0, 200, 150, 0.05);
        }
        
        .metric-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        
        .metric-card {
            background: rgba(255, 0, 110, 0.05);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 20px;
            text-align: center;
        }
        
        .metric-value {
            font-size: 2em;
            font-weight: bold;
            color: var(--secondary-color);
            margin: 10px 0;
        }
        
        .metric-label {
            color: var(--text-muted);
            font-size: 0.9em;
        }
        
        .badge {
            display: inline-block;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: 600;
            margin: 5px;
        }
        
        .badge-success {
            background: var(--success-color);
            color: white;
        }
        
        .badge-warning {
            background: var(--warning-color);
            color: #0A0F1E;
        }
        
        .badge-danger {
            background: var(--danger-color);
            color: white;
        }
        
        .badge-info {
            background: #3F51B5;
            color: white;
        }
        
        .rating {
            font-size: 2em;
            margin: 15px 0;
        }
        
        .insight-box {
            background: rgba(0, 200, 150, 0.05);
            border-left: 4px solid var(--secondary-color);
            padding: 20px;
            margin: 15px 0;
            border-radius: 4px;
        }
        
        .insight-box.high {
            border-left-color: var(--danger-color);
            background: rgba(255, 0, 110, 0.05);
        }
        
        .insight-box.medium {
            border-left-color: var(--warning-color);
            background: rgba(255, 184, 0, 0.05);
        }
        
        .insight-title {
            font-weight: 600;
            color: var(--text-color);
            margin-bottom: 10px;
        }
        
        .insight-action {
            background: rgba(255, 255, 255, 0.05);
            padding: 10px;
            border-radius: 4px;
            margin-top: 10px;
            font-style: italic;
        }
        
        .blockquote {
            border-left: 4px solid var(--primary-color);
            padding: 10px;
            margin: 20px 0;
            background: rgba(255, 0, 110, 0.05);
            border-radius: 10px;
        }
        
        .progress-bar {
            width: 100%%;
            height: 30px;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 15px;
            overflow: hidden;
            margin: 10px 0;
        }
        
        .progress-fill {
            height: 100%%;
            background: linear-gradient(90deg, var(--primary-color), var(--secondary-color));
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: 600;
            transition: width 0.5s ease;
        }
        
        footer {
            text-align: center;
            padding: 30px;
            color: var(--text-muted);
            border-top: 1px solid var(--border-color);
            margin-top: 40px;
        }
        
        @media (max-width: 768px) {
            body {
                padding: 10px;
            }
            
            header {
                padding: 20px;
            }
            
            h1 {
                font-size: 1.8em;
            }
            
            .card {
                padding: 20px;
            }
            
            table {
                font-size: 0.9em;
            }
        }
        
        @media print {
            body {
                background: white;
                color: black;
            }
            
            .card {
                border: 1px solid #ddd;
                break-inside: avoid;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>⚡ Octopus Energy Budget Analysis</h1>
            <div class="subtitle">Generated: %s</div>
            <div class="subtitle">Analysis Period: %s to %s (%d days)</div>
            <div class="subtitle" style="opacity: 0.7; font-size: 0.9em; margin-top: 10px;">octobudget %s</div>
        </header>
`,
		result.GeneratedAt.Format("Monday, 2 January 2006 at 15:04"),
		result.AnalysisPeriodStart.Format("2 Jan 2006"),
		result.AnalysisPeriodEnd.Format("2 Jan 2006"),
		result.AnalysisPeriodDays,
		GetVersion(),
	)
}

func (r *HTMLReporter) writeHTMLSummary(w io.Writer, result *AnalysisResult) {
	balanceStatus := "success"
	balanceIcon := "✅"
	if result.CurrentBalance < 0 {
		balanceStatus = "danger"
		balanceIcon = "⚠️"
	}

	fmt.Fprintf(w, `
        <div class="card">
            <h2>📊 Summary</h2>
            
            <div class="metric-grid">
                <div class="metric-card">
                    <div class="metric-label">Current Account Balance</div>
                    <div class="metric-value">%s %s</div>
                    <span class="badge badge-%s">%s</span>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Net Daily Cost</div>
                    <div class="metric-value">%s</div>
                    <span class="badge badge-info">After Exports</span>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Projected Monthly</div>
                    <div class="metric-value">%s</div>
                    <span class="badge badge-info">30-Day Estimate</span>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Recommended Direct Debit</div>
                    <div class="metric-value">%s</div>
                    <span class="badge badge-success">Seasonal Adjusted</span>
                </div>
            </div>
            
            <h3>💷 Daily Cost Breakdown</h3>
            <table>
                <thead>
                    <tr>
                        <th>Item</th>
                        <th>Cost</th>
                        <th>Consumption</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>⚡ Electricity Import</td>
                        <td>%s</td>
                        <td>%.2f kWh</td>
                    </tr>
                    <tr>
                        <td>☀️ Solar/Battery Export</td>
                        <td style="color: var(--success-color)">-%s</td>
                        <td>%.2f kWh</td>
                    </tr>
                    <tr>
                        <td>🔥 Gas</td>
                        <td>%s</td>
                        <td>%.2f kWh</td>
                    </tr>
                    <tr style="font-weight: bold; background: rgba(0, 200, 150, 0.1);">
                        <td>💰 Net Total</td>
                        <td>%s</td>
                        <td>%.2f kWh</td>
                    </tr>
                </tbody>
            </table>
        </div>
`,
		balanceIcon,
		FormatCurrency(result.CurrentBalance),
		balanceStatus,
		func() string {
			if result.CurrentBalance >= 0 {
				return "Credit"
			}
			return "Debit"
		}(),
		FormatCurrency(result.AvgDailyCostTotal),
		FormatCurrency(result.ProjectedMonthlyCost),
		FormatCurrency(result.RecommendedDirectDebit),
		FormatCurrency(result.AvgDailyCostElectricity),
		result.AvgDailyElectricity,
		FormatCurrency(result.AvgDailyEarningsExport),
		result.AvgDailyExport,
		FormatCurrency(result.AvgDailyCostGas),
		result.AvgDailyGas,
		FormatCurrency(result.AvgDailyCostTotal),
		result.AvgDailyElectricity-result.AvgDailyExport+result.AvgDailyGas,
	)
}

func (r *HTMLReporter) writeHTMLPaymentAnalysis(w io.Writer, result *AnalysisResult) {
	fmt.Fprintf(w, `
        <div class="card">
            <h2>💳 Payment Analysis</h2>
`)

	if result.CurrentBalance > 100 {
		monthsOfCredit := result.CurrentBalance / result.ProjectedMonthlyCost
		fmt.Fprintf(w, `
            <div class="blockquote">
                <h3>💵 Credit Balance Analysis</h3>
                <p>Your account holds <strong>%s in credit</strong>, equivalent to <strong>%.1f months</strong> of current usage.</p>
            </div>
`,
			FormatCurrency(result.CurrentBalance),
			monthsOfCredit,
		)

		if result.CurrentBalance > 500 && monthsOfCredit > 6 {
			balanceReduction := result.CurrentBalance / 12
			optimalDD := result.ProjectedMonthlyCost - balanceReduction
			if optimalDD < 0 {
				optimalDD = 0
			}

			fmt.Fprintf(w, `
            <h4>Options for managing your credit:</h4>
            <ol>
                <li><strong>Request a refund</strong> of %s (50%% of credit) and maintain current Direct Debit</li>
                <li><strong>Reduce Direct Debit</strong> to %s/month to gradually use credit over 12 months</li>
                <li><strong>Request full refund</strong> of %s and set new Direct Debit to %s</li>
            </ol>
`,
				FormatCurrency(result.CurrentBalance/2),
				FormatCurrency(optimalDD),
				FormatCurrency(result.CurrentBalance),
				FormatCurrency(result.RecommendedDirectDebit),
			)
		}
	}

	fmt.Fprintf(w, `
            <h3>📐 How the Recommendation is Calculated</h3>
            <p>The recommended Direct Debit of <strong>%s/month</strong> accounts for:</p>
            <ul>
                <li><strong>Current usage patterns</strong> (£%.2f/day average)</li>
                <li><strong>Seasonal variations</strong> (winter: +40%%, spring/autumn: +20%%, summer: baseline)</li>
                <li><strong>10%% buffer</strong> for unexpected increases</li>
                <li><strong>Year-round stability</strong> to avoid large seasonal swings</li>
            </ul>
`,
		FormatCurrency(result.RecommendedDirectDebit),
		result.AvgDailyCostTotal,
	)

	currentMonth := time.Now().Month()
	if currentMonth >= 11 || currentMonth <= 2 {
		fmt.Fprintf(w, `
            <div class="blockquote">
                🌡️ <strong>Winter Period:</strong> Currently in winter months when heating usage is typically 30-50%% higher. 
                The recommendation ensures you can cover peak winter costs while building modest credit in summer.
            </div>
`)
	}

	fmt.Fprintf(w, `
        </div>
`)
}

func (r *HTMLReporter) writeHTMLConsumptionAnalysis(w io.Writer, result *AnalysisResult) {
	fmt.Fprintf(w, `
        <div class="card">
            <h2>⚡ Consumption Analysis</h2>
            <table>
                <thead>
                    <tr>
                        <th>Metric</th>
                        <th>Value</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>⚡ Daily Electricity Import</td>
                        <td>%.2f kWh</td>
                    </tr>
`,
		result.AvgDailyElectricity,
	)

	if result.AvgDailyExport > 0 {
		fmt.Fprintf(w, `
                    <tr>
                        <td>☀️ Daily Solar/Battery Export</td>
                        <td>%.2f kWh</td>
                    </tr>
                    <tr>
                        <td>🔌 Net Electricity from Grid</td>
                        <td>%.2f kWh</td>
                    </tr>
`,
			result.AvgDailyExport,
			result.AvgDailyElectricity-result.AvgDailyExport,
		)
	}

	fmt.Fprintf(w, `
                    <tr>
                        <td>🔥 Daily Gas Usage</td>
                        <td>%.2f kWh</td>
                    </tr>
                </tbody>
            </table>
        </div>
`,
		result.AvgDailyGas,
	)
}

func (r *HTMLReporter) writeHTMLExportPerformance(w io.Writer, result *AnalysisResult) {
	if result.AvgDailyExport == 0 {
		return
	}

	importTotal := result.AvgDailyElectricity
	exportTotal := result.AvgDailyExport
	netImport := importTotal - exportTotal
	exportRatio := 0.0
	if importTotal > 0 {
		exportRatio = (exportTotal / importTotal) * 100
	}

	gridDependency := 0.0
	if importTotal > 0 {
		gridDependency = (netImport / importTotal) * 100
	}

	savingsRate := 0.0
	if result.AvgDailyCostElectricity > 0 {
		savingsRate = (result.AvgDailyEarningsExport / result.AvgDailyCostElectricity) * 100
	}

	fmt.Fprintf(w, `
        <div class="card">
            <h2>☀️ Solar/Battery Export Performance</h2>
            
            <h3>📊 Performance Overview</h3>
            <div class="metric-grid">
                <div class="metric-card">
                    <div class="metric-label">Export Ratio</div>
                    <div class="metric-value">%.1f%%</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: %.1f%%">%.1f%%</div>
                    </div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Grid Independence</div>
                    <div class="metric-value">%.1f%%</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: %.1f%%">%.1f%%</div>
                    </div>
                </div>
            </div>
            
            <h3>💰 Financial Impact</h3>
            <table>
                <thead>
                    <tr>
                        <th>Period</th>
                        <th>Import Cost</th>
                        <th>Export Earnings</th>
                        <th>Net Cost</th>
                        <th>Savings</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Daily</td>
                        <td>%s</td>
                        <td style="color: var(--success-color)">%s</td>
                        <td>%s</td>
                        <td>%.1f%%</td>
                    </tr>
                    <tr>
                        <td>Monthly</td>
                        <td>%s</td>
                        <td style="color: var(--success-color)">%s</td>
                        <td>%s</td>
                        <td>%.1f%%</td>
                    </tr>
                    <tr>
                        <td>Annual</td>
                        <td>%s</td>
                        <td style="color: var(--success-color)">%s</td>
                        <td>%s</td>
                        <td>%.1f%%</td>
                    </tr>
                </tbody>
            </table>
            
            <h3>⭐ Performance Rating</h3>
`,
		exportRatio,
		math.Min(exportRatio, 100),
		exportRatio,
		100-gridDependency,
		100-gridDependency,
		100-gridDependency,
		FormatCurrency(result.AvgDailyCostElectricity),
		FormatCurrency(result.AvgDailyEarningsExport),
		FormatCurrency(result.AvgDailyCostElectricity-result.AvgDailyEarningsExport),
		savingsRate,
		FormatCurrency(result.AvgDailyCostElectricity*30),
		FormatCurrency(result.AvgDailyEarningsExport*30),
		FormatCurrency((result.AvgDailyCostElectricity-result.AvgDailyEarningsExport)*30),
		savingsRate,
		FormatCurrency(result.AvgDailyCostElectricity*365),
		FormatCurrency(result.AvgDailyEarningsExport*365),
		FormatCurrency((result.AvgDailyCostElectricity-result.AvgDailyEarningsExport)*365),
		savingsRate,
	)

	rating := ""
	ratingText := ""
	if exportRatio >= 50 {
		rating = "⭐⭐⭐⭐⭐"
		ratingText = "Excellent"
	} else if exportRatio >= 30 {
		rating = "⭐⭐⭐⭐"
		ratingText = "Very Good"
	} else if exportRatio >= 15 {
		rating = "⭐⭐⭐"
		ratingText = "Good"
	} else {
		rating = "⭐⭐"
		ratingText = "Fair"
	}

	fmt.Fprintf(w, `
            <div class="rating">%s</div>
            <p><strong>%s</strong> - Your export rate of %.1f%% shows strong performance.</p>
`,
		rating,
		ratingText,
		exportRatio,
	)

	if gridDependency < 50 {
		fmt.Fprintf(w, `
            <div class="blockquote">
                🏆 <strong>Exceptional Grid Independence:</strong> You're only %.1f%% dependent on the grid!
            </div>
`,
			gridDependency,
		)
	}

	if savingsRate >= 40 {
		fmt.Fprintf(w, `
            <div class="blockquote">
                💚 <strong>High Financial Benefit:</strong> Exports offset %.1f%% of your import costs - excellent ROI!
            </div>
`,
			savingsRate,
		)
	}

	fmt.Fprintf(w, `
        </div>
`)
}

func (r *HTMLReporter) writeHTMLTariffInformation(w io.Writer, result *AnalysisResult) {
	hasAnyTariff := len(result.ElectricityAgreements) > 0 ||
		len(result.ElectricityExportAgreements) > 0 ||
		len(result.GasAgreements) > 0

	if !hasAnyTariff {
		return // No tariff data, skip this section
	}

	fmt.Fprintf(w, `
        <div class="card">
            <h2>📋 Detected Tariffs</h2>
`)

	// Electricity Import Tariff
	if len(result.ElectricityAgreements) > 0 {
		fmt.Fprintf(w, `
            <h3>⚡ Electricity Import</h3>
`)

		// Show all agreements (current and upcoming)
		for i, agreement := range result.ElectricityAgreements {
			if i > 0 {
				fmt.Fprintf(w, `
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--border-color);">
`)
			}

			fmt.Fprintf(w, `
            <p><strong>Tariff:</strong> %s</p>
`,
				html.EscapeString(agreement.Tariff.DisplayName),
			)

		if agreement.Tariff.FullName != "" && agreement.Tariff.FullName != agreement.Tariff.DisplayName {
			fmt.Fprintf(w, `
            <p><strong>Full Name:</strong> %s</p>
`,
				html.EscapeString(agreement.Tariff.FullName),
			)
		}

		fmt.Fprintf(w, `
            <table>
                <tbody>
                    <tr>
                        <td>💰 Standing Charge</td>
                        <td>%.2fp per day</td>
                    </tr>
`,
			agreement.Tariff.StandingCharge,
		)

		hasAnyRate := agreement.Tariff.UnitRate > 0 || agreement.Tariff.DayRate > 0 ||
			agreement.Tariff.NightRate > 0 || agreement.Tariff.OffPeakRate > 0

		if agreement.Tariff.UnitRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>⚡ Unit Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.UnitRate,
			)
		}
		if agreement.Tariff.DayRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>🌞 Day Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.DayRate,
			)
		}
		if agreement.Tariff.NightRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>🌙 Night Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.NightRate,
			)
		}
		if agreement.Tariff.OffPeakRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>🔋 Off-Peak Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.OffPeakRate,
			)
		}

			if !hasAnyRate {
				fmt.Fprintf(w, `
                    <tr>
                        <td>⚡ Unit Rate</td>
                        <td><em>Time-varying (see costs in analysis)</em></td>
                    </tr>
`)
			}

			validityPeriod := fmt.Sprintf("Valid From: %s", agreement.ValidFrom.Format("2006-01-02"))
			if agreement.ValidTo != nil {
				validityPeriod += fmt.Sprintf(" to %s", agreement.ValidFrom.Format("2006-01-02"))
			}

			fmt.Fprintf(w, `
                </tbody>
            </table>
            <p style="margin-top: 10px; opacity: 0.7;"><em>%s</em></p>
`,
				html.EscapeString(validityPeriod),
			)
		}
	}

	// Electricity Export Tariff
	if len(result.ElectricityExportAgreements) > 0 {
		fmt.Fprintf(w, `
            <h3 style="margin-top: 30px;">☀️ Electricity Export</h3>
`)

		// Show all agreements (current and upcoming)
		for i, agreement := range result.ElectricityExportAgreements {
			if i > 0 {
				fmt.Fprintf(w, `
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--border-color);">
`)
			}

			fmt.Fprintf(w, `
            <p><strong>Tariff:</strong> %s</p>
`,
				html.EscapeString(agreement.Tariff.DisplayName),
			)

		if agreement.Tariff.FullName != "" && agreement.Tariff.FullName != agreement.Tariff.DisplayName {
			fmt.Fprintf(w, `
            <p><strong>Full Name:</strong> %s</p>
`,
				html.EscapeString(agreement.Tariff.FullName),
			)
		}

		fmt.Fprintf(w, `
            <table>
                <tbody>
`)

		hasAnyExportRate := agreement.Tariff.UnitRate > 0 || agreement.Tariff.DayRate > 0 ||
			agreement.Tariff.NightRate > 0 || agreement.Tariff.OffPeakRate > 0

		if agreement.Tariff.UnitRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>☀️ Export Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.UnitRate,
			)
		}
		if agreement.Tariff.DayRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>🌞 Day Export Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.DayRate,
			)
		}
		if agreement.Tariff.NightRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>🌙 Night Export Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.NightRate,
			)
		}
		if agreement.Tariff.OffPeakRate > 0 {
			fmt.Fprintf(w, `
                    <tr>
                        <td>🔋 Off-Peak Export Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
				agreement.Tariff.OffPeakRate,
			)
		}

			if !hasAnyExportRate {
				fmt.Fprintf(w, `
                    <tr>
                        <td>☀️ Export Rate</td>
                        <td><em>Time-varying (see earnings in analysis)</em></td>
                    </tr>
`)
			}

			validityPeriod := fmt.Sprintf("Valid From: %s", agreement.ValidFrom.Format("2006-01-02"))
			if agreement.ValidTo != nil {
				validityPeriod += fmt.Sprintf(" to %s", agreement.ValidTo.Format("2006-01-02"))
			}

			fmt.Fprintf(w, `
                </tbody>
            </table>
            <p style="margin-top: 10px; opacity: 0.7;"><em>%s</em></p>
`,
				html.EscapeString(validityPeriod),
			)
		}
	}

	// Gas Tariff
	if len(result.GasAgreements) > 0 {
		fmt.Fprintf(w, `
            <h3 style="margin-top: 30px;">🔥 Gas</h3>
`)

		// Show all agreements (current and upcoming)
		for i, agreement := range result.GasAgreements {
			if i > 0 {
				fmt.Fprintf(w, `
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--border-color);">
`)
			}

			fmt.Fprintf(w, `
            <p><strong>Tariff:</strong> %s</p>
`,
				html.EscapeString(agreement.Tariff.DisplayName),
			)

		if agreement.Tariff.FullName != "" && agreement.Tariff.FullName != agreement.Tariff.DisplayName {
			fmt.Fprintf(w, `
            <p><strong>Full Name:</strong> %s</p>
`,
				html.EscapeString(agreement.Tariff.FullName),
			)
		}

		fmt.Fprintf(w, `
            <table>
                <tbody>
                    <tr>
                        <td>💰 Standing Charge</td>
                        <td>%.2fp per day</td>
                    </tr>
`,
			agreement.Tariff.StandingCharge,
		)

			if agreement.Tariff.UnitRate > 0 {
				fmt.Fprintf(w, `
                    <tr>
                        <td>🔥 Unit Rate</td>
                        <td>%.2fp per kWh</td>
                    </tr>
`,
					agreement.Tariff.UnitRate,
				)
			}

			validityPeriod := fmt.Sprintf("Valid From: %s", agreement.ValidFrom.Format("2006-01-02"))
			if agreement.ValidTo != nil {
				validityPeriod += fmt.Sprintf(" to %s", agreement.ValidTo.Format("2006-01-02"))
			}

			fmt.Fprintf(w, `
                </tbody>
            </table>
            <p style="margin-top: 10px; opacity: 0.7;"><em>%s</em></p>
`,
				html.EscapeString(validityPeriod),
			)
		}
	}

	fmt.Fprintf(w, `
        </div>
`)
}

func (r *HTMLReporter) writeHTMLAnomalies(w io.Writer, result *AnalysisResult) {
	if len(result.Anomalies) == 0 {
		return
	}

	// Sort and take top 10
	anomalies := make([]Anomaly, len(result.Anomalies))
	copy(anomalies, result.Anomalies)
	sort.Slice(anomalies, func(i, j int) bool {
		return math.Abs(anomalies[i].DeviationPercent) > math.Abs(anomalies[j].DeviationPercent)
	})
	if len(anomalies) > 10 {
		anomalies = anomalies[:10]
	}

	fmt.Fprintf(w, `
        <div class="card">
            <h2>🔍 Anomalies Detected</h2>
            <p>Found <strong>%d anomalies</strong> in your consumption data. Showing top 10 most significant:</p>
            
            <table>
                <thead>
                    <tr>
                        <th>Date</th>
                        <th>Fuel</th>
                        <th>Type</th>
                        <th>Actual</th>
                        <th>Expected</th>
                        <th>Deviation</th>
                        <th>Weather</th>
                    </tr>
                </thead>
                <tbody>
`,
		len(result.Anomalies),
	)

	for _, anomaly := range anomalies {
		fuelIcon := "⚡"
		if anomaly.FuelType == "gas" {
			fuelIcon = "🔥"
		}

		typeIcon := "⚠️"
		typeText := "spike"
		if anomaly.Type == "low_usage" {
			typeIcon = "🔵"
			typeText = "low usage"
		}

		weatherDesc := "N/A"
		if anomaly.Weather != nil {
			weatherDesc = fmt.Sprintf("%s, %.1f°C, %.1fmm",
				anomaly.Weather.WeatherDesc,
				anomaly.Weather.TempMean,
				anomaly.Weather.Precipitation,
			)
		}

		fmt.Fprintf(w, `
                    <tr>
                        <td>%s</td>
                        <td>%s</td>
                        <td>%s %s</td>
                        <td>%.2f kWh</td>
                        <td>%.2f kWh</td>
                        <td>%.1f%%</td>
                        <td>%s</td>
                    </tr>
`,
			anomaly.Date.Format("2006-01-02"),
			fuelIcon,
			typeIcon,
			typeText,
			anomaly.ActualValue,
			anomaly.ExpectedValue,
			anomaly.DeviationPercent,
			html.EscapeString(weatherDesc),
		)
	}

	fmt.Fprintf(w, `
                </tbody>
            </table>
        </div>
`)
}

func (r *HTMLReporter) writeHTMLRecommendations(w io.Writer, result *AnalysisResult) {
	if len(result.Insights) == 0 {
		return
	}

	// Separate export and general insights
	exportInsights := []Insight{}
	generalInsights := []Insight{}

	for _, insight := range result.Insights {
		if insight.Category == "export" {
			exportInsights = append(exportInsights, insight)
		} else {
			generalInsights = append(generalInsights, insight)
		}
	}

	fmt.Fprintf(w, `
        <div class="card">
            <h2>💡 Recommendations</h2>
`)

	if len(exportInsights) > 0 {
		fmt.Fprintf(w, `
            <h3>☀️ Solar/Battery Export Insights</h3>
`)
		for _, insight := range exportInsights {
			priorityClass := "low"
			if insight.Priority == "high" {
				priorityClass = "high"
			} else if insight.Priority == "medium" {
				priorityClass = "medium"
			}

			fmt.Fprintf(w, `
            <div class="insight-box %s">
                <div class="insight-title">%s</div>
                <p>%s</p>
                <div class="insight-action">
                    <strong>Recommended Action:</strong> %s
                </div>
            </div>
`,
				priorityClass,
				html.EscapeString(insight.Title),
				html.EscapeString(insight.Description),
				html.EscapeString(insight.Action),
			)
		}
	}

	if len(generalInsights) > 0 {
		fmt.Fprintf(w, `
            <h3>🎯 General Insights</h3>
`)

		// Group by priority
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

		if len(highPriority) > 0 {
			fmt.Fprintf(w, `<h4>🔴 High Priority</h4>`)
			for _, insight := range highPriority {
				fmt.Fprintf(w, `
            <div class="insight-box high">
                <div class="insight-title">%s</div>
                <p>%s</p>
                <div class="insight-action">
                    <strong>Recommended Action:</strong> %s
                </div>
            </div>
`,
					html.EscapeString(insight.Title),
					html.EscapeString(insight.Description),
					html.EscapeString(insight.Action),
				)
			}
		}

		if len(mediumPriority) > 0 {
			fmt.Fprintf(w, `<h4>🟡 Medium Priority</h4>`)
			for _, insight := range mediumPriority {
				fmt.Fprintf(w, `
            <div class="insight-box medium">
                <div class="insight-title">%s</div>
                <p>%s</p>
                <div class="insight-action">
                    <strong>Recommended Action:</strong> %s
                </div>
            </div>
`,
					html.EscapeString(insight.Title),
					html.EscapeString(insight.Description),
					html.EscapeString(insight.Action),
				)
			}
		}

		if len(lowPriority) > 0 {
			fmt.Fprintf(w, `<h4>🔵 Low Priority</h4>`)
			for _, insight := range lowPriority {
				fmt.Fprintf(w, `
            <div class="insight-box">
                <div class="insight-title">%s</div>
                <p>%s</p>
                <div class="insight-action">
                    <strong>Recommended Action:</strong> %s
                </div>
            </div>
`,
					html.EscapeString(insight.Title),
					html.EscapeString(insight.Description),
					html.EscapeString(insight.Action),
				)
			}
		}
	}

	fmt.Fprintf(w, `
        </div>
`)
}

func (r *HTMLReporter) writeHTMLFooter(w io.Writer) {
	fmt.Fprintf(w, `
        <footer>
            <p><em>This report is based on historical data and projections may vary based on seasonal changes, tariff adjustments, and usage patterns. Please review your actual bills and account statements for precise information.</em></p>
            <p style="margin-top: 10px;">Generated by <a href="https://github.com/matthewgall/octobudget" style="color: var(--primary-color); text-decoration: none;">octobudget</a></p>
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--border-color); opacity: 0.3;">
            <p style="opacity: 0.7; font-size: 0.9em;">This is an unofficial third-party application. "Octopus Energy" is a trademark of Octopus Energy Group Limited. This application is not affiliated with, endorsed by, or connected to Octopus Energy.</p>
        </footer>
    </div>
</body>
</html>
`)
}
