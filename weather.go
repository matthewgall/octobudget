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
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WeatherClient fetches historical weather data
type WeatherClient struct {
	httpClient *http.Client
	logger     *Logger
	// UK approximate center coordinates (used if no location specified)
	latitude  float64
	longitude float64
}

// NewWeatherClient creates a new weather client
// Default coordinates are for central UK (around Birmingham)
func NewWeatherClient(logger *Logger) *WeatherClient {
	return &WeatherClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		latitude:   52.4862,  // Birmingham, UK
		longitude:  -1.8904,
	}
}

// FetchWeatherForDates fetches historical weather data for specific dates
func (w *WeatherClient) FetchWeatherForDates(dates []time.Time) (map[string]*WeatherData, error) {
	if len(dates) == 0 {
		return nil, nil
	}

	// Find date range
	var startDate, endDate time.Time
	for i, date := range dates {
		if i == 0 {
			startDate = date
			endDate = date
		} else {
			if date.Before(startDate) {
				startDate = date
			}
			if date.After(endDate) {
				endDate = date
			}
		}
	}

	// Fetch weather data for the date range
	url := fmt.Sprintf("https://archive-api.open-meteo.com/v1/archive?latitude=%.4f&longitude=%.4f&start_date=%s&end_date=%s&daily=temperature_2m_max,temperature_2m_min,temperature_2m_mean,precipitation_sum,weather_code&timezone=Europe%%2FLondon",
		w.latitude,
		w.longitude,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	)

	w.logger.Info("Fetching weather data", "start", startDate.Format("2006-01-02"), "end", endDate.Format("2006-01-02"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create weather request: %w", err)
	}

	req.Header.Set("User-Agent", GetUserAgent())

	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.logger.Warn("Failed to fetch weather data", "error", err)
		return nil, nil // Non-fatal, return nil to continue without weather
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logger.Warn("Weather API returned non-200 status", "status", resp.StatusCode)
		return nil, nil // Non-fatal
	}

	var weatherResp OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		w.logger.Warn("Failed to decode weather response", "error", err)
		return nil, nil // Non-fatal
	}

	// Convert to map for easy lookup
	weatherMap := make(map[string]*WeatherData)
	for i, dateStr := range weatherResp.Daily.Time {
		date, _ := time.Parse("2006-01-02", dateStr)
		weatherMap[dateStr] = &WeatherData{
			Date:          date,
			TempMax:       weatherResp.Daily.TempMax[i],
			TempMin:       weatherResp.Daily.TempMin[i],
			TempMean:      weatherResp.Daily.TempMean[i],
			Precipitation: weatherResp.Daily.Precipitation[i],
			WeatherCode:   weatherResp.Daily.WeatherCode[i],
			WeatherDesc:   getWeatherDescription(weatherResp.Daily.WeatherCode[i]),
		}
	}

	w.logger.Info("Fetched weather data", "days", len(weatherMap))
	return weatherMap, nil
}

// getWeatherDescription converts WMO weather code to human-readable description
func getWeatherDescription(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1, 2, 3:
		return "Partly cloudy"
	case 45, 48:
		return "Foggy"
	case 51, 53, 55:
		return "Drizzle"
	case 61, 63, 65:
		return "Rain"
	case 71, 73, 75:
		return "Snow"
	case 77:
		return "Snow grains"
	case 80, 81, 82:
		return "Rain showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}
