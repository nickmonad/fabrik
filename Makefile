IMAGE = opolis/build:dev
GOPATH = /go/src/github.com/opolis/build


RUN = docker run -it --rm \
	  -v $(PWD):$(GOPATH) \
	  -w $(GOPATH) \
	  $(IMAGE)

.PHONY: image
image:
	@docker build -t $(IMAGE) .

.PHONY: deps
deps:
	@$(RUN) glide install

.PHONY: build
build:
	@$(RUN) go build -o handler main.go
	@$(RUN) chmod a+x handler
	@$(RUN) zip handler.zip ./handler

.PHONY: shell
shell:
	@$(RUN) sh

.PHONY: clean
clean:
	@rm -f handler
	@rm -f handler.zip
	@rm -rf vendor
