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
