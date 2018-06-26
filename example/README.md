Adding a Repository
===================

Repositories can be watched by the build system by adding a few files to the repo root directory,
and configuring a GitHub webhook.

## Required Files

The build system expects the following files to be present in the root of the repository when it picks up
a `push` event from GitHub. (These files do not have to be present in the `master` branch right away. The build
system will simply see them as part of your branch, allowing you to test the entire deployment lifecycle in isolation
from the rest of the work in the repository.)

The following is a list of required files, along with their description. Feel free to follow the links, copy them
to your repository, and adapt as needed.

### [`pipeline.json`](./pipeline.json)

Defines the entire CI/CD pipeline as a
[CloudFormation](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/Welcome.html) template.

Each template is *required* to accept the following parameters. If any are missing, the prepartion phase
of the build system will fail. These parameters are provided by the build system at runtime, you are not
responsible for specifying them, just including and referencing them in the template.

|Key|Value|
|---|-----|
|`ArtifactStore`|Name of the S3 bucket responsible for storing pipeline artifacts|
|`RepoOwner`|GitHub repo namespace, i.e. `opolis`|
|`RepoName`|GitHub repo name|
|`RepoBranch`|Branch name to build|
|`RepoToken`|OAuth token with `repo` scope|
|`Stage`|Used to reference pipeline parameters `development`, `master`, or `release`|

#### Dockerfile

Each pipeline file should specify an AWS CodeBuild project that defines the build environment
and docker image used during the build stage. A development image should be defined in the repo's `Dockerfile`,
built locally, and pushed to ECR at the path,

`{ACCOUNT_ID}.dkr.ecr.{REGION}.amazonaws.com/{REPO_OWNER}/{REPO_NAME}:dev`

### [`parameters.json`](./parameters.json)

Defines a set of parameters to set during a particular _invocation_ of the build pipeline. There are three
invocation types: `development`, `master`, and `release`. The `development` invocation occurs for every branch
pushed to the repository, `master` occurs after a merge into master, and `release` occurs on every tag.
The parameters file is keyed accordingly, and each set of parameters must be a list of objects in the form:

```
{
    "ParameterKey": "...",
    "ParameterValue": "..."
}
```

### [`buildspec.yml`](./buildspec.yml)

Defines the build steps and commands run inside the Docker container defined by `Dockerfile`. CodeBuild
reads this file automatically, running each command in sequence, failing the entire build if any command
fails. If the build is successful, the artifacts are pushed the S3 bucket specified by `ArtifactStore`,
and can be used in a subsequent stage in the pipeline.

## Configuring the webhook

"WebHooks" are a means for GitHub to notify third party services that a particular event has occurred on a particular
repository. An event can be anything from opening a pull request, to merging into master. For our purposes,
we are only interested in what are known as `push` events.

After deploying the serverless project in this repo, make note of the API Gateway endpoint,

```
Serverless: Packaging service...
Serverless: Excluding development dependencies...
Serverless: Uploading CloudFormation file to S3...
Serverless: Uploading artifacts...
Serverless: Uploading service .zip file to S3 (10.31 MB)...
Serverless: Validating template...
Serverless: Updating Stack...
Serverless: Checking Stack update progress...
..............
Serverless: Stack update finished...
Service Information
service: opolis-build
stage: dev
region: us-west-2
stack: opolis-build-dev
api keys:
  None
endpoints:
  POST - https://xxxxxxx.execute-api.us-west-2.amazonaws.com/dev/event <------- *
functions:
  listener: opolis-build-dev-listener
  builder: opolis-build-dev-builder
  notifier: opolis-build-dev-notifier
Serverless: Removing old service versions...
```

On the GitHub repo page, go to "Settings" > "Webhooks" > "Add webhook". Provide the API Gateway endpoint as the hook
destination, the HMAC key set as a build system parameter during setup, and select "Just the `push` event".

And that's it! Check out the status updates on each commit pushed to GitHub to track that commit's
progress through the build system.
