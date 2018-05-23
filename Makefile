PACKAGE  = databox
DATABOX_GOPATH="${GOPATH}":$(shell pwd)

.PHONY: all
all: build

.PHONY: build
build:
	docker build -t go-container-manager .
	@GOPATH=$(DATABOX_GOPATH) go build -o bin/$(PACKAGE) main.go

.PHONY: build-cm
build-cm:
	docker build -t go-container-manager .

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
	./databox-test
