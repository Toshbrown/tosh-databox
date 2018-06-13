PACKAGE  = databox
DATABOX_GOPATH="$(shell echo ~/go):$(shell pwd):$(shell echo ${GOPATH})"
.PHONY: all
all: build

.PHONY: deps
deps:
	go get -u github.com/pebbe/zmq4
	go get -u github.com/me-box/goZestClient
	go get -u golang.org/x/net/proxy
	go get -u github.com/docker/go-connections/nat
	go get -u github.com/pkg/errors
	go get -u github.com/docker/docker/api/types
	go get -u github.com/docker/docker/client
	go get -u github.com/toshbrown/lib-go-databox

.PHONY: build
build:
	echo $(DATABOX_GOPATH)
	rm -rf ${GOPATH}/src/github.com/docker/docker/vendor/github.com/docker/go-connections > /dev/null
	@GOPATH=$(DATABOX_GOPATH) go build -o bin/$(PACKAGE) main.go
	docker build -t go-container-manager:0.4.0 .
	docker tag go-container-manager:0.4.0 go-container-manager:latest

.PHONY: build-cm
build-cm:
	docker build -t go-container-manager:0.4.0 .
	docker tag go-container-manager:0.4.0 go-container-manager:latest

.PHONY: build-cm-no-cache
build-cm-no-cache:
	docker build -t go-container-manager:0.4.0 . --no-cache
	docker tag go-container-manager:0.4.0 go-container-manager:latest

.PHONY: build-cmd
build-cmd:
	@GOPATH=$(DATABOX_GOPATH) go build -o bin/$(PACKAGE) main.go

.PHONY: start
start:
	#TODO runing latest for now so that we can use core-store with zest 0.0.7
	bin/databox start --release latest -v

.PHONY: startlatest
startlatest:
	bin/databox start --release latest

.PHONY: startflushslas
startflushslas:
	bin/databox start --flushSLAs true

.PHONY: stop
stop:
	bin/databox stop

.PHONY: logs
logs:
	bin/databox logs

.PHONY: test
test:
	@GOPATH=$(DATABOX_GOPATH) ./databox-test
