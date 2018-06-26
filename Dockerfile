FROM golang:1.10

RUN apt-get update && \
    apt-get install -y curl git zip

RUN curl -sL https://deb.nodesource.com/setup_9.x | bash - && \
    apt-get install -y nodejs

# Go Dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | bash

# Node config and serverless framework
RUN npm -g config set user root
RUN npm install -g serverless
