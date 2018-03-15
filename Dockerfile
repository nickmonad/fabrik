FROM golang:1.10-alpine

RUN apk update && \
    apk add curl git nodejs zip

# Go Dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# Node config and serverless framework
RUN npm -g config set user root
RUN npm install -g serverless
