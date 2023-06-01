# discord-rss
A simple Discord Bot that periodically parses provided RSS feeds and posts the latest updates to your favorite Discord Server.

### Publication Status:
[![Build & Publish Docker Image](https://github.com/jacob-card-howe/discord-rss/actions/workflows/publish-docker.yaml/badge.svg)](https://github.com/jacob-card-howe/discord-rss/actions/workflows/publish-docker.yaml)

## Usage
This project is designed to be run either as a Docker container or as a Go Binary. Once the bot is running, you can use the following commands to interact with it from within your Discord Server:

* `!help` - Displays a list of available commands
* `!status` - Displays the current status of the Bot and RSS parser
* `!pause` - Pauses the RSS parser in the channel this command is called
* `!resume` - Resumes the RSS parser in the channel this command is called
* `!add <url> <channel id> <timer> <username> <password>` - Adds a new RSS feed to the parser. For RSS feeds that do not use any basic authentication, you can pass in `""` for the `username` and `password` arguments.
* `!remove` - Removes all RSS feeds from the parser in the channel this command is called

### Running the project via Go Binary
First, you'll need to be using [Go version 1.16](https://golang.org/doc/go1.16) or later.

Kick things off by downloading any dependencies from `go.mod` by entering `go download` into your terminal.

Next, run `go build`

Once you've built the Go Binary (titled `discord-rss`), navigate to the location of the Binary and run it. You'll get an error because you're missing the necessary parameters for the Bot to function.

For use with multiple RSS feeds, you'll need to create a CSV file with the following format. The first line of the CSV serves as the defaults for all RSS feeds should you leave a column blank. For instance, if you want to parse all of your RSS feeds every 60 seconds, you can leave the `timer` column blank for all of your RSS feeds after the first row. 

The CSV below will parse the first two RSS feeds every 60 seconds and send updates to channel `123456789`. The third RSS feed will be parsed every 45 seconds and send updates to channel `987654321:

```csv
,123456789,60,,
https://example.com,,,,
https://example2.com,,,,
https://example3.com,987654321,45,,
```

To use the CSV file, run the following command:

> `./discord-rss -t YOUR_BOT_TOKEN -f YOUR/CSV/PATH`

Alternatively, if you only want to make use of a single RSS feed in a single channel, you can run the following command:

> `./discord-rss -t YOUR_BOT_TOKEN -u YOUR_RSS_FEED -c YOUR_DISCORD_CHANNEL_ID -i YOUR_TIMER_INT -user YOUR_RSS_FEED_USERNAME -pass YOUR_RSS_FEED_PASSWORD`

### Running the project via Docker
To run `discord-rss` on Docker, you'll need to leverage a CSV file (`feeds.csv`) to define your RSS feeds. Start by building the image

> `docker build . -t discord-rss:latest`

Next, run the image 
> `docker run -e BOT_TOKEN=YOUR_BOT_TOKEN -e PATH_TO_CSV=YOUR/CSV/PATH discord-rss`

## But what about Discord?
To generate a Bot Token, you'll need to go to the [Discord Developer Portal](https://discord.com/developers/applications/). [This article](https://www.freecodecamp.org/news/create-a-discord-bot-with-python/) by [freecodecamp.org](https://www.freecodecamp.org) does a great job of going through the steps / permissions you'll need for a simple Discord Bot.

To get your `CHANNEL_ID`, you'll need to enable developer mode on your Discord Client. [This Support Documentation by Discord](https://support.discord.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-) does a really good walkthrough of how to set that up.

## Additional Documentation / References
* [mmcdole/gofeed](https://github.com/mmcdole/gofeed)
* [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo)
* [Discord Support](https://support.discord.com/hc/en-us)
* [Discord API Documentation](https://discord.com/developers/docs/intro)
