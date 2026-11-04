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
- Deployment file `template-file-path` (T-1414): `LoadDeploymentFile` rewrites a relative path to be joined with the loaded deployment file's directory, and `readTemplateFromSource` then reads that exact path with `os.ReadFile`. It deliberately does NOT go through `ReadFile(..., "templates")`: that fallback would silently pick e.g. `templates/deployments/missing.yaml` when the anchored file is missing. Before the fix, the path was resolved from cwd/`templates.directory`, which only worked by accident (e.g. `templates` + `../templates/x.yaml` cleans to `templates/x.yaml`).
