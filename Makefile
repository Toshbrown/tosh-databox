PACKAGE  = databox
DATABOX_GOPATH="$(shell echo ~/go):$(shell pwd)"

.PHONY: all
all: build

.PHONY: build
build:
	docker build -t go-container-manager:0.4.0 .
	@GOPATH=$(DATABOX_GOPATH) go build -o bin/$(PACKAGE) main.go

.PHONY: build-cm
build-cm:
	docker build -t go-container-manager:0.4.0 .

.PHONY: build-cmd
build-cmd:
	@GOPATH=$(DATABOX_GOPATH) go build -o bin/$(PACKAGE) main.go

.PHONY: start
start:
	bin/databox start

.PHONY: stop
stop:
	bin/databox stop

.PHONY: logs
logs:
	bin/databox logs

.PHONY: test
test:
	@GOPATH=$(DATABOX_GOPATH) ./databox-test
