FROM golang:1.17 as builder

ENV GO111MODULE=on

ARG BUILD
ARG TAG

WORKDIR /go/src/geeo.io/GeeoServer
COPY go.sum go.mod ./
RUN go mod download

COPY . .
RUN echo "Building ${BUILD} : ${TAG}"
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.Build=${BUILD} -X main.Tag=${TAG}" -o GeeoServer

FROM alpine:3.7
# add ca-certificates in case you need them
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
# set working directory
WORKDIR /root
# copy the binary from builder
COPY --from=builder /go/src/geeo.io/GeeoServer/GeeoServer .
# run the binary

CMD ["/root/GeeoServer"]
