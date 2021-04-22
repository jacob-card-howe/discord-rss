FROM golang:1.16.3-alpine3.13

WORKDIR /app
COPY /discord-rss/ .
RUN go mod download
ENV BOT_TOKEN="defaultvalue"
ENV RSS_URL="https://fake_url.com"
RUN go build
CMD ./discord-rss -t $BOT_TOKEN -u $RSS_URL