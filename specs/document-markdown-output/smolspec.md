# Document Markdown Output Format

## Overview
The fog CLI application supports markdown (along with yaml and html) as output formats through the go-output v2 library. The implementation is complete and functional - markdown support is wired through config.getFormatForOutput() and works for all commands that use the output system (exports, resources, dependencies, drift, history, describe changeset, report). However, user-facing documentation inconsistently mentions these formats. Specifically, the global --output flag help text only lists "table, csv, json, and dot" when it should also mention "yaml, markdown, html" to accurately reflect the available options.

## Requirements
- Update cmd/root.go line 131 to change the --output flag description from "currently supported are table, csv, json, and dot" to include yaml, markdown, and html
- Update cmd/root.go line 58 (package documentation comment) to include yaml, markdown, html in the output format list
- Update README.md line 68 to include yaml, markdown, html in the global flags output format description
- Update example-fog.yaml line 4 comment to include yaml, markdown, html in the list of choices
- Verify that docs/user-guide/configuration-reference.md correctly lists all formats (it already does at line 40)
- Verify that command-specific long descriptions that mention output formats are accurate (report.go already correctly references markdown and html at lines 40-52)

## Implementation Approach
This is a documentation-only change with no code modifications to the output logic:
- Key files to modify: cmd/root.go (2 locations), README.md (1 location), example-fog.yaml (1 location)
- Search pattern: Look for the specific string "table, csv, json, and dot" which appears in the outdated help text
- The complete format list should be: "table, csv, json, yaml, markdown, html, and dot (for certain functions)"
- Order rationale: Grouped by type - table formats (table), delimiter-separated (csv), structured data (json, yaml), markup (markdown, html), graph (dot)
- Note: docs/user-guide/configuration-reference.md already correctly documents all formats at line 40, no changes needed there
- Note: cmd/report.go already correctly mentions markdown and html support in its long description, no changes needed
- Focus only on the global --output flag documentation and the example config file
