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

const (
	// OctopusGraphQLEndpoint is the Octopus Energy GraphQL API endpoint
	OctopusGraphQLEndpoint = "https://api.octopus.energy/v1/graphql/"

	// OctopusRESTAPIBase is the base URL for REST API endpoints
	OctopusRESTAPIBase = "https://api.octopus.energy/v1"
)

// GraphQL query to obtain JWT token
const obtainJWTTokenQuery = `
mutation obtainKrakenToken($apiKey: String!) {
  obtainKrakenToken(input: { APIKey: $apiKey }) {
    token
  }
}
`

// GraphQL query to fetch account details
const accountDetailsQuery = `
query AccountDetails($accountNumber: String!) {
  account(accountNumber: $accountNumber) {
    number
    balance
    properties {
      id
      electricityMeterPoints {
        mpan
        meters {
          serialNumber
          smartDevices {
            deviceId
          }
        }
        agreements {
          validFrom
          validTo
          tariff {
            ... on TariffType {
              displayName
              fullName
              standingCharge
            }
            ... on StandardTariff {
              unitRate
            }
            ... on DayNightTariff {
              dayRate
              nightRate
            }
            ... on PrepayTariff {
              unitRate
            }
          }
        }
      }
      gasMeterPoints {
        mprn
        meters {
          serialNumber
          smartDevices {
            deviceId
          }
        }
        agreements {
          validFrom
          validTo
          tariff {
            ... on TariffType {
              displayName
              fullName
              standingCharge
            }
            ... on GasTariffType {
              unitRate
            }
          }
        }
      }
    }
  }
}
`

// GraphQL query to fetch electricity consumption using measurements
const electricityConsumptionQuery = `
query ElectricityConsumption($accountNumber: String!, $mpan: String!, $deviceId: String!, $startAt: DateTime!, $endAt: DateTime!) {
  account(accountNumber: $accountNumber) {
    properties {
      id
      electricityMeterPoints {
        mpan
        meters {
          serialNumber
          smartDevices {
            deviceId
          }
        }
        agreements {
          validFrom
          validTo
          tariff {
            ... on TariffType {
              displayName
              standingCharge
            }
            ... on StandardTariff {
              unitRate
            }
            ... on PrepayTariff {
              unitRate
            }
          }
        }
      }
      measurements(
        first: 1000
        utilityFilters: [{
          electricityFilters: {
            readingFrequencyType: RAW_INTERVAL
            marketSupplyPointId: $mpan
            readingDirection: CONSUMPTION
            deviceId: $deviceId
          }
        }]
        startAt: $startAt
        endAt: $endAt
        timezone: "Europe/London"
      ) {
        edges {
          node {
            value
            unit
            ... on IntervalMeasurementType {
              startAt
              endAt
            }
            metaData {
              statistics {
                costInclTax {
                  estimatedAmount
                  costCurrency
                }
              }
            }
          }
        }
      }
    }
  }
}
`

// GraphQL query to fetch gas consumption using measurements
const gasConsumptionQuery = `
query GasConsumption($accountNumber: String!, $mprn: String!, $deviceId: String!, $startAt: DateTime!, $endAt: DateTime!) {
  account(accountNumber: $accountNumber) {
    properties {
      id
      gasMeterPoints {
        mprn
        meters {
          serialNumber
          smartDevices {
            deviceId
          }
        }
        agreements {
          validFrom
          validTo
          tariff {
            ... on TariffType {
              displayName
              standingCharge
            }
            ... on GasTariffType {
              unitRate
            }
          }
        }
      }
      measurements(
        first: 1000
        utilityFilters: [{
          gasFilters: {
            readingFrequencyType: RAW_INTERVAL
            marketSupplyPointId: $mprn
            readingDirection: CONSUMPTION
            deviceId: $deviceId
          }
        }]
        startAt: $startAt
        endAt: $endAt
        timezone: "Europe/London"
      ) {
        edges {
          node {
            value
            unit
            ... on IntervalMeasurementType {
              startAt
              endAt
            }
            metaData {
              statistics {
                costInclTax {
                  estimatedAmount
                  costCurrency
                }
              }
            }
          }
        }
      }
    }
  }
}
`

// GraphQL query to fetch billing statements
const billingStatementsQuery = `
query BillingStatements($accountNumber: String!) {
  account(accountNumber: $accountNumber) {
    statements {
      id
      issuedDate
      fromDate
      toDate
      totalAmount
      chargeAmount
      paymentAmount
      outstandingBalance
    }
  }
}
`

// GraphQL query to fetch payment history
const paymentHistoryQuery = `
query PaymentHistory($accountNumber: String!) {
  account(accountNumber: $accountNumber) {
    payments {
      id
      amount
      paymentDate
      method
    }
  }
}
`
