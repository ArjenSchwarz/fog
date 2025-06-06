# Command Structure Migration Guide

## Overview
This directory contains the new command structure for the fog CLI application.

## Structure
Each command is organized in its own directory with:
- `command.go` - Command builder and definition
- `handler.go` - Business logic handler
- `flags.go` - Flag definitions and validation

## Migration Status
- [x] Deploy command - Refactored with new structure
- [ ] Drift command - TODO
- [ ] Describe command - TODO
- [ ] Dependencies command - TODO
- [ ] Exports command - TODO
- [ ] History command - TODO
- [ ] Report command - TODO
- [ ] Resources command - TODO

## Adding New Commands
1. Create directory under `cmd/commands/`
2. Implement the three required files
3. Register in `cmd/root.go`
4. Add tests in `cmd/testing/`

