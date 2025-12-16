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
	"io"
	"net/http"
	"time"
)

// FetchElectricityConsumption fetches electricity consumption data using REST API
func (c *OctopusClient) FetchElectricityConsumption(mpan, serialNumber string, startDate, endDate time.Time) ([]Consumption, []Agreement, error) {
	c.logger.Info("Fetching electricity consumption",
		"mpan", mpan,
		"serial", serialNumber,
		"start", startDate.Format("2006-01-02"),
		"end", endDate.Format("2006-01-02"),
	)

	// Build REST API URL with UTC timestamps
	url := fmt.Sprintf("%s/electricity-meter-points/%s/meters/%s/consumption/?page_size=25000&period_from=%sZ&period_to=%sZ&order_by=period",
		OctopusRESTAPIBase,
		mpan,
		serialNumber,
		startDate.Format("2006-01-02T15:04:05"),
		endDate.Format("2006-01-02T15:04:05"),
	)

	consumptions, err := c.fetchConsumptionREST(url, "electricity")
	if err != nil {
		return nil, nil, err
	}

	c.logger.LogDataCollection("electricity_consumption", len(consumptions))
	// Agreements come from account details, not consumption endpoint
	return consumptions, nil, nil
}

// FetchGasConsumption fetches gas consumption data using REST API
func (c *OctopusClient) FetchGasConsumption(mprn, serialNumber string, startDate, endDate time.Time) ([]Consumption, []Agreement, error) {
	c.logger.Info("Fetching gas consumption",
		"mprn", mprn,
		"serial", serialNumber,
		"start", startDate.Format("2006-01-02"),
		"end", endDate.Format("2006-01-02"),
	)

	// Build REST API URL with UTC timestamps
	url := fmt.Sprintf("%s/gas-meter-points/%s/meters/%s/consumption/?page_size=25000&period_from=%sZ&period_to=%sZ&order_by=period",
		OctopusRESTAPIBase,
		mprn,
		serialNumber,
		startDate.Format("2006-01-02T15:04:05"),
		endDate.Format("2006-01-02T15:04:05"),
	)

	consumptions, err := c.fetchConsumptionREST(url, "gas")
	if err != nil {
		return nil, nil, err
	}

	c.logger.LogDataCollection("gas_consumption", len(consumptions))
	// Agreements come from account details, not consumption endpoint
	return consumptions, nil, nil
}

// fetchConsumptionREST is a helper method to fetch consumption data from REST API
func (c *OctopusClient) fetchConsumptionREST(url, fuelType string) ([]Consumption, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use basic auth with API key
	req.SetBasicAuth(c.apiKey, "")
	req.Header.Set("User-Agent", GetUserAgent())

	c.logger.LogAPIRequest("GET", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &APIError{
			Endpoint: url,
			Message:  fmt.Sprintf("failed to fetch %s consumption", fuelType),
			Err:      err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.LogAPIError(url, resp.StatusCode, fmt.Errorf("%s", string(bodyBytes)))
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Endpoint:   url,
			Message:    string(bodyBytes),
		}
	}

	var restResp RESTConsumptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&restResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Consumption model
	consumptions := make([]Consumption, len(restResp.Results))
	for i, r := range restResp.Results {
		startAt, _ := time.Parse(time.RFC3339, r.IntervalStart)
		endAt, _ := time.Parse(time.RFC3339, r.IntervalEnd)

		consumptions[i] = Consumption{
			StartAt: startAt,
			EndAt:   endAt,
			Value:   r.Consumption,
			Cost:    0, // Cost not provided by REST API, will be calculated from tariff
		}
	}

	return consumptions, nil
}

// FetchProductCode fetches the product code from the Products API based on tariff display name
func (c *OctopusClient) FetchProductCode(tariffDisplayName string) (string, error) {
	url := fmt.Sprintf("%s/products/", OctopusRESTAPIBase)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &APIError{
			Endpoint: url,
			Message:  "failed to fetch products",
			Err:      err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", &APIError{
			StatusCode: resp.StatusCode,
			Endpoint:   url,
			Message:    string(bodyBytes),
		}
	}

	var productsResp ProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&productsResp); err != nil {
		return "", fmt.Errorf("failed to decode products response: %w", err)
	}

	// Find matching product by display name
	for _, product := range productsResp.Results {
		if product.DisplayName == tariffDisplayName {
			c.logger.Info("Found product code", "tariff", tariffDisplayName, "code", product.Code)
			return product.Code, nil
		}
	}

	return "", fmt.Errorf("product code not found for tariff: %s", tariffDisplayName)
}

// FetchElectricityTariffRates fetches time-varying unit rates for an electricity tariff
func (c *OctopusClient) FetchElectricityTariffRates(productCode string, startDate, endDate time.Time) ([]TariffRate, error) {
	// Construct tariff code from product code (format: E-1R-{PRODUCT_CODE}-C for standard region)
	tariffCode := fmt.Sprintf("E-1R-%s-C", productCode)

	url := fmt.Sprintf("%s/products/%s/electricity-tariffs/%s/standard-unit-rates/?period_from=%sZ&period_to=%sZ",
		OctopusRESTAPIBase,
		productCode,
		tariffCode,
		startDate.Format("2006-01-02T15:04:05"),
		endDate.Format("2006-01-02T15:04:05"),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", GetUserAgent())

	c.logger.LogAPIRequest("GET", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &APIError{
			Endpoint: url,
			Message:  "failed to fetch tariff rates",
			Err:      err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.LogAPIError(url, resp.StatusCode, fmt.Errorf("%s", string(bodyBytes)))
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Endpoint:   url,
			Message:    string(bodyBytes),
		}
	}

	var ratesResp TariffRatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&ratesResp); err != nil {
		return nil, fmt.Errorf("failed to decode rates response: %w", err)
	}

	// Convert to TariffRate model
	rates := make([]TariffRate, len(ratesResp.Results))
	for i, r := range ratesResp.Results {
		validFrom, _ := time.Parse(time.RFC3339, r.ValidFrom)
		var validTo *time.Time
		if r.ValidTo != nil {
			t, _ := time.Parse(time.RFC3339, *r.ValidTo)
			validTo = &t
		}

		rates[i] = TariffRate{
			ValidFrom:   validFrom,
			ValidTo:     validTo,
			ValueExcVAT: r.ValueExcVAT,
			ValueIncVAT: r.ValueIncVAT,
		}
	}

	c.logger.Info("Fetched electricity tariff rates", "count", len(rates))
	return rates, nil
}
