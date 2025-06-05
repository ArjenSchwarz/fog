# Refactoring the Deploy Command

The `deploy` command is the largest entry point in the project. The main function
`deployTemplate` currently spans more than 250 lines and performs a variety of
tasks ranging from flag validation to streaming stack events.  Refactoring this
function will make the code easier to maintain and unit test.

## Suggested structure

1. **Validation and setup**
   - Create `prepareDeployment` in a new file `cmd/deploy_helpers.go`.
   - This function validates all flags, loads the AWS config and populates a
     `lib.DeployInfo` instance.
   - Return both the deployment info and the `config.AWSConfig` object so the
     caller does not need to know how these are created.
   - Example:

     ```go
     func prepareDeployment() (lib.DeployInfo, config.AWSConfig, error) {
         if err := deployFlags.Validate(); err != nil {
             return lib.DeployInfo{}, config.AWSConfig{}, err
         }
         awsCfg, err := config.DefaultAwsConfig(*settings)
         if err != nil {
             return lib.DeployInfo{}, config.AWSConfig{}, err
         }
         info := lib.DeployInfo{StackName: deployFlags.StackName}
         // additional fields populated here
         return info, awsCfg, nil
     }
     ```
2. **Pre-check handling**
   - Implement `runPrechecks(info *lib.DeployInfo, cfg config.AWSConfig, log *lib.DeploymentLog)`.
   - Execute the commands from `viper.GetStringSlice("templates.prechecks")` and
     update `info.PrechecksFailed` and `log.PreChecks` based on the results.
   - Return any output so that the caller can decide how to display it.
3. **Change set logic**
   - Move change set creation and display logic into `createAndShowChangeset`.
   - This helper returns the generated `ChangesetInfo` and appends it to the
     deployment log.
   - It should also delete the change set when running in dry‑run mode.
4. **Deployment confirmation**
   - Encapsulate user prompts and the final deployment call inside
     `confirmAndDeployChangeset`.
   - This function should return a boolean indicating if the stack was actually
     deployed so the caller can skip result processing when not executed.
5. **Result handling**
   - Move stack result checks and output rendering into `printDeploymentResults`.
   - Fetch the final stack state, print outputs and record success or failure in
     the `DeploymentLog`.

## Putting it together

Once these helpers exist the `deployTemplate` function becomes a thin wrapper:

```go
func deployTemplate(cmd *cobra.Command, args []string) {
    info, cfg, err := prepareDeployment()
    if err != nil {
        fmt.Print(outputsettings.StringFailure(err.Error()))
        os.Exit(1)
    }
    log := lib.NewDeploymentLog(cfg, info)

    precheckOutput := runPrechecks(&info, cfg, log)
    fmt.Println(precheckOutput)

    changeset := createAndShowChangeset(&info, cfg, log)
    if confirmAndDeployChangeset(changeset, &info, cfg) {
        printDeploymentResults(&info, cfg, log)
    }
}
```

This refactor keeps the flow of the existing command intact while giving each
piece a well defined responsibility.

Breaking the function up along these lines will isolate concerns and keep each piece focused on a single responsibility. This improves readability and makes future testing easier.
