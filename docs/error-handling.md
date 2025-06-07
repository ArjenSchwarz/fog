# Fog Error Handling

Fog uses a structured error system so commands and services can return rich error
information. The key type is `FogError` which wraps an error code with
contextual data.

## FogError overview

`FogError` is defined in `cmd/errors/types.go` and exposes methods for
retrieving the error code, message and metadata such as the originating
operation and component. A simplified example:

```go
err := errors.NewError(errors.ErrStackNotFound, "Stack 'my-stack' not found").
    WithOperation("deploy").
    WithComponent("cloudformation")
```

Every error includes:

- a machine readable `ErrorCode`
- severity and retryable information
- context fields for debugging
- optional user facing suggestions

## Error codes

Error codes are grouped by concern in `cmd/errors/types.go`. Functions in
`cmd/errors/codes.go` map codes to categories and severity levels.
Metadata can be retrieved with `GetErrorMetadata`:

```go
meta := errors.GetErrorMetadata(errors.ErrTemplateNotFound)
```

Refer to the source for the full list of codes.

### Adding a new error code

1. Declare a new constant in the appropriate section of `types.go`.
2. Update `GetErrorCategory`, `GetErrorSeverity` and `IsRetryable` in
   `codes.go` so the new code is classified correctly.
3. Provide descriptive text and suggestions by extending `GetErrorMetadata`.
4. Add unit tests in `cmd/errors` verifying the classification and metadata.

## Error handling middleware

All commands use `ErrorHandlingMiddleware` (`cmd/middleware/error_handler.go`).
It converts unknown errors to `FogError`, formats them and prints the result via
the configured UI handler. The middleware respects the error severity when
choosing whether to show the message as info, warning or error.

## Validation helpers

`cmd/validation/errors.go` provides `ValidationErrorBuilder` for collecting
multiple validation issues:

```go
builder := validation.NewValidationErrorBuilder("deploy")
builder.RequiredField("stack-name").InvalidValue("region", "bad", "unknown")
if builder.HasErrors() {
    return builder.Build()
}
```

When multiple errors are present `Build()` returns a `MultiError` so the
middleware can display them together.

## Verbose output

Pass `--verbose` (or `-v`) to any command to include additional error context
and stack traces in the console output. This flag is defined on the root command
and applies to all sub-commands.
