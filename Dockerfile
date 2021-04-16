FROM golang:1.16.0-buster as build

RUN apt update \
    && apt install -y curl patch

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

RUN curl -LO https://patch-diff.githubusercontent.com/raw/go-telegram-bot-api/telegram-bot-api/pull/345.patch \
    && patch -d /go/pkg/mod/github.com/go-telegram-bot-api/telegram-bot-api@v4.6.4+incompatible -f < 345.patch

COPY . .
RUN go build -o ./alertmanager_bot ./cmd/bot


FROM ubuntu

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY templates ./templates
COPY --from=build /app/alertmanager_bot /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/alertmanager_bot"]
