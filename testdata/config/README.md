# Config Test Fixtures

This directory contains test fixtures for the config package tests.

## Files

### Valid Configurations

- **valid-config.yaml**: Standard YAML configuration with common settings
  - Region: us-west-2
  - Profile: default
  - Output: table
  - Timezone: UTC

- **valid-config.json**: JSON configuration with production-like settings
  - Region: us-east-1
  - Profile: production
  - Output: json with file output
  - Timezone: America/New_York

- **valid-config.toml**: TOML configuration with European settings
  - Region: eu-west-1
  - Profile: staging
  - Output: csv with file output
  - Timezone: Europe/London

- **minimal-config.yaml**: Minimal valid configuration with just region
  - Region: us-west-2
  - Tests default value handling

### Invalid Configurations

- **invalid-config.yaml**: Malformed YAML with unclosed bracket
  - Used to test error handling during configuration parsing

## Usage

These fixtures are used by tests in `config/config_test.go` and `config/awsconfig_test.go` to verify:
- Configuration loading from different formats (YAML, JSON, TOML)
- Proper parsing of all configuration options
- Error handling for invalid configurations
- Default value handling for minimal configurations

## Adding New Fixtures

When adding new test fixtures:
1. Create a descriptive filename indicating the test scenario
2. Document the purpose in this README
3. Include comments in the fixture file explaining its contents
4. Reference the fixture in the appropriate test file
