fabrik
======

*Infrastructure as code.*

Fabrik is a serverless continuous integration and deployment ("CI/CD") orchestrator for services built on AWS. It allows your service's build, test, and deployment lifecycle to be completely defined *as code*, and live right alongside the service implementation. This allows the lifecycle to be completely automated, tested in isolation from other deployments of the same service, and most importantly, *reliable*.

## Getting Started

Before setting up Fabrik in your AWS account, it's important to know that it isn't designed to be a "plug-and-play" system. For it to be used effectively, it requires having in-depth knowledge of your deployment architecture, how various
components integrate with one another, and how those components share resources. Fabrik simply provides a framework and
set of conventions to take that knowledge, and turn it into a repeatable and reliable process.

At a minimum, Fabrik assumes you are deploying to AWS, and have a basic working knowledge of CloudFormation. If you haven't spent much time with CloudFormation, don't worry, I try to explain the high-level concepts where appropriate in the [examples](./examples/). The CloudFormation [Resource and Property Type Reference](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html) will be your best friend.

### Prerequisites

* [`aws`](https://aws.amazon.com/cli/) CLI utility
* [`docker`](https://docs.docker.com/install/) daemon
* `make`

### Configure `aws`

Add a profile to `$HOME/.aws/credentials` that will serve as your fabrik deployment identity in AWS. This should
be a set of access credentials that have administrator privileges. It's a good idea to rotate this key on
a regular basis. Access keys can be created at the [IAM console](https://console.aws.amazon.com/iam/home?#/users).

```
[fabrik]
aws_access_key_id = AKIA...
aws_secret_access_key = O4vew...
region = <any valid region> # i.e. us-west-2
output = json
```

### Clone

For now, the primary deployment mechanism is from a local clone of this repository.

`$ git clone git@github.com:ngmiller/fabrik.git`

### Install

Build the Docker image that will provide a runtime environment for fetching dependencies and the [`serverless`](https://serverless.com/) deployment.

`$ make image`

### Build

Fetch dependencies

`$ make deps`

Build each Lambda function

`$ make build`

### Configure `fabrik`

Fabrik needs two secret keys to interact with GitHub. These should be stored and encrypted using AWS

|Key|Description|
|---|-----------|
|`fabrik.github.hmac`|GitHub OAuth token with `repo` scope|
|`fabrik.github.token`|GitHub HMAC key used in webhook configuration|

### Deploy

Deploy the entire stack defined in `serverless.yml`

`$ make deploy`

**NOTE: This can be used for updating Fabrik as well. Simply pull the latest version of the repo, and rerun this command.**

### SSM Parameters

We utilize [AWS SSM](https://us-west-2.console.aws.amazon.com/systems-manager/parameters) for secure parameter storage. Values are encrypted at rest using a KMS key.

Set a parameter like so,

```
$ aws --profile opolis ssm put-parameter \
    --type SecureString \
    --name {name} \
    --value $(cat {file}) \
    --key-id 344d9fba-07d2-45c8-9bde-2356aaedc6c3
```

*The secret value should first be written to a temporary file to avoid storing the value in shell history.*

## Adding a Repository

See [`example/`](./example/)

#### Doesn't AWS have a product that does this already?

Yes and no. CloudFormation, CodeBuild, and CodePipeline provide the majority of the heavy lifting Fabrik doesn't do itself,
but there are certain properties I wanted my deployments to have, like unlimited isolated deployments, that I wasn't
getting with those products on their own.

As far as I can tell, CodePipeline only allows a user to configure a single
source repository branch to be the target for processing, meaning that if I setup a CodePipeline instance to point
at `master`, a push to `my-feature-branch` will not be processed. Fabrik fixes this by automating the provisioning
of a new CodePipeline for each branch created, including `master`. Tagged builds reuse the same `production` pipeline.

