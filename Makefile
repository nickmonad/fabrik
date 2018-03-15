IMAGE = opolis/build:dev
GOPATH = /go/src/github.com/opolis/build

RUN = docker run -it --rm \
	  -v $(HOME)/.aws:/root/.aws \
	  -v $(PWD):$(GOPATH) \
	  -w $(GOPATH) \
	  $(IMAGE)

.PHONY: image
image:
	@docker build -t $(IMAGE) .

.PHONY: build
build:
	@$(RUN) dep ensure
	@$(RUN) env GOOS=linux go build \
		-ldflags="-d -s -w" -a -tags netgo \
		-installsuffix netgo \
		-o bin/build build/main.go

.PHONY: deploy
deploy:
	@$(RUN) serverless deploy

.PHONY: shell
shell:
	@$(RUN) sh
