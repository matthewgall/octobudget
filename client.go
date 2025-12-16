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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// OctopusClient handles communication with the Octopus Energy GraphQL API
type OctopusClient struct {
	accountID  string
	apiKey     string
	httpClient *http.Client
	logger     *Logger

	// JWT token management
	jwtToken  string
	jwtExpiry time.Time
	jwtMutex  sync.RWMutex

	// Rate limiting
	lastRequest  time.Time
	requestMutex sync.Mutex
}

// NewOctopusClient creates a new Octopus Energy API client
func NewOctopusClient(accountID, apiKey string, logger *Logger) *OctopusClient {
	return &OctopusClient{
		accountID: accountID,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ensureValidToken ensures we have a valid JWT token
func (c *OctopusClient) ensureValidToken() error {
	c.jwtMutex.RLock()
	hasValidToken := c.jwtToken != "" && time.Now().Before(c.jwtExpiry)
	c.jwtMutex.RUnlock()

	if hasValidToken {
		return nil
	}

	return c.refreshJWTToken()
}

// refreshJWTToken obtains a new JWT token from the API
func (c *OctopusClient) refreshJWTToken() error {
	c.jwtMutex.Lock()
	defer c.jwtMutex.Unlock()

	c.logger.Debug("Refreshing JWT token")

	variables := map[string]interface{}{
		"apiKey": c.apiKey,
	}

	payload := map[string]interface{}{
		"query":     obtainJWTTokenQuery,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal token request: %w", err)
	}

	req, err := http.NewRequest("POST", OctopusGraphQLEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &APIError{
			Endpoint: OctopusGraphQLEndpoint,
			Message:  "failed to request JWT token",
			Err:      err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Endpoint:   OctopusGraphQLEndpoint,
			Message:    fmt.Sprintf("token request failed: %s", string(bodyBytes)),
		}
	}

	var tokenResp ObtainTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	if len(tokenResp.Errors) > 0 {
		return &AuthError{
			Message: fmt.Sprintf("GraphQL error obtaining token: %s", tokenResp.Errors[0].Message),
		}
	}

	if tokenResp.Data.ObtainKrakenToken.Token == "" {
		return &AuthError{
			Message: "empty token received from API",
		}
	}

	c.jwtToken = tokenResp.Data.ObtainKrakenToken.Token
	// Set expiry to 23 hours from now (tokens typically last 24 hours)
	c.jwtExpiry = time.Now().Add(23 * time.Hour)

	c.logger.Debug("JWT token refreshed successfully")
	return nil
}

// makeGraphQLRequest makes a GraphQL request with proper authentication
func (c *OctopusClient) makeGraphQLRequest(query string, variables map[string]interface{}, result interface{}) error {
	// Ensure we have a valid token
	if err := c.ensureValidToken(); err != nil {
		return err
	}

	// Rate limiting: minimum 100ms between requests
	c.requestMutex.Lock()
	if elapsed := time.Since(c.lastRequest); elapsed < 100*time.Millisecond {
		time.Sleep(100*time.Millisecond - elapsed)
	}
	c.lastRequest = time.Now()
	c.requestMutex.Unlock()

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequest("POST", OctopusGraphQLEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create GraphQL request: %w", err)
	}

	c.jwtMutex.RLock()
	token := c.jwtToken
	c.jwtMutex.RUnlock()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", GetUserAgent())

	c.logger.LogAPIRequest("POST", OctopusGraphQLEndpoint)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &APIError{
			Endpoint: OctopusGraphQLEndpoint,
			Message:  "GraphQL request failed",
			Err:      err,
		}
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for auth errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// Invalidate token and retry once
		c.jwtMutex.Lock()
		c.jwtToken = ""
		c.jwtMutex.Unlock()

		return &AuthError{
			Message: fmt.Sprintf("authentication failed (status %d)", resp.StatusCode),
		}
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.LogAPIError(OctopusGraphQLEndpoint, resp.StatusCode, fmt.Errorf("%s", string(bodyBytes)))
		return &APIError{
			StatusCode: resp.StatusCode,
			Endpoint:   OctopusGraphQLEndpoint,
			Message:    string(bodyBytes),
		}
	}

	// Decode response
	if err := json.Unmarshal(bodyBytes, result); err != nil {
		return fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	return nil
}

// FetchAccountDetails fetches account details including meter points and tariffs
func (c *OctopusClient) FetchAccountDetails() (*Account, error) {
	c.logger.Info("Fetching account details", "account", c.accountID)

	variables := map[string]interface{}{
		"accountNumber": c.accountID,
	}

	var response AccountDetailsResponse
	if err := c.makeGraphQLRequest(accountDetailsQuery, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to fetch account details: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", response.Errors[0].Message)
	}

	// Convert response to Account model
	account := &Account{
		Number:     response.Data.Account.Number,
		Balance:    float64(response.Data.Account.Balance) / 100.0, // Convert pence to pounds
		Properties: make([]Property, len(response.Data.Account.Properties)),
	}

	for i, prop := range response.Data.Account.Properties {
		property := Property{
			ID:                     prop.ID,
			Address:                prop.Address,
			ElectricityMeterPoints: make([]ElectricityMeterPoint, len(prop.ElectricityMeterPoints)),
			GasMeterPoints:         make([]GasMeterPoint, len(prop.GasMeterPoints)),
		}

		// Convert electricity meter points
		for j, emp := range prop.ElectricityMeterPoints {
			meterPoint := ElectricityMeterPoint{
				MPAN:       emp.MPAN,
				Meters:     make([]Meter, len(emp.Meters)),
				Agreements: make([]Agreement, len(emp.Agreements)),
			}

			for k, m := range emp.Meters {
				smartDevices := make([]SmartDevice, len(m.SmartDevices))
				for i, sd := range m.SmartDevices {
					smartDevices[i] = SmartDevice{DeviceID: sd.DeviceID}
				}
				meterPoint.Meters[k] = Meter{
					SerialNumber: m.SerialNumber,
					SmartDevices: smartDevices,
				}
			}

			for k, a := range emp.Agreements {
				validFrom, _ := time.Parse(time.RFC3339, a.ValidFrom)
				var validTo *time.Time
				if a.ValidTo != nil {
					t, _ := time.Parse(time.RFC3339, *a.ValidTo)
					validTo = &t
				}

				meterPoint.Agreements[k] = Agreement{
					ValidFrom: validFrom,
					ValidTo:   validTo,
					Tariff: Tariff{
						DisplayName:    a.Tariff.DisplayName,
						FullName:       a.Tariff.FullName,
						StandingCharge: a.Tariff.StandingCharge,
						UnitRate:       a.Tariff.UnitRate,
						DayRate:        a.Tariff.DayRate,
						NightRate:      a.Tariff.NightRate,
					},
				}
			}

			property.ElectricityMeterPoints[j] = meterPoint
		}

		// Convert gas meter points
		for j, gmp := range prop.GasMeterPoints {
			meterPoint := GasMeterPoint{
				MPRN:       gmp.MPRN,
				Meters:     make([]Meter, len(gmp.Meters)),
				Agreements: make([]Agreement, len(gmp.Agreements)),
			}

			for k, m := range gmp.Meters {
				smartDevices := make([]SmartDevice, len(m.SmartDevices))
				for i, sd := range m.SmartDevices {
					smartDevices[i] = SmartDevice{DeviceID: sd.DeviceID}
				}
				meterPoint.Meters[k] = Meter{
					SerialNumber: m.SerialNumber,
					SmartDevices: smartDevices,
				}
			}

			for k, a := range gmp.Agreements {
				validFrom, _ := time.Parse(time.RFC3339, a.ValidFrom)
				var validTo *time.Time
				if a.ValidTo != nil {
					t, _ := time.Parse(time.RFC3339, *a.ValidTo)
					validTo = &t
				}

				meterPoint.Agreements[k] = Agreement{
					ValidFrom: validFrom,
					ValidTo:   validTo,
					Tariff: Tariff{
						DisplayName:    a.Tariff.DisplayName,
						FullName:       a.Tariff.FullName,
						StandingCharge: a.Tariff.StandingCharge,
						UnitRate:       a.Tariff.UnitRate,
						DayRate:        a.Tariff.DayRate,
						NightRate:      a.Tariff.NightRate,
					},
				}
			}

			property.GasMeterPoints[j] = meterPoint
		}

		account.Properties[i] = property
	}

	c.logger.Info("Account details fetched successfully")
	return account, nil
}
