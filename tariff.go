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
	"time"
)

// CalculateConsumptionCosts calculates costs for consumption data using tariff agreements
// This function uses simple tariff rates (unit rate, day/night rates) from agreements
func CalculateConsumptionCosts(consumptions []Consumption, agreements []Agreement) []Consumption {
	if len(consumptions) == 0 || len(agreements) == 0 {
		return consumptions
	}

	// Process each consumption record
	for i := range consumptions {
		// Find the tariff that was active during this consumption period
		tariff := findActiveTariff(consumptions[i].StartAt, agreements)
		if tariff == nil {
			continue
		}

		// Determine the rate to use based on time of day
		rate := getRateForTime(consumptions[i].StartAt, tariff)

		// Calculate cost: consumption (kWh) × rate (p/kWh) = cost in pence
		consumptions[i].Cost = consumptions[i].Value * rate
	}

	return consumptions
}

// CalculateConsumptionCostsWithRates calculates costs using time-varying rates from Products API
// This function provides more accurate costs for complex tariffs like Intelligent Octopus Flux
func CalculateConsumptionCostsWithRates(consumptions []Consumption, rates []TariffRate) []Consumption {
	if len(consumptions) == 0 || len(rates) == 0 {
		return consumptions
	}

	// Process each consumption record
	for i := range consumptions {
		// Find the rate that was active during this consumption period
		rate := findActiveRate(consumptions[i].StartAt, rates)
		if rate == nil {
			continue
		}

		// Calculate cost: consumption (kWh) × rate (p/kWh) = cost in pence
		consumptions[i].Cost = consumptions[i].Value * rate.ValueIncVAT
	}

	return consumptions
}

// findActiveRate finds the tariff rate that was active at the given time
func findActiveRate(t time.Time, rates []TariffRate) *TariffRate {
	for i := range rates {
		// Check if time is within the rate period
		if t.Before(rates[i].ValidFrom) {
			continue
		}
		if rates[i].ValidTo != nil && t.After(*rates[i].ValidTo) {
			continue
		}
		return &rates[i]
	}
	return nil
}

// findActiveTariff finds the tariff agreement that was active at the given time
func findActiveTariff(t time.Time, agreements []Agreement) *Tariff {
	for _, agreement := range agreements {
		// Check if time is within the agreement period
		if t.Before(agreement.ValidFrom) {
			continue
		}
		if agreement.ValidTo != nil && t.After(*agreement.ValidTo) {
			continue
		}
		return &agreement.Tariff
	}
	return nil
}

// getRateForTime determines the appropriate rate based on time of day
func getRateForTime(t time.Time, tariff *Tariff) float64 {
	// If it's a simple tariff with only unitRate, use that
	if tariff.UnitRate > 0 && tariff.DayRate == 0 && tariff.NightRate == 0 {
		return tariff.UnitRate
	}

	// If it's a day/night tariff, determine which rate to use
	if tariff.DayRate > 0 || tariff.NightRate > 0 {
		if isNightRate(t) {
			return tariff.NightRate
		}
		return tariff.DayRate
	}

	// Fallback to unit rate
	return tariff.UnitRate
}

// isNightRate determines if the given time falls within night rate hours
// For Intelligent Octopus Flux, typical night rates are:
// - 02:00-05:00 (off-peak/cheap rate)
// This can be made configurable in the future
func isNightRate(t time.Time) bool {
	hour := t.Hour()

	// Intelligent Octopus Flux off-peak: 02:00-05:00
	// Note: This is a common pattern but may vary by tariff
	if hour >= 2 && hour < 5 {
		return true
	}

	return false
}
