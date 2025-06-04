# Deploying CloudFormation Stacks

Fog's primary command is `fog deploy`. It takes a template and optional parameter and tag files to create or update a stack through a Change Set. The command validates your input, optionally runs precheck commands and then asks for confirmation before the deployment proceeds. During the deployment Fog streams the stack events so you can follow progress in real time.

```mermaid
flowchart TD
    Start([Start]) --> CheckDF{Deployment file provided?}
    CheckDF -- Yes --> LoadDF[Load deployment file]
    CheckDF -- No --> LoadFiles[Load template, parameters and tags]
    LoadDF --> Prechecks{Run prechecks?}
    LoadFiles --> Prechecks
    Prechecks -- Yes --> RunPre[Execute precheck commands]
    Prechecks -- No --> CreateCS
    RunPre --> Passed{Prechecks passed?}
    Passed -- No --> StopCheck{Stop on failure?}
    StopCheck -- Yes --> ExitFail([Exit])
    StopCheck -- No --> CreateCS
    Passed -- Yes --> CreateCS
    CreateCS[Create change set] --> HasChanges{Changes detected?}
    HasChanges -- No --> ExitNoChanges[Exit]
    HasChanges -- Yes --> ShowCS[Show change set summary]
    ShowCS --> Approve{Deploy change set?}
    Approve -- No --> DeleteCS[Delete change set]
    Approve -- Yes --> DeployCS[Deploy change set]
    DeployCS --> Monitor[Monitor stack events]
    Monitor --> Success{Deployment successful?}
    Success -- Yes --> Outputs[Show stack outputs]
    Success -- No --> Failed[Show failed events]
    Failed --> NewStack{Was this a new stack?}
    NewStack -- Yes --> OfferDelete[Offer to delete stack]
    OfferDelete --> End
    NewStack -- No --> End
    Outputs --> End
    DeleteCS --> End
    ExitFail --> End
    ExitNoChanges --> End
```

This flow illustrates the high level steps Fog takes when deploying a stack.
