FROM golang:1.16.3-alpine3.13

ENV GO111MODULE=on
WORKDIR /app
COPY /discord-rss/ .
RUN go mod download
ENV BOT_TOKEN="defaultvalue"
RUN go build
RUN ./discord-rss -t $BOT_TOKEN