# fog

Fog is a tool to manage your CloudFormation deployments and ensure you have all your config as code. Please note, fog is **not** a DSL. It only works with standard CloudFormation files and doesn't do anything you can't do using the AWS CLI. It just makes it easier by combining functionality and preventing you from needing to deal with unnecesary overhead like different commands for creating a new stack or updating an existing one. In addition, it has little helper functionalities like offering to remove an empty stack for you.

## Deployment workflow

The below diagram shows the flow that fog goes through when doing a deployment.

![](docs/deployment-flow.drawio.svg)