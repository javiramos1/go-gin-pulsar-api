FROM golang:1.17-alpine3.15 AS builder

EXPOSE 8000

RUN apk update && \
    apk add --no-cache --update alpine-sdk git

ARG CGO_ENABLED=0

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=amd64 go build -mod=readonly -o dist/go-gin-pulsar-api -v -ldflags "-w -s" .

FROM  gcr.io/distroless/static

COPY .env /etc/

ADD schema /etc

COPY --from=builder /app/dist/go-gin-pulsar-api /usr/local/bin/go-gin-pulsar-api

CMD ["/usr/local/bin/go-gin-pulsar-api"]
