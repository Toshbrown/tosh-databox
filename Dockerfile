FROM golang:1.8.3-alpine3.6 as gobuild
WORKDIR /
ENV GOPATH="/"
RUN apk update && apk add pkgconfig build-base bash autoconf automake libtool gettext openrc git libzmq zeromq-dev mercurial
#COPY . . if you update the libs below build with --no-cache
RUN go get -u github.com/gorilla/mux
RUN go get -u golang.org/x/net/websocket
RUN git clone https://github.com/moby/moby.git /src/github.com/docker/docker --depth 1
RUN go get -u github.com/pebbe/zmq4
RUN go get -u github.com/toshbrown/lib-go-databox
RUN go get -u github.com/docker/go-connections/nat
RUN go get -u github.com/pkg/errors
RUN rm -rf /src/github.com/docker/docker/vendor/github.com/docker/go-connections
COPY . /
RUN addgroup -S databox && adduser -S -g databox databox
RUN GGO_ENABLED=0 GOOS=linux go build -a -tags netgo -installsuffix netgo -ldflags '-s -w' -o app ./src/containerManager/app.go


FROM alpine
COPY --from=gobuild /etc/passwd /etc/passwd
RUN apk update && apk add libzmq curl
#TODO security
USER root
#TODO security
WORKDIR /
COPY --from=gobuild /app .
COPY --from=gobuild /src/containerManager/www /www
LABEL databox.type="container-manager"
EXPOSE 80 443
RUN rm -rf /certs/*
CMD ["./app"]
#CMD ["sleep","9999999"]