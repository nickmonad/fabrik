build
-----

## Configure

`~/.aws/credentials`

```
[opolis]
aws_access_key_id = AKIA...
aws_secret_access_key = O4vew...
region = us-west-2
output = json
```

## Deploy Lambda Bucket

```
aws --profile opolis \
    cloudformation create-stack \
    --stack-name opolis-build-lambda \
    --template-body file://deploy.json
```
