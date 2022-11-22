FROM golang:1.16.3-alpine3.13

WORKDIR /app
COPY /discord-rss/ .
RUN go mod download

# Discord RSS Set Up | Build Arguments:
ENV BOT_TOKEN="defaultvalue"
ENV RSS_URL="https://fake_url.com"
ENV CHANNEL_ID="12345678910"
ENV TIMER_INT=8
ENV USERNAME=""
ENV PASSWORD=""

RUN go build
CMD ./discord-rss -t $BOT_TOKEN -u $RSS_URL -c $CHANNEL_ID -timer $TIMER_INT -user $USERNAME -pass $PASSWORD