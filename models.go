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

// Account represents an Octopus Energy account
type Account struct {
	Number     string     `json:"number"`
	Balance    float64    `json:"balance"` // In pence
	Properties []Property `json:"properties"`
}

// Property represents a property on an account
type Property struct {
	ID                     string                  `json:"id"`
	Address                string                  `json:"address"`
	ElectricityMeterPoints []ElectricityMeterPoint `json:"electricityMeterPoints"`
	GasMeterPoints         []GasMeterPoint         `json:"gasMeterPoints"`
}

// ElectricityMeterPoint represents an electricity meter point
type ElectricityMeterPoint struct {
	MPAN       string      `json:"mpan"`
	Meters     []Meter     `json:"meters"`
	Agreements []Agreement `json:"agreements"`
}

// GasMeterPoint represents a gas meter point
type GasMeterPoint struct {
	MPRN       string      `json:"mprn"`
	Meters     []Meter     `json:"meters"`
	Agreements []Agreement `json:"agreements"`
}

// Meter represents a physical meter
type Meter struct {
	SerialNumber string        `json:"serialNumber"`
	SmartDevices []SmartDevice `json:"smartDevices"`
}

// SmartDevice represents a smart meter device
type SmartDevice struct {
	DeviceID string `json:"deviceId"`
}

// Agreement represents a tariff agreement
type Agreement struct {
	ValidFrom time.Time  `json:"validFrom"`
	ValidTo   *time.Time `json:"validTo,omitempty"`
	Tariff    Tariff     `json:"tariff"`
}

// Tariff represents pricing information
type Tariff struct {
	DisplayName    string  `json:"displayName"`
	FullName       string  `json:"fullName"`
	StandingCharge float64 `json:"standingCharge"` // Pence per day
	UnitRate       float64 `json:"unitRate"`       // Pence per kWh (for simple tariffs)
	DayRate        float64 `json:"dayRate"`        // Pence per kWh (for day/night tariffs)
	NightRate      float64 `json:"nightRate"`      // Pence per kWh (for day/night tariffs)
	OffPeakRate    float64 `json:"offPeakRate"`    // Pence per kWh (for Flux tariffs)
}

// Consumption represents energy consumption data
type Consumption struct {
	StartAt time.Time `json:"startAt"`
	EndAt   time.Time `json:"endAt"`
	Value   float64   `json:"value"` // kWh
	Cost    float64   `json:"cost"`  // Pence
}

// Statement represents a billing statement
type Statement struct {
	ID                 string    `json:"id"`
	IssuedDate         time.Time `json:"issuedDate"`
	FromDate           time.Time `json:"fromDate"`
	ToDate             time.Time `json:"toDate"`
	TotalAmount        float64   `json:"totalAmount"`        // Pence
	ChargeAmount       float64   `json:"chargeAmount"`       // Pence
	PaymentAmount      float64   `json:"paymentAmount"`      // Pence
	OutstandingBalance float64   `json:"outstandingBalance"` // Pence
}

// Payment represents a payment transaction
type Payment struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"` // Pence
	PaymentDate time.Time `json:"paymentDate"`
	Method      string    `json:"method"`
}

// CollectedData holds all data fetched from the API
type CollectedData struct {
	Account                *Account      `json:"account"`
	ElectricityConsumption []Consumption `json:"electricityConsumption"` // Import/consumption
	ElectricityExport      []Consumption `json:"electricityExport"`      // Solar/battery export
	GasConsumption         []Consumption `json:"gasConsumption"`
	ElectricityAgreements  []Agreement   `json:"electricityAgreements"`
	ElectricityExportAgreements []Agreement `json:"electricityExportAgreements"`
	GasAgreements          []Agreement   `json:"gasAgreements"`
	Statements             []Statement   `json:"statements"`
	Payments               []Payment     `json:"payments"`
	FetchedAt              time.Time     `json:"fetchedAt"`
}

