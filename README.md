# octobudget

> A powerful budget analysis tool for Octopus Energy customers

Analyze your Octopus Energy consumption, costs, and tariffs to optimize your energy budget. Perfect for households with solar panels, battery storage, or standard energy setups.

## Features

- 📊 **Comprehensive Analysis** - Track electricity import, solar/battery export, and gas consumption
- 💰 **Smart Payment Recommendations** - Get Direct Debit suggestions with seasonal adjustments (winter +40%, summer baseline)
- ☀️ **Solar/Battery Performance** - Monitor export ratios, grid independence, and earnings with performance ratings
- 🌡️ **Weather-Aware Anomalies** - Understand consumption spikes with automatic weather correlation
- 📋 **Tariff Tracking** - Monitor current and upcoming tariff changes with detailed rate information
- 📄 **Multiple Output Formats** - Beautiful HTML reports or clean Markdown
- 💾 **Local Storage** - Keep historical data for trend analysis and comparisons

## Installation

### Quick Start

```bash
# Clone the repository
git clone https://github.com/matthewgall/octobudget.git
cd octobudget

# Build the application
go build

# Run the analysis
./octobudget
```

### Download Binary

Pre-built binaries are available on the [releases page](https://github.com/matthewgall/octobudget/releases).

## Configuration

### Option 1: Configuration File

Create `config.yaml`:

```yaml
# Required: Your Octopus Energy credentials
account_id: "A-1234ABCD"
api_key: "sk_live_your_api_key_here"

# Optional: Meter configuration (auto-discovered if not provided)
electricity_mpan: "1234567890123"
electricity_serial: "12L3456789"
gas_mprn: "1234567890"
gas_serial: "G4B12345678"

# Optional: Direct Debit amount for payment recommendations
direct_debit_amount: 150

# Optional: Analysis settings
analysis_period_days: 90  # Default: 90 days
anomaly_threshold: 50.0   # Default: 50%
storage_path: "~/.config/octobudget"  # Default storage location
```

### Option 2: Environment Variables

```bash
export OCTOPUS_ACCOUNT_ID="A-1234ABCD"
export OCTOPUS_API_KEY="sk_live_your_api_key_here"
export OCTOPUS_ELECTRICITY_MPAN="1234567890123"
export OCTOPUS_ELECTRICITY_SERIAL="12L3456789"
export OCTOPUS_GAS_MPRN="1234567890"
export OCTOPUS_GAS_SERIAL="G4B12345678"
export OCTOPUS_DIRECT_DEBIT_AMOUNT="150"
```

### Option 3: Command-Line Flags

```bash
./octobudget -account A-1234ABCD -key sk_live_your_api_key_here
```

### Finding Your Credentials

- **Account ID**: Found in your Octopus Energy dashboard (starts with "A-")
- **API Key**: Generate at [octopus.energy/dashboard/developer](https://octopus.energy/dashboard/developer/)
- **MPAN/MPRN**: Found on your energy bills or account dashboard
- **Serial Numbers**: Found on your physical meters or bills

## Usage

### Generate Reports

```bash
# Output to stdout (Markdown)
./octobudget

# Save Markdown report to file
./octobudget -output report.md

# Generate beautiful HTML report
./octobudget -html -output report.html

# Enable debug logging
./octobudget -debug

# Show version
./octobudget -version
```

### Command-Line Options

```
  -config string
        Path to configuration file (default "config.yaml")
  -account string
        Octopus Energy Account ID (overrides config)
  -key string
        Octopus Energy API Key (overrides config)
  -output string
        Output file for report (default: stdout)
  -html
        Generate HTML report instead of Markdown
  -debug
        Enable debug logging
  -version
        Show version and exit
```

## What You Get

Smart analysis of your energy usage with actionable insights:

- **Payment recommendations** with seasonal adjustments (winter +40%, summer baseline) and 10% buffer
- **Solar/battery performance ratings** showing export efficiency, grid independence, and ROI
- **Anomaly detection** with weather correlation to understand unusual consumption
- **Tariff tracking** for current and upcoming rate changes
- **Prioritized recommendations** to optimize costs and usage

Reports available in clean Markdown or beautiful HTML with Octopus Energy brand colors.

## Privacy & Security

**Your data stays on your machine.** octobudget runs entirely locally on your computer:

- API credentials and usage data never leave your PC
- Only communicates directly with Octopus Energy API
- No third-party services or data collection
- All historical data stored locally (default: `~/.config/octobudget/`)
- Open source - verify the code yourself

Your API key is only used to authenticate with the official Octopus Energy API. No telemetry, tracking, or external data sharing.

## How It Works

1. **Data Collection**
   - Fetches account details via Octopus Energy GraphQL API
   - Retrieves consumption data for configured analysis period (default: 90 days)
   - Gets time-varying tariff rates from Products API (for Intelligent Octopus Flux, etc.)
   - Identifies solar/battery export meters automatically

2. **Cost Calculation**
   - Applies time-varying rates to half-hourly consumption data
   - Calculates daily averages and projects monthly costs
   - Computes net costs (import - export + gas)

3. **Weather Enrichment**
   - Fetches historical weather data from Open-Meteo API
   - Correlates temperature and precipitation with consumption spikes
   - Filters out weather-expected anomalies (cold snaps, heat waves)

4. **Analysis & Insights**
   - Detects statistical anomalies (2σ+ deviations)
   - Calculates seasonal Direct Debit recommendations
   - Generates export performance metrics for solar/battery users
   - Creates prioritized, actionable recommendations

5. **Report Generation**
   - Produces formatted Markdown or HTML reports
   - Includes all metrics, tariffs, anomalies, and insights
   - Stores results locally for historical comparison

## Advanced Features

### Auto-Discovery
If you don't specify meter details, octobudget will:
- Automatically discover your electricity import meter
- Detect solar/battery export meters
- Identify gas meters
- Extract current tariff information

### Intelligent Caching
- Account data cached for 1 hour to reduce API calls
- Tariff rates cached based on date ranges
- Weather data cached by date

### Seasonal Payment Adjustment
Direct Debit recommendations account for:
- **Winter (Nov-Feb)**: 40% increase for heating
- **Spring (Mar-Apr) & Autumn (Sep-Oct)**: 20% increase
- **Summer (May-Aug)**: Baseline usage
- **10% safety buffer**: Covers unexpected variations
- **Year-round stability**: Avoid large seasonal swings

### Time-Varying Tariff Support
Full support for dynamic pricing tariffs:
- Intelligent Octopus Flux
- Agile Octopus
- Go
- Any tariff with half-hourly rates

Fetches actual rates from Products API and applies them to your consumption data for accurate cost calculations.

## Troubleshooting

### "Failed to fetch account details"
- Verify your account ID starts with "A-"
- Check your API key is correct and active at [octopus.energy/dashboard/developer](https://octopus.energy/dashboard/developer/)
- Ensure internet connectivity

### "No consumption data available"
- Verify MPAN/MPRN and serial numbers are correct
- Check that your meters are submitting readings (smart meters)
- Ensure analysis period includes dates with available data
- Try running with `-debug` to see detailed API responses

### "Export meter detected but no data"
If you have solar/battery but no export data shows:
- The export meter may use a different serial number
- Check your Octopus account dashboard for the correct serial
- octobudget will try all discovered serial numbers automatically

### Missing tariff rates
For time-varying tariffs (Intelligent Octopus Flux, Agile, etc.):
- Tariff rates are fetched from the Products API
- If rates aren't available, reports will show "Time-varying (see costs in analysis)"
- Check with `-debug` to see if rate fetching is successful

### Enable Debug Logging

```bash
./octobudget -debug
```

This shows:
- API requests and responses
- Data fetching progress
- Analysis stages
- Cache hits/misses
- Weather data integration

## Development

### Building from Source

```bash
go mod download
go build
```

## Support the Project

If you find octobudget useful, here are some ways to support its continued development:

### 💷 Join Octopus Energy

Not an Octopus Energy customer yet? Use my referral link to join and we'll both get £50 credit:

**[Join Octopus Energy - Get £50 credit](https://share.octopus.energy/maize-ape-570)**

This helps fund development of octobudget and you'll get access to:
- Saving Sessions (earn money for reducing usage during peak times)
- Free electricity sessions (completely free electricity during certain periods)
- Competitive energy rates and excellent customer service
- The greenest energy supplier in the UK

### ❤️ GitHub Sponsor

Support ongoing development and maintenance:

**[Become a GitHub Sponsor](https://github.com/sponsors/matthewgall)**

Your sponsorship helps with:
- Adding new features and improvements
- Maintaining compatibility with API changes  
- Providing support and bug fixes
- Keeping the project free and open source

### ⭐ Star the Repository

Show your appreciation by starring the repository on GitHub - it helps others discover the project!

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) file for details.

## Related Projects

- **[octojoin](https://github.com/matthewgall/octojoin)** - Automatically join Octopus Energy Saving Sessions and earn OctoPoints

## Disclaimer

This is an unofficial third-party application. "Octopus Energy" is a trademark of Octopus Energy Group Limited. This application is not affiliated with, endorsed by, or connected to Octopus Energy.

All projections and recommendations are based on historical data and may not reflect future usage or costs. Always verify information with your official Octopus Energy bills and account statements.
