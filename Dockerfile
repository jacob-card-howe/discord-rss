FROM golang:alpine3.17

WORKDIR /app
COPY /discord-rss/ .
RUN go mod download

# Discord RSS Set Up | Build Arguments:
ENV BOT_TOKEN="defaultvalue"
ENV PATH_TO_CSV="feeds.csv"

RUN go build
CMD ./discord-rss -t $BOT_TOKEN -f $PATH_TO_CSV
