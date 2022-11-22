FROM golang:1.16.3-alpine3.13

WORKDIR /app
COPY /discord-rss/ .
RUN go mod download

# Discord RSS Set Up | Build Arguments:
ARG bot_token
ARG rss_url
ARG channel_id
ARG timer_int=8
ARG username
ARG password

RUN go build
CMD ./discord-rss -t $bot_token -u $rss_url -c $channel_id -timer $timer_int -user $username -pass $password