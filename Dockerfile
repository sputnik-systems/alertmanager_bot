FROM ubuntu

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY templates ./templates
COPY alertmanager_bot ./alertmanager_bot

ENTRYPOINT ["./alertmanager_bot"]
