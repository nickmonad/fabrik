FROM golang:1.10-alpine

RUN apk update && \
    apk add curl git nodejs zip

# Go
RUN curl https://glide.sh/get | sh

# Node
RUN npm -g config set user root
RUN npm install -g serverless
