
FROM golang:1.17.3-alpine3.14 AS builder

RUN apk add --update --no-cache make

WORKDIR /go/src/fleet-controller

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 make build

#####

FROM alpine:3.14

COPY --from=builder /go/src/fleet-controller/fleet-controller /fleet-controller

ENTRYPOINT [ "/fleet-controller" ]
