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
	"encoding/base64"
	"fmt"
	"time"

	charts "github.com/vicanso/go-charts/v2"
)

// ChartGenerator handles chart generation
type ChartGenerator struct {
	theme string
}

// NewChartGenerator creates a new chart generator
func NewChartGenerator() *ChartGenerator {
	return &ChartGenerator{
		theme: "dark", // Match our HTML report dark theme
	}
}

// GenerateDailyUsageChart creates a line chart showing daily electricity and gas usage
func (cg *ChartGenerator) GenerateDailyUsageChart(data *CollectedData) (string, error) {
	if len(data.ElectricityConsumption) == 0 && len(data.GasConsumption) == 0 {
		return "", fmt.Errorf("no consumption data available")
	}

	// Aggregate consumption by day
	dailyElectricity := aggregateByDay(data.ElectricityConsumption)
	dailyGas := aggregateByDay(data.GasConsumption)

	// Get all unique dates and sort them
	dates := getUniqueSortedDates(dailyElectricity, dailyGas)
	if len(dates) == 0 {
		return "", fmt.Errorf("no dates found in consumption data")
	}

	// Build series data
	var electricityValues []float64
	var gasValues []float64
	var labels []string

	for _, date := range dates {
		labels = append(labels, date.Format("Jan 2"))
		electricityValues = append(electricityValues, dailyElectricity[date])
		gasValues = append(gasValues, dailyGas[date])
	}

	// Prepare chart values
	values := [][]float64{}
	legendLabels := []string{}

	if len(data.ElectricityConsumption) > 0 {
		values = append(values, electricityValues)
		legendLabels = append(legendLabels, "Electricity (kWh)")
	}
	if len(data.GasConsumption) > 0 {
		values = append(values, gasValues)
		legendLabels = append(legendLabels, "Gas (kWh)")
	}

	// Create the chart
	p, err := charts.LineRender(
		values,
		charts.TitleTextOptionFunc("Daily Energy Usage"),
		charts.XAxisDataOptionFunc(labels),
		charts.LegendLabelsOptionFunc(legendLabels, charts.PositionRight),
		charts.ThemeOptionFunc(cg.getTheme()),
		charts.WidthOptionFunc(1200),
		charts.HeightOptionFunc(400),
		charts.PaddingOptionFunc(charts.Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		}),
	)
	if err != nil {
		return "", fmt.Errorf("failed to render usage chart: %w", err)
	}

	// Convert to base64 for embedding in HTML
	buf, err := p.Bytes()
	if err != nil {
		return "", fmt.Errorf("failed to generate chart bytes: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

// GenerateDailyCostChart creates a line chart showing daily costs
func (cg *ChartGenerator) GenerateDailyCostChart(data *CollectedData) (string, error) {
	if len(data.ElectricityConsumption) == 0 && len(data.GasConsumption) == 0 {
		return "", fmt.Errorf("no consumption data available")
	}

	// Aggregate costs by day (convert pence to pounds)
	dailyElectricityCost := aggregateCostByDay(data.ElectricityConsumption)
	dailyGasCost := aggregateCostByDay(data.GasConsumption)
	dailyElectricityExportEarnings := aggregateCostByDay(data.ElectricityExport)

	// Get all unique dates and sort them
	dates := getUniqueSortedDates(dailyElectricityCost, dailyGasCost, dailyElectricityExportEarnings)
	if len(dates) == 0 {
		return "", fmt.Errorf("no dates found in consumption data")
	}

	// Build series data
	var electricityValues []float64
	var gasValues []float64
	var exportValues []float64
	var netValues []float64
	var labels []string

	for _, date := range dates {
		labels = append(labels, date.Format("Jan 2"))
		elecCost := dailyElectricityCost[date] / 100.0 // pence to pounds
		gasCost := dailyGasCost[date] / 100.0
		exportEarnings := dailyElectricityExportEarnings[date] / 100.0
		netCost := elecCost + gasCost - exportEarnings

		electricityValues = append(electricityValues, elecCost)
		gasValues = append(gasValues, gasCost)
		exportValues = append(exportValues, exportEarnings)
		netValues = append(netValues, netCost)
	}

	// Prepare chart values
	values := [][]float64{}
	legendLabels := []string{}

	// Always show net cost first
	values = append(values, netValues)
	legendLabels = append(legendLabels, "Net Daily Cost (£)")

	if len(data.ElectricityConsumption) > 0 {
		values = append(values, electricityValues)
		legendLabels = append(legendLabels, "Electricity Cost (£)")
	}
	if len(data.GasConsumption) > 0 {
		values = append(values, gasValues)
		legendLabels = append(legendLabels, "Gas Cost (£)")
	}
	if len(data.ElectricityExport) > 0 {
		values = append(values, exportValues)
		legendLabels = append(legendLabels, "Export Earnings (£)")
	}

	// Create the chart
	p, err := charts.LineRender(
		values,
		charts.TitleTextOptionFunc("Daily Energy Costs"),
		charts.XAxisDataOptionFunc(labels),
		charts.LegendLabelsOptionFunc(legendLabels, charts.PositionRight),
		charts.ThemeOptionFunc(cg.getTheme()),
		charts.WidthOptionFunc(1200),
		charts.HeightOptionFunc(400),
		charts.PaddingOptionFunc(charts.Box{
			Top:    20,
			Right:  20,
			Bottom: 20,
			Left:   20,
		}),
	)
	if err != nil {
		return "", fmt.Errorf("failed to render cost chart: %w", err)
	}

	// Convert to base64 for embedding in HTML
	buf, err := p.Bytes()
	if err != nil {
		return "", fmt.Errorf("failed to generate chart bytes: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

// aggregateByDay groups consumption values by date and sums them
func aggregateByDay(consumption []Consumption) map[time.Time]float64 {
	daily := make(map[time.Time]float64)
	for _, c := range consumption {
		date := time.Date(c.StartAt.Year(), c.StartAt.Month(), c.StartAt.Day(), 0, 0, 0, 0, c.StartAt.Location())
		daily[date] += c.Value
	}
	return daily
}

// aggregateCostByDay groups consumption costs by date and sums them
func aggregateCostByDay(consumption []Consumption) map[time.Time]float64 {
	daily := make(map[time.Time]float64)
	for _, c := range consumption {
		date := time.Date(c.StartAt.Year(), c.StartAt.Month(), c.StartAt.Day(), 0, 0, 0, 0, c.StartAt.Location())
		daily[date] += c.Cost
	}
	return daily
}

// getUniqueSortedDates extracts and sorts all unique dates from multiple maps
func getUniqueSortedDates(maps ...map[time.Time]float64) []time.Time {
	dateSet := make(map[time.Time]bool)
	for _, m := range maps {
		for date := range m {
			dateSet[date] = true
		}
	}

	dates := make([]time.Time, 0, len(dateSet))
	for date := range dateSet {
		dates = append(dates, date)
	}

	// Sort dates
	for i := 0; i < len(dates)-1; i++ {
		for j := i + 1; j < len(dates); j++ {
			if dates[i].After(dates[j]) {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}

	return dates
}

// getTheme returns the chart theme name
func (cg *ChartGenerator) getTheme() string {
	return cg.theme
}
