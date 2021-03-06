FROM golang:1.16.3-alpine3.13

WORKDIR /app
COPY /discord-rss/ .
RUN go mod download
ENV BOT_TOKEN="defaultvalue"
ENV RSS_URL="https://fake_url.com"
ENV CHANNEL_ID="12345678910"
ENV TIMER_INT=8
RUN go build
CMD ./discord-rss -t $BOT_TOKEN -u $RSS_URL -c $CHANNEL_ID -timer $TIMER_INT