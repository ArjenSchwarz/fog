# Deploy Command Multi-Format Output Support

## Introduction

This feature enhances the `fog deploy` command to support multiple output formats with proper stream separation. Currently, the deploy command outputs a streaming table to stdout showing state changes in real-time. This enhancement will:

1. Move streaming progress output to stderr (following Unix conventions)
2. Output final deployment results to stdout in the specified format (JSON, CSV, YAML, Markdown, table, etc.)
3. Enable users to redirect final results to files while still seeing progress in the terminal

The feature maintains backwards compatibility by preserving the streaming table behavior on stderr and using table format as the default.

## Requirements

### 1. Backwards Compatibility

**User Story:** As an existing fog user, I want the deploy command to behave exactly as it does today when I use the default table format, so that my existing workflows and scripts are not disrupted.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL use table format as the default output format
2. <a name="1.2"></a>The system SHALL write streaming progress output to stderr for all deployment modes
3. <a name="1.3"></a>The system SHALL write final deployment results to stdout in the specified format
4. <a name="1.4"></a>The system SHALL maintain current behavior for stderr streaming output (table format showing state changes)

### 2. Output Format Support

**User Story:** As a fog user, I want to specify an output format for deployment results, so that I can process deployment information programmatically or in my preferred format.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL support all existing fog output formats including JSON, CSV, YAML, Markdown, and table
2. <a name="2.2"></a>The system SHALL accept output format specification via CLI flag (--output)
3. <a name="2.3"></a>The system SHALL accept output format specification via configuration file
4. <a name="2.4"></a>The system SHALL accept output format specification via environment variable
5. <a name="2.5"></a>The system SHALL document the Markdown output format in the official documentation
6. <a name="2.6"></a>The system SHALL follow the standard fog precedence order for format specification (CLI flag > env var > config file)

### 3. Output Stream Separation

**User Story:** As a fog user, I want progress information and data output to be separated into different streams, so that I can redirect data output while still seeing progress in the terminal.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL write all progress and diagnostic information to stderr
2. <a name="3.2"></a>The system SHALL write final formatted output to stdout for all deployment outcomes (successful, failed, no-changes, dry-run, create-changeset)
3. <a name="3.3"></a>Progress information to stderr SHALL include: streaming table output, stack information, changeset overview, deployment progress updates, and interactive prompts
4. <a name="3.4"></a>The system SHALL enable users to redirect stdout to a file while still seeing progress information in the terminal
5. <a name="3.5"></a>Interactive prompts (e.g., "Do you want to continue?") SHALL be written to stderr and read from stdin

### 4. Streaming Output Behavior

**User Story:** As a fog user, I want to see real-time progress of my deployment in the terminal, so that I can monitor what's happening even when requesting formatted output.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL display streaming progress output to stderr during deployment unless quiet mode is enabled
2. <a name="4.2"></a>The system SHALL display streaming output even when a non-table output format is specified
3. <a name="4.3"></a>The system SHALL display streaming output even when outputting to a file
4. <a name="4.4"></a>The system SHALL maintain the current streaming output format and behavior
5. <a name="4.5"></a>The streaming output SHALL show state changes in real-time as they occur
6. <a name="4.6"></a>Stack information and changeset overview shown before deployment SHALL be written to stderr as contextual information
7. <a name="4.7"></a>The system SHALL render all stderr streaming output in table format regardless of the user's output format preference

### 5. Quiet Mode

**User Story:** As a CI/CD automation engineer, I want to suppress all progress output and see only the final result, so that my pipeline logs are clean and focused on the deployment outcome.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL provide a `--quiet` flag to suppress all stderr output
2. <a name="5.2"></a>When quiet mode is enabled, the system SHALL NOT output streaming progress to stderr
3. <a name="5.3"></a>When quiet mode is enabled, the system SHALL NOT output stack information or changeset overview to stderr
4. <a name="5.4"></a>When quiet mode is enabled, the system SHALL NOT display interactive prompts
5. <a name="5.5"></a>When quiet mode is enabled, the system SHALL automatically enable non-interactive mode (auto-approve all prompts)
6. <a name="5.6"></a>When quiet mode is enabled, the system SHALL output the final formatted result to stdout
7. <a name="5.7"></a>The system SHALL document quiet mode usage for CI/CD scenarios

### 6. Final Formatted Output - Dry Run and Create Changeset

**User Story:** As a fog user performing a dry-run deployment or creating a changeset, I want to receive the changeset details in my specified format, so that I can analyze proposed changes programmatically.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL output the changeset result to stdout after streaming output completes for dry-run deployments
2. <a name="6.2"></a>The system SHALL output the changeset result to stdout after streaming output completes when using `--create-changeset` flag
3. <a name="6.3"></a>The dry-run and create-changeset output SHALL be identical to the output of the `describe changeset` command by reusing the same output builder function
4. <a name="6.4"></a>The system SHALL NOT include additional metadata beyond what `describe changeset` provides
5. <a name="6.5"></a>The system SHALL format the changeset output according to the specified output format
6. <a name="6.6"></a>When using `--create-changeset`, the changeset SHALL NOT be deleted after output is produced

### 7. Final Formatted Output - Successful Deployment

