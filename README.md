build
-----

Serverless CI/CD

## Configure

Add an `opolis` profile to `~/.aws/credentials`

```
[opolis]
aws_access_key_id = AKIA...
aws_secret_access_key = O4vew...
region = us-west-2
output = json
```

## Install

Build the development Docker image

`$ make image`

## Build

Build each Lambda function

`$ make build`

## Deploy

Deploy to Lambda and API Gateway

`$ make deploy`

### Setting SSM Parameters

We utilize AWS SSM for secure parameter storage. Values are encrypted at rest using a KMS key.

Set a parameter like so,

```
$ aws --profile opolis ssm put-parameter \
    --type SecureString \
    --name {name} \
    --value $(cat {file}) \
    --key-id 344d9fba-07d2-45c8-9bde-2356aaedc6c3
```

*The secret value should first be written to a temporary file to avoid storing the value in shell history.*
