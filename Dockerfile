FROM golang:1.13.8-alpine3.11

WORKDIR /go/github.com/h3poteto/slack-rage

ADD . ./

RUN set -ex && \
    go mod download && \
    go build

CMD ["./slack-rage", "rtm"]
