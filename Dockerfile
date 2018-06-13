FROM golang:1.10.3-alpine as gobuild
WORKDIR /
ENV GOPATH="/go"
RUN apk update && apk add pkgconfig build-base bash autoconf automake libtool gettext openrc git libzmq zeromq-dev mercurial
#COPY . . if you update the libs below build with --no-cache
RUN go get -u github.com/gorilla/mux
RUN go get -u golang.org/x/net/websocket
RUN go get -u github.com/pebbe/zmq4
RUN go get -u github.com/me-box/lib-go-databox
RUN go get -u github.com/me-box/goZestClient
RUN go get -u github.com/docker/docker/client
RUN go get -u github.com/docker/docker/api/types
RUN go get -u github.com/pkg/errors
RUN go get -u github.com/skip2/go-qrcode
RUN go get -u github.com/docker/go-connections
RUN rm -rf /go/src/github.com/docker/docker/vendor/github.com/docker/go-connections
ENV GOPATH="/go:/go/src/github.com/toshbrown/tosh-databox/"
RUN go env
COPY . /go/src/github.com/toshbrown/tosh-databox/
RUN addgroup -S databox && adduser -S -g databox databox
RUN GGO_ENABLED=0 GOOS=linux go build -a -tags netgo -installsuffix netgo -ldflags '-s -w' -o app /go/src/github.com/toshbrown/tosh-databox/src/containerManager/app.go

FROM alpine
COPY --from=gobuild /etc/passwd /etc/passwd
RUN apk update && apk add libzmq
#TODO security
USER root
#TODO security
WORKDIR /
COPY --from=gobuild /app .
COPY --from=gobuild /go/src/github.com/toshbrown/tosh-databox/src/containerManager/www /www
LABEL databox.type="container-manager"
EXPOSE 80 443
RUN rm -rf /certs/*
CMD ["./app"]
#CMD ["sleep","9999999"]