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

Each template is required to accept the following parameters. If any are missing, the prepartion phase
of the build system will fail. These parameters are provided by the build system at runtime, you are not
responsible for specifying them, just including and referencing them in the template.

|Key|Value|
|---|-----|
|`ArtifactStore`|Name of the S3 bucket responsible for storing pipeline artifacts|
|`RepoOwner`|GitHub repo namespace, i.e. `opolis`|
|`RepoName`|GitHub repo name|
|`RepoBranch`|Branch name to build|
|`RepoToken`|OAuth token with `repo` scope|

### [`parameters.json`](./parameters.json)

### [`buildspec.yml`](./buildspec.yml)

## Configuring the webhook

"WebHooks" are a means for GitHub to notify third party services that a particular event has occurred on a particular
repository. An event can be anything from opening a pull request, to merging into master. For our purposes,
we are only interested in what are known as `push` events.