**User Story:** As a fog user, I want to receive a summary of my successful deployment in my specified format, so that I can record and process the deployment results.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL output a deployment summary to stdout after streaming output completes for successful deployments
2. <a name="7.2"></a>The summary SHALL include the final stack status
3. <a name="7.3"></a>The summary SHALL include the planned changes from the changeset (resources that were created/updated/deleted)
4. <a name="7.4"></a>The summary SHALL include deployment metadata (timestamps, stack ARN, changeset ID)
5. <a name="7.5"></a>The summary SHALL include stack outputs
6. <a name="7.6"></a>The system SHALL format the deployment summary according to the specified output format
7. <a name="7.7"></a>The data structure for deployment summary output SHALL use the same structure as the current table-based output

### 8. Final Formatted Output - No Changes

**User Story:** As a fog user, I want to receive notification when no changes are found, so that I understand why no deployment occurred.

**Acceptance Criteria:**

1. <a name="8.1"></a>The system SHALL output a no-changes message to stdout when CloudFormation determines there are no changes to apply
2. <a name="8.2"></a>The no-changes output SHALL include a message stating that no changes were found
3. <a name="8.3"></a>The no-changes output SHALL include current stack information
4. <a name="8.4"></a>The system SHALL format the no-changes output according to the specified output format
5. <a name="8.5"></a>The system SHALL treat no-changes scenario as a successful outcome (exit code 0)

### 9. Final Formatted Output - Failed Deployment

**User Story:** As a fog user, I want to receive error details in my specified format when a deployment fails, so that I can programmatically process and respond to deployment failures.

**Acceptance Criteria:**

1. <a name="9.1"></a>The system SHALL output error details to stdout after streaming output completes for failed deployments
2. <a name="9.2"></a>The error output SHALL include CloudFormation error messages
3. <a name="9.3"></a>The error output SHALL include information about which resources failed and why
4. <a name="9.4"></a>The error output SHALL include the final stack status after rollback (if applicable)
5. <a name="9.5"></a>The error output SHALL include stack-level StatusReason from CloudFormation
6. <a name="9.6"></a>The system SHALL format the error output according to the specified output format
7. <a name="9.7"></a>The data structure for failed deployment output SHALL use the same structure as the current table-based error output

### 10. Output Timing and Location

**User Story:** As a fog user, I want the formatted output to appear after the deployment completes, so that I receive the final state rather than intermediate states.

**Acceptance Criteria:**

1. <a name="10.1"></a>The system SHALL output the formatted summary only after the deployment process has completely finished
2. <a name="10.2"></a>The system SHALL output formatted data to stdout when no file target is specified
3. <a name="10.3"></a>The system SHALL support file output using the existing --file flag mechanism (behavior follows existing fog conventions)
4. <a name="10.4"></a>The system SHALL maintain streaming progress output to stderr even when formatted output is directed to a file
5. <a name="10.5"></a>The system SHALL flush stderr before writing final formatted output to stdout to ensure proper ordering
6. <a name="10.6"></a>The system SHALL visually separate stderr streaming output from stdout final output by writing a header to stdout before the formatted data begins

### 11. Error Handling

**User Story:** As a fog user, I want to be notified if the formatted output generation fails, so that I'm aware that the deployment summary may be incomplete or unavailable.

**Acceptance Criteria:**

1. <a name="11.1"></a>The system SHALL treat output generation failures as warnings, not command failures
2. <a name="11.2"></a>The system SHALL write error messages about output generation failures to stderr
3. <a name="11.3"></a>The system SHALL determine command success/failure based on deployment outcome, not output generation
4. <a name="11.4"></a>The system SHALL validate that the output format is supported during flag parsing before any AWS operations
5. <a name="11.5"></a>The system SHALL handle serialization errors gracefully with descriptive error messages
6. <a name="11.6"></a>The system SHALL handle file system errors (invalid path, permission denied, disk full) with clear error messages
7. <a name="11.7"></a>Pre-deployment failures (authentication errors, invalid templates, validation failures, flag errors) SHALL write error messages to stderr only
8. <a name="11.8"></a>Pre-deployment failures SHALL exit with non-zero code without producing formatted stdout output
9. <a name="11.9"></a>User cancellation (Ctrl+C) during deployment SHALL exit immediately without producing formatted stdout output

### 12. Data Consistency

**User Story:** As a fog user, I want the formatted output to accurately reflect the deployment results, so that I can trust the data for automation and record-keeping.

**Acceptance Criteria:**

1. <a name="12.1"></a>The system SHALL ensure formatted output data matches the actual deployment state
2. <a name="12.2"></a>The system SHALL capture changeset data immediately after creation for use in final output
3. <a name="12.3"></a>The system SHALL only delete changesets created during dry-run or when deployment is cancelled after review
4. <a name="12.4"></a>The system SHALL allow CloudFormation to handle changeset deletion for executed deployments
5. <a name="12.5"></a>The system SHALL use the same data sources for both streaming and formatted output where applicable
6. <a name="12.6"></a>The formatted output SHALL reflect the state at the time of deployment completion, not intermediate states

### 13. Documentation

**User Story:** As a fog user, I want clear documentation on how to use multi-format output, so that I can effectively utilize this feature.

**Acceptance Criteria:**

1. <a name="13.1"></a>The system SHALL document all supported output formats including the previously undocumented Markdown format
2. <a name="13.2"></a>The documentation SHALL include examples of formatted output for each supported format
3. <a name="13.3"></a>The documentation SHALL explain the difference between streaming progress output (stderr) and final formatted output (stdout)
4. <a name="13.4"></a>The documentation SHALL provide examples of using output format with file targets and redirection
5. <a name="13.5"></a>The documentation SHALL clarify when final formatted output is produced versus when it is not
6. <a name="13.6"></a>The documentation SHALL explain how to redirect stdout to a file while seeing progress in the terminal
