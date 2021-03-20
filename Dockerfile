FROM golang:1.16.0-buster as build

WORKDIR /app
COPY . .
RUN go get -v ./...
RUN go build -o ./alertmanager_bot ./cmd/bot


FROM ubuntu

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /app/alertmanager_bot /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/alertmanager_bot"]
