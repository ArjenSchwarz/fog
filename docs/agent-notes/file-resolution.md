# File Resolution (lib/files.go)

## How ReadFile works

`ReadFile(fileName, fileType)` resolves a file through a two-step process:

1. **Direct path check**: Try `os.Stat(fileName)` as-is. If found, read it.
2. **Directory lookup** (if step 1 fails):
   a. Try `<configured-directory>/<fileName>` (bare name, no extension appended)
   b. For each configured extension, try `<configured-directory>/<fileName><extension>`

Configuration comes from Viper: `<fileType>.directory` and `<fileType>.extensions`.

## Wrapper functions

- `ReadTemplate` - uses `templates` config key
- `ReadTagsfile` - uses `tags` config key
- `ReadParametersfile` - uses `parameters` config key
- `ReadDeploymentFile` - uses `deployments` config key

## Gotchas

- The bare-name check (step 2a) was added in T-529. Without it, filenames that already include an extension (e.g., `my-template.yaml`) get double-appended extensions (e.g., `my-template.yaml.yaml`).
- The function takes `*string` for fileName, not a value. The wrapper functions handle the address-of conversion.
- File type extensions in config include the dot (e.g., `.yaml`, `.json`).