// AnalysisResult holds the complete analysis output
type AnalysisResult struct {
	GeneratedAt                 time.Time      `json:"generatedAt"`
	AnalysisPeriodStart         time.Time      `json:"analysisPeriodStart"`
	AnalysisPeriodEnd           time.Time      `json:"analysisPeriodEnd"`
	AnalysisPeriodDays          int            `json:"analysisPeriodDays"`
	CurrentBalance              float64        `json:"currentBalance"`          // Pounds
	AvgDailyElectricity         float64        `json:"avgDailyElectricity"`     // kWh (import)
	AvgDailyExport              float64        `json:"avgDailyExport"`          // kWh (solar/battery export)
	AvgDailyGas                 float64        `json:"avgDailyGas"`             // kWh
	AvgDailyCostElectricity     float64        `json:"avgDailyCostElectricity"` // Pounds (import cost)
	AvgDailyEarningsExport      float64        `json:"avgDailyEarningsExport"`  // Pounds (export earnings)
	AvgDailyCostGas             float64        `json:"avgDailyCostGas"`         // Pounds
	AvgDailyCostTotal           float64        `json:"avgDailyCostTotal"`       // Pounds (net: import - export + gas)
	ProjectedMonthlyCost        float64        `json:"projectedMonthlyCost"`    // Pounds
	RecommendedDirectDebit      float64        `json:"recommendedDirectDebit"`  // Pounds
	CurrentDirectDebit          float64        `json:"currentDirectDebit"`      // Pounds
	PaymentStatus               string         `json:"paymentStatus"`           // Balanced/Underpaying/Overpaying
	ElectricityAgreements       []Agreement    `json:"electricityAgreements"`
	ElectricityExportAgreements []Agreement    `json:"electricityExportAgreements"`
	GasAgreements               []Agreement    `json:"gasAgreements"`
	Anomalies                   []Anomaly      `json:"anomalies"`
	TariffChanges               []TariffChange `json:"tariffChanges"`
	Insights                    []Insight      `json:"insights"`
	// Charts (base64 encoded PNG images)
	DailyUsageChart string `json:"dailyUsageChart,omitempty"`
	DailyCostChart  string `json:"dailyCostChart,omitempty"`
}

// Anomaly represents a detected anomaly in consumption or cost
type Anomaly struct {
	Date             time.Time    `json:"date"`
	FuelType         string       `json:"fuelType"` // electricity, gas, export
	Type             string       `json:"type"`     // consumption_spike, cost_spike, low_usage
	Description      string       `json:"description"`
	ActualValue      float64      `json:"actualValue"`
	ExpectedValue    float64      `json:"expectedValue"`
	DeviationPercent float64      `json:"deviationPercent"`
	Weather          *WeatherData `json:"weather,omitempty"` // Optional weather context
}

// WeatherData represents weather information for a specific date
type WeatherData struct {
	Date          time.Time `json:"date"`
	TempMax       float64   `json:"temp_max"`       // Celsius
	TempMin       float64   `json:"temp_min"`       // Celsius
	TempMean      float64   `json:"temp_mean"`      // Celsius
	Precipitation float64   `json:"precipitation"`  // mm
	WeatherCode   int       `json:"weather_code"`   // WMO weather code
	WeatherDesc   string    `json:"weather_desc"`   // Human-readable description
}

// TariffChange represents a detected tariff change
type TariffChange struct {
	ChangeDate        time.Time `json:"changeDate"`
	FuelType          string    `json:"fuelType"` // electricity or gas
	OldTariffName     string    `json:"oldTariffName"`
	NewTariffName     string    `json:"newTariffName"`
	UnitRateChange    float64   `json:"unitRateChange"` // Pence per kWh (positive = increase)
	ImpactDescription string    `json:"impactDescription"`
}

// Insight represents an actionable recommendation
type Insight struct {
	Category    string `json:"category"` // payment, usage, tariff, seasonal
	Priority    string `json:"priority"` // high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// GraphQL response structures for API calls

// ObtainTokenResponse represents the JWT token response
type ObtainTokenResponse struct {
	Data struct {
		ObtainKrakenToken struct {
			Token string `json:"token"`
		} `json:"obtainKrakenToken"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// AccountDetailsResponse represents the account details GraphQL response
type AccountDetailsResponse struct {
	Data struct {
		Account struct {
			Number     string `json:"number"`
			Balance    int    `json:"balance"` // Pence
			Properties []struct {
				ID                     string `json:"id"`
				Address                string `json:"address"`
				ElectricityMeterPoints []struct {
					MPAN   string `json:"mpan"`
					Meters []struct {
						SerialNumber string `json:"serialNumber"`
						SmartDevices []struct {
							DeviceID string `json:"deviceId"`
						} `json:"smartDevices"`
					} `json:"meters"`
					Agreements []struct {
						ValidFrom string  `json:"validFrom"`
						ValidTo   *string `json:"validTo"`
						Tariff    struct {
							DisplayName    string  `json:"displayName"`
							FullName       string  `json:"fullName"`
							StandingCharge float64 `json:"standingCharge"`
							UnitRate       float64 `json:"unitRate"`
							DayRate        float64 `json:"dayRate"`
							NightRate      float64 `json:"nightRate"`
						} `json:"tariff"`
					} `json:"agreements"`
				} `json:"electricityMeterPoints"`
				GasMeterPoints []struct {
					MPRN   string `json:"mprn"`
					Meters []struct {
						SerialNumber string `json:"serialNumber"`
						SmartDevices []struct {
							DeviceID string `json:"deviceId"`
						} `json:"smartDevices"`
					} `json:"meters"`
					Agreements []struct {
						ValidFrom string  `json:"validFrom"`
						ValidTo   *string `json:"validTo"`
						Tariff    struct {
							DisplayName    string  `json:"displayName"`
							FullName       string  `json:"fullName"`
							StandingCharge float64 `json:"standingCharge"`
							UnitRate       float64 `json:"unitRate"`
							DayRate        float64 `json:"dayRate"`
							NightRate      float64 `json:"nightRate"`
						} `json:"tariff"`
					} `json:"agreements"`
				} `json:"gasMeterPoints"`
			} `json:"properties"`
		} `json:"account"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// ConsumptionResponse represents consumption data GraphQL response
type ConsumptionResponse struct {
	Data struct {
		Account *struct {
			ElectricityAgreements []struct {
				ValidFrom string  `json:"validFrom"`
				ValidTo   *string `json:"validTo"`
				Tariff    struct {
					DisplayName    string  `json:"displayName"`
					StandingCharge float64 `json:"standingCharge"`
					UnitRate       float64 `json:"unitRate"`
				} `json:"tariff"`
			} `json:"electricityAgreements,omitempty"`
			GasAgreements []struct {
				ValidFrom string  `json:"validFrom"`
				ValidTo   *string `json:"validTo"`
				Tariff    struct {
					DisplayName    string  `json:"displayName"`
					StandingCharge float64 `json:"standingCharge"`
					UnitRate       float64 `json:"unitRate"`
				} `json:"tariff"`
			} `json:"gasAgreements,omitempty"`
		} `json:"account"`
		Electricity []struct {
			StartAt string  `json:"startAt"`
			EndAt   string  `json:"endAt"`
			Value   float64 `json:"value"`
			Cost    float64 `json:"cost"`
		} `json:"electricity,omitempty"`
		Gas []struct {
			StartAt string  `json:"startAt"`
			EndAt   string  `json:"endAt"`
			Value   float64 `json:"value"`
			Cost    float64 `json:"cost"`
		} `json:"gas,omitempty"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string   `json:"message"`
	Path    []string `json:"path,omitempty"`
}

// RESTConsumptionResponse represents the REST API consumption response
type RESTConsumptionResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		IntervalStart string  `json:"interval_start"`
		IntervalEnd   string  `json:"interval_end"`
		Consumption   float64 `json:"consumption"`
	} `json:"results"`
}

// ProductsResponse represents the REST API products response
type ProductsResponse struct {
	Count   int `json:"count"`
	Results []struct {
		Code        string `json:"code"`
		DisplayName string `json:"display_name"`
		Brand       string `json:"brand"`
	} `json:"results"`
}

// TariffRate represents a time-varying unit rate
type TariffRate struct {
	ValidFrom    time.Time `json:"valid_from"`
	ValidTo      *time.Time `json:"valid_to"`
	ValueExcVAT  float64   `json:"value_exc_vat"`
	ValueIncVAT  float64   `json:"value_inc_vat"`
}

// TariffRatesResponse represents the REST API tariff rates response
type TariffRatesResponse struct {
	Count   int `json:"count"`
	Next    string `json:"next"`
	Results []struct {
		ValidFrom   string  `json:"valid_from"`
		ValidTo     *string `json:"valid_to"`
		ValueExcVAT float64 `json:"value_exc_vat"`
		ValueIncVAT float64 `json:"value_inc_vat"`
	} `json:"results"`
}

// OpenMeteoResponse represents the response from Open-Meteo historical weather API
type OpenMeteoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Daily     struct {
		Time            []string  `json:"time"`
		TempMax         []float64 `json:"temperature_2m_max"`
		TempMin         []float64 `json:"temperature_2m_min"`
		TempMean        []float64 `json:"temperature_2m_mean"`
		Precipitation   []float64 `json:"precipitation_sum"`
		WeatherCode     []int     `json:"weather_code"`
	} `json:"daily"`
}
